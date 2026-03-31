import base64
import logging
import mimetypes
import os
import re
import sys
import tempfile
from concurrent import futures

import grpc

# Add proto to path
sys.path.insert(
    0,
    os.path.join(os.path.dirname(__file__), "../../../proto/python"),
)
import paper_pb2  # noqa: E402
import paper_pb2_grpc  # noqa: E402

from converter import extract_metadata, fix_utf8_mojibake, refine_markdown  # noqa: E402

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)


def embed_images_as_base64(markdown: str) -> str:
    """Replace local image paths in markdown with base64 data URLs.

    MinerU's get_markdown() produces references like ![](/tmp/xxx/image.jpg).
    Since the temp directory is deleted after conversion, these references
    become dead links. This function reads each image file and replaces
    the path with a self-contained data:image/...;base64,... URL.
    """

    def _replace(match: re.Match) -> str:
        alt_text = match.group(1)
        path = match.group(2)
        if path.startswith("data:") or path.startswith("http"):
            return match.group(0)
        if not os.path.isfile(path):
            logger.warning("Image file not found, skipping: %s", path)
            return match.group(0)
        mime_type, _ = mimetypes.guess_type(path)
        if mime_type is None:
            mime_type = "image/jpeg"
        try:
            with open(path, "rb") as f:
                b64 = base64.b64encode(f.read()).decode("ascii")
            return f"![{alt_text}](data:{mime_type};base64,{b64})"
        except Exception as e:
            logger.warning("Failed to embed image %s: %s", path, e)
            return match.group(0)

    return re.sub(r"!\[([^\]]*)\]\(([^)\s]+)\)", _replace, markdown)


def extract_and_placeholder_images(markdown: str) -> tuple[str, list[str]]:
    """Replace local image paths with placeholders and return base64 data list.

    This protects image data from being processed/corrupted by the LLM.
    Returns (markdown_with_placeholders, [base64_data_url, ...]).
    """

    def _read_image(path: str) -> str:
        if not os.path.isfile(path):
            return ""
        mime_type, _ = mimetypes.guess_type(path)
        if mime_type is None:
            mime_type = "image/jpeg"
        try:
            with open(path, "rb") as f:
                b64 = base64.b64encode(f.read()).decode("ascii")
            return f"data:{mime_type};base64,{b64}"
        except Exception as e:
            logger.warning("Failed to read image %s: %s", path, e)
            return ""

    images: list[str] = []

    def _replace(match: re.Match) -> str:
        alt_text = match.group(1)
        path = match.group(2)
        if path.startswith("data:") or path.startswith("http"):
            return match.group(0)
        data_url = _read_image(path)
        if not data_url:
            return match.group(0)
        idx = len(images)
        images.append(data_url)
        return f"![{alt_text}](<!--IMG_{idx}-->)"

    result = re.sub(r"!\[([^\]]*)\]\(([^)\s]+)\)", _replace, markdown)
    return result, images


def restore_image_placeholders(markdown: str, images: list[str]) -> str:
    """Restore image placeholders with actual base64 data URLs."""
    for idx, data_url in enumerate(images):
        markdown = markdown.replace(f"(<!--IMG_{idx}-->)", f"({data_url})")
    return markdown


class PaperConverterServicer(paper_pb2_grpc.PaperConverterServicer):
    def Convert(self, request, context):
        """Convert PDF to Markdown with streaming progress."""
        logger.info(
            "Converting PDF: %s (%d bytes)",
            request.filename,
            len(request.pdf_content),
        )

        # Phase 1: Mining (MinerU)
        yield paper_pb2.ConvertProgress(status="mining", progress=10)
        images: list[str] = []
        try:
            from magic_pdf.data.data_reader_writer.filebase import (
                FileBasedDataWriter,
            )
            from magic_pdf.data.dataset import PymuDocDataset
            from magic_pdf.model.doc_analyze_by_custom_model import doc_analyze

            ds = PymuDocDataset(request.pdf_content)
            if ds.classify() == "ocr":
                infer_result = ds.apply(doc_analyze, ocr=True)
            else:
                infer_result = ds.apply(doc_analyze, ocr=False)

            # InferenceResult → pipe mode → PipeResult → get_markdown
            with tempfile.TemporaryDirectory() as tmp_dir:
                image_writer = FileBasedDataWriter(tmp_dir)
                if ds.classify() == "ocr":
                    pipe_result = infer_result.pipe_ocr_mode(image_writer)
                else:
                    pipe_result = infer_result.pipe_txt_mode(image_writer)
                markdown = pipe_result.get_markdown(tmp_dir)
                # Read images into memory and replace with placeholders
                # before temp dir is deleted
                markdown, images = extract_and_placeholder_images(markdown)
                logger.info(
                    "Extracted %d images from PDF",
                    len(images),
                )

            yield paper_pb2.ConvertProgress(status="mining", progress=50)

            # Fix UTF-8 mojibake before LLM processing
            markdown = fix_utf8_mojibake(markdown)

        except Exception as e:
            logger.error("MinerU conversion failed: %s", e)
            yield paper_pb2.ConvertProgress(
                status="failed",
                progress=0,
                error=f"PDF mining failed: {e}",
            )
            return

        # Phase 2: LLM refinement (markdown has image placeholders, not base64)
        yield paper_pb2.ConvertProgress(status="llm_refining", progress=70)
        markdown = refine_markdown(markdown)
        yield paper_pb2.ConvertProgress(status="llm_refining", progress=90)

        # Restore base64 image data after LLM processing
        if images:
            markdown = restore_image_placeholders(markdown, images)

        # Phase 3: Complete
        yield paper_pb2.ConvertProgress(
            status="completed",
            progress=100,
            markdown=markdown,
        )
        logger.info("Conversion completed for: %s", request.filename)

    def ExtractMetadata(self, request, context):
        """Extract metadata from markdown content."""
        logger.info(
            "Extracting metadata from markdown (%d chars)",
            len(request.markdown),
        )

        result = extract_metadata(request.markdown)

        return paper_pb2.PaperMetadata(
            title=result.title,
            authors=result.authors,
            abstract=result.abstract,
            keywords=result.keywords,
            published_year=result.published_year,
            doi=result.doi,
        )


def serve():
    port = os.getenv("GRPC_PORT", "50051")
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    paper_pb2_grpc.add_PaperConverterServicer_to_server(
        PaperConverterServicer(),
        server,
    )
    server.add_insecure_port(f"[::]:{port}")
    server.start()
    logger.info("Paper converter gRPC server started on port %s", port)
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
