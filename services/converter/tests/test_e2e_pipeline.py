"""
E2E Pipeline Test for the Paper Converter Service.

Tests the full conversion pipeline:
  1. PDF input -> MinerU extraction -> raw markdown + images
  2. UTF-8 mojibake fix
  3. LLM markdown refinement (if LLM_API_KEY is set)
  4. Image embedding (base64 data URLs)
  5. Metadata extraction (if LLM_API_KEY is set)

Prerequisites:
  - MinerU (magic-pdf) installed
  - Test PDF at the expected path
  - Optional: LLM_API_KEY set for LLM refinement tests

Run:
  cd services/converter
  uv run pytest tests/test_e2e_pipeline.py -v -s
"""

import os
import sys
import tempfile
import unittest

# Add src to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))  # noqa: E402

# Test PDF path (project-root relative)
TEST_PDF_PATH = os.path.join(
    os.path.dirname(__file__),
    "..",
    "..",
    "uploads",
    "papers",
    "019d2fae-69a3-73f4-90ea-ce15ee1db999",
    "2026-03-28",
    "rdma.pdf",
)


def _pdf_exists() -> bool:
    return os.path.isfile(TEST_PDF_PATH)


def _mineru_available() -> bool:
    try:
        from magic_pdf.data.dataset import PymuDocDataset  # noqa: F401

        return True
    except ImportError:
        return False


def _llm_available() -> bool:
    return bool(os.getenv("LLM_API_KEY"))


@unittest.skipUnless(_pdf_exists(), "Test PDF not found")
class TestMinerUExtraction(unittest.TestCase):
    """Test MinerU PDF-to-Markdown extraction."""

    def test_extract_markdown_from_pdf(self):
        """MinerU should extract markdown content from the RDMA paper."""
        from magic_pdf.data.data_reader_writer.filebase import FileBasedDataWriter
        from magic_pdf.data.dataset import PymuDocDataset
        from magic_pdf.model.doc_analyze_by_custom_model import doc_analyze

        with open(TEST_PDF_PATH, "rb") as f:
            pdf_bytes = f.read()

        self.assertGreater(len(pdf_bytes), 1000, "PDF should be non-trivial")

        ds = PymuDocDataset(pdf_bytes)
        if ds.classify() == "ocr":
            infer_result = ds.apply(doc_analyze, ocr=True)
        else:
            infer_result = ds.apply(doc_analyze, ocr=False)

        with tempfile.TemporaryDirectory() as tmp_dir:
            image_writer = FileBasedDataWriter(tmp_dir)
            if ds.classify() == "ocr":
                pipe_result = infer_result.pipe_ocr_mode(image_writer)
            else:
                pipe_result = infer_result.pipe_txt_mode(image_writer)
            markdown = pipe_result.get_markdown(tmp_dir)

        # Basic assertions
        self.assertIsInstance(markdown, str)
        self.assertGreater(len(markdown), 500, "Extracted markdown should be substantial")

    @unittest.skipUnless(_mineru_available(), "MinerU not installed")
    def test_extraction_contains_text_structure(self):
        """Extracted markdown should have headings or paragraphs."""
        from magic_pdf.data.data_reader_writer.filebase import FileBasedDataWriter
        from magic_pdf.data.dataset import PymuDocDataset
        from magic_pdf.model.doc_analyze_by_custom_model import doc_analyze

        with open(TEST_PDF_PATH, "rb") as f:
            pdf_bytes = f.read()

        ds = PymuDocDataset(pdf_bytes)
        if ds.classify() == "ocr":
            infer_result = ds.apply(doc_analyze, ocr=True)
        else:
            infer_result = ds.apply(doc_analyze, ocr=False)

        with tempfile.TemporaryDirectory() as tmp_dir:
            image_writer = FileBasedDataWriter(tmp_dir)
            if ds.classify() == "ocr":
                pipe_result = infer_result.pipe_ocr_mode(image_writer)
            else:
                pipe_result = infer_result.pipe_txt_mode(image_writer)
            markdown = pipe_result.get_markdown(tmp_dir)

        # Academic paper should have some structure
        self.assertTrue(
            "#" in markdown or "\n\n" in markdown,
            "Markdown should contain headings or paragraphs",
        )


@unittest.skipUnless(_pdf_exists(), "Test PDF not found")
class TestMojibakeFix(unittest.TestCase):
    """Test that UTF-8 mojibake is fixed during conversion."""

    def test_fix_utf8_mojibake_on_extracted_text(self):
        """The mojibake fixer should clean up MinerU output."""
        from converter import fix_utf8_mojibake

        # Simulate typical MinerU mojibake patterns
        broken = "range 1.6\u00e2\u20ac\u201c72\u00e2\u20ac\u201c meters"
        fixed = fix_utf8_mojibake(broken)

        # Should have proper en-dash
        self.assertIn("\u2013", fixed)
        self.assertNotIn("\u00e2\u20ac\u201c", fixed)

    def test_mojibake_fix_preserves_clean_text(self):
        """Clean text should pass through unchanged."""
        from converter import fix_utf8_mojibake

        clean = "Hello world, this is fine text with CJK: \u4e2d\u6587"
        result = fix_utf8_mojibake(clean)
        self.assertEqual(result, clean)


@unittest.skipUnless(_pdf_exists(), "Test PDF not found")
class TestImageEmbedding(unittest.TestCase):
    """Test image embedding functions with real MinerU output."""

    def test_embed_images_with_real_pdf_images(self):
        """Image embedding should handle image references from MinerU extraction."""
        from server import embed_images_as_base64

        # Create a temporary image to simulate MinerU output
        with tempfile.TemporaryDirectory() as tmp_dir:
            img_path = os.path.join(tmp_dir, "fig1.jpg")
            with open(img_path, "wb") as f:
                f.write(b"\xff\xd8\xff\xe0" + b"test image data" * 100)

            md = f"See figure: ![]({img_path})"
            result = embed_images_as_base64(md)

            self.assertIn("data:image/jpeg;base64,", result)
            self.assertNotIn(img_path, result)

    def test_placeholder_roundtrip(self):
        """Full roundtrip: extract -> placeholder -> restore should be consistent."""
        from server import embed_images_as_base64, extract_and_placeholder_images, restore_image_placeholders

        with tempfile.TemporaryDirectory() as tmp_dir:
            img_path = os.path.join(tmp_dir, "test.jpg")
            with open(img_path, "wb") as f:
                f.write(b"\xff\xd8\xff\xe0" + b"test image data" * 100)

            original = f"Before ![]({img_path}) After"
            result_md, images = extract_and_placeholder_images(original)
            restored = restore_image_placeholders(result_md, images)
            direct = embed_images_as_base64(original)

            self.assertEqual(restored, direct)
            self.assertIn("data:image/jpeg;base64,", restored)


@unittest.skipUnless(_pdf_exists(), "Test PDF not found")
@unittest.skipUnless(_llm_available(), "LLM_API_KEY not set")
class TestLLMRefinement(unittest.TestCase):
    """Test LLM-based markdown refinement (requires API key)."""

    def test_refine_markdown_returns_string(self):
        """LLM refinement should return a non-empty string."""
        from converter import refine_markdown

        sample_md = """
        # Test Paper

        This is a test paper with some formulas like E=mc2 and tables.

        | Col1 | Col2 |
        |------|------|
        | A    | B    |

        Some more text with special characters: alpha, beta, gamma.
        """

        result = refine_markdown(sample_md)
        self.assertIsInstance(result, str)
        self.assertGreater(len(result), 50)

    def test_extract_metadata_from_markdown(self):
        """LLM should extract metadata from markdown content."""
        from converter import extract_metadata

        sample_md = """
        # RDMA over Converged Ethernet: A Comprehensive Study

        Authors: John Smith, Jane Doe

        Abstract: This paper presents a comprehensive study of RDMA technology.

        Keywords: RDMA, networking, high-performance computing
        """

        result = extract_metadata(sample_md)
        self.assertIsNotNone(result)
        # Title should be extracted
        self.assertTrue(
            len(result.title) > 0,
            "LLM should extract a title from the markdown",
        )


@unittest.skipUnless(_pdf_exists(), "Test PDF not found")
class TestFullPipeline(unittest.TestCase):
    """
    Full E2E pipeline test: PDF -> MinerU extraction -> mojibake fix -> image embed.

    This tests the core conversion pipeline without requiring the gRPC server.
    """

    def _extract_markdown(self):
        """Helper: extract markdown from the test PDF using MinerU."""
        from magic_pdf.data.data_reader_writer.filebase import FileBasedDataWriter
        from magic_pdf.data.dataset import PymuDocDataset
        from magic_pdf.model.doc_analyze_by_custom_model import doc_analyze

        with open(TEST_PDF_PATH, "rb") as f:
            pdf_bytes = f.read()

        ds = PymuDocDataset(pdf_bytes)
        if ds.classify() == "ocr":
            infer_result = ds.apply(doc_analyze, ocr=True)
        else:
            infer_result = ds.apply(doc_analyze, ocr=False)

        with tempfile.TemporaryDirectory() as tmp_dir:
            image_writer = FileBasedDataWriter(tmp_dir)
            if ds.classify() == "ocr":
                pipe_result = infer_result.pipe_ocr_mode(image_writer)
            else:
                pipe_result = infer_result.pipe_txt_mode(image_writer)
            markdown = pipe_result.get_markdown(tmp_dir)
            # Also extract images
            from server import embed_images_as_base64

            markdown_with_images = embed_images_as_base64(markdown)

        return markdown, markdown_with_images

    @unittest.skipUnless(_mineru_available(), "MinerU not installed")
    def test_pipeline_produces_valid_markdown(self):
        """Full pipeline should produce valid markdown content."""
        raw_md, final_md = self._extract_markdown()

        # Apply mojibake fix
        from converter import fix_utf8_mojibake

        fixed_md = fix_utf8_mojibake(final_md)

        # Basic quality checks
        self.assertGreater(len(fixed_md), 500, "Final markdown should be substantial")
        self.assertIsInstance(fixed_md, str)

        # Should not contain mojibake
        mojibake_patterns = [
            "\u00e2\u20ac\u201c",  # should be en-dash
            "\u00e2\u20ac\u201d",  # should be em-dash
            "\u00e2\u20ac\u0153",  # should be left double quote
            "\u00e2\u20ac\u2122",  # should be right single quote
            "\u00e2\u20ac\u00a6",  # should be ellipsis
        ]
        for broken in mojibake_patterns:
            self.assertNotIn(
                broken,
                fixed_md,
                f"Pipeline output should not contain mojibake: {broken!r}",
            )

    @unittest.skipUnless(_mineru_available(), "MinerU not installed")
    def test_pipeline_preserves_images(self):
        """Pipeline should preserve images as base64 data URLs."""
        _raw_md, final_md = self._extract_markdown()

        from converter import fix_utf8_mojibake

        fixed_md = fix_utf8_mojibake(final_md)

        # Check for image presence (markdown image syntax or base64 data URLs)
        has_images = "data:image/" in fixed_md or "![" in fixed_md
        self.assertTrue(
            has_images,
            "Pipeline output should contain image references or base64 data URLs",
        )


if __name__ == "__main__":
    unittest.main()
