import os
import sys
from concurrent import futures

import grpc

# Add proto to path
sys.path.insert(
    0,
    os.path.join(os.path.dirname(__file__), "../../../proto/python"),
)
import paper_pb2  # noqa: E402
import paper_pb2_grpc  # noqa: E402

from converter import (  # noqa: E402
    convert_with_marker,
    extract_metadata,
    fix_utf8_mojibake,
    images_to_base64,
)
from interceptor import LoggingInterceptor  # noqa: E402
from logging_config import configure_logging, get_logger  # noqa: E402

logger = get_logger(__name__)


class PaperConverterServicer(paper_pb2_grpc.PaperConverterServicer):
    def Convert(self, request, context):
        """Convert PDF to Markdown using Marker, with streaming progress."""
        logger.info(
            "Converting PDF",
            filename=request.filename,
            size_bytes=len(request.pdf_content),
        )

        # Phase 1: Marker conversion
        yield paper_pb2.ConvertProgress(status="converting", progress=10)
        try:
            markdown, image_data = convert_with_marker(
                request.pdf_content,
                request.filename,
            )
            yield paper_pb2.ConvertProgress(status="converting", progress=70)

            # Fix any residual encoding issues
            markdown = fix_utf8_mojibake(markdown)
            yield paper_pb2.ConvertProgress(status="converting", progress=80)

            logger.info(
                "Marker conversion completed",
                markdown_length=len(markdown),
                image_count=len(image_data),
            )

        except Exception as e:
            logger.error("Marker conversion failed", error=str(e))
            yield paper_pb2.ConvertProgress(
                status="failed",
                progress=0,
                error=f"PDF conversion failed: {e}",
            )
            return

        # Phase 2: Post-processing (image embedding)
        yield paper_pb2.ConvertProgress(status="processing", progress=85)
        if image_data:
            markdown = images_to_base64(markdown, image_data)
            logger.info("Images embedded as base64", image_count=len(image_data))
        yield paper_pb2.ConvertProgress(status="processing", progress=95)

        # Phase 3: Complete
        yield paper_pb2.ConvertProgress(
            status="completed",
            progress=100,
            markdown=markdown,
        )
        logger.info("Conversion completed", filename=request.filename)

    def ExtractMetadata(self, request, context):
        """Extract metadata from markdown content."""
        logger.info(
            "Extracting metadata",
            content_length=len(request.markdown),
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
    configure_logging()
    port = os.getenv("GRPC_PORT", "50051")
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=4),
        interceptors=[LoggingInterceptor()],
    )
    paper_pb2_grpc.add_PaperConverterServicer_to_server(
        PaperConverterServicer(),
        server,
    )
    server.add_insecure_port(f"[::]:{port}")
    server.start()
    logger.info("Server started", port=port)
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
