import logging
import os
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

from converter import extract_metadata, refine_markdown  # noqa: E402

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)


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

            yield paper_pb2.ConvertProgress(status="mining", progress=50)

        except Exception as e:
            logger.error("MinerU conversion failed: %s", e)
            yield paper_pb2.ConvertProgress(
                status="failed",
                progress=0,
                error=f"PDF mining failed: {e}",
            )
            return

        # Phase 2: LLM refinement
        yield paper_pb2.ConvertProgress(status="llm_refining", progress=70)
        markdown = refine_markdown(markdown)
        yield paper_pb2.ConvertProgress(status="llm_refining", progress=90)

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
