import logging
import os
import shutil
import sys
import tempfile
from concurrent import futures

import grpc

# Add proto to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../proto/python"))
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
        logger.info("Converting PDF: %s (%d bytes)", request.filename, len(request.pdf_content))

        # Phase 1: Mining (MinerU)
        yield paper_pb2.ConvertProgress(status="mining", progress=10)
        try:
            # Save PDF to temp file
            with tempfile.NamedTemporaryFile(suffix=".pdf", delete=False) as tmp:
                tmp.write(request.pdf_content)
                tmp_path = tmp.name

            output_dir = tempfile.mkdtemp()
            try:
                from magic_pdf.data.data_reader_writer import (
                    FileBasedDataReader,
                    FileBasedDataWriter,
                )
                from magic_pdf.pipe.UNIPipe import UNIPipe

                # Use MinerU to convert
                reader = FileBasedDataReader("")
                writer = FileBasedDataWriter(output_dir)

                pipe = UNIPipe(tmp_path, "", reader, writer)
                pipe.pipe_classify()
                pipe.pipe_parse()
                pipe.pipe_mk_markdown(output_dir, drop_mode="none")

                # Read generated markdown
                md_path = os.path.join(
                    output_dir,
                    "auto",
                    os.path.splitext(request.filename)[0] + ".md",
                )
                if os.path.exists(md_path):
                    with open(md_path) as f:
                        markdown = f.read()
                else:
                    # Fallback: try to find any .md file
                    md_files = []
                    for root, dirs, files in os.walk(output_dir):
                        for f in files:
                            if f.endswith(".md"):
                                md_files.append(os.path.join(root, f))
                    if md_files:
                        with open(md_files[0]) as f:
                            markdown = f.read()
                    else:
                        markdown = "# Conversion Error\n\nMinerU did not produce markdown output."
            finally:
                # B5: Clean up both temp PDF and output directory
                os.unlink(tmp_path)
                shutil.rmtree(output_dir, ignore_errors=True)

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
        logger.info("Extracting metadata from markdown (%d chars)", len(request.markdown))

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
