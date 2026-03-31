import json
import os
import sys
import tempfile
import unittest
from unittest.mock import MagicMock

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))  # noqa: E402

# Mock openai before importing converter (not installed in test env)
sys.modules["openai"] = MagicMock()
# Mock grpc and proto modules before importing server
sys.modules["grpc"] = MagicMock()
sys.modules["paper_pb2"] = MagicMock()
sys.modules["paper_pb2_grpc"] = MagicMock()

from converter import (  # noqa: E402
    extract_metadata_prompt,
    fix_utf8_mojibake,
    parse_metadata_response,
)
from server import (  # noqa: E402
    embed_images_as_base64,
    extract_and_placeholder_images,
    restore_image_placeholders,
)


class TestConverter(unittest.TestCase):
    def test_extract_metadata_prompt_contains_required_fields(self):
        prompt = extract_metadata_prompt("# Test Paper\n\nSome content")
        self.assertIn("title", prompt.lower())
        self.assertIn("authors", prompt.lower())
        self.assertIn("abstract", prompt.lower())
        self.assertIn("keywords", prompt.lower())

    def test_parse_metadata_response_valid_json(self):
        response = json.dumps(
            {
                "title": "Test Paper",
                "authors": ["Alice", "Bob"],
                "abstract": "This is abstract",
                "keywords": ["ML", "AI"],
                "published_year": "2024",
                "doi": "10.1234/test",
            }
        )
        result = parse_metadata_response(response)
        self.assertEqual(result.title, "Test Paper")
        self.assertEqual(result.authors, ["Alice", "Bob"])
        self.assertEqual(result.published_year, "2024")

    def test_parse_metadata_response_partial(self):
        response = json.dumps({"title": "Only Title"})
        result = parse_metadata_response(response)
        self.assertEqual(result.title, "Only Title")
        self.assertEqual(result.authors, [])
        self.assertEqual(result.keywords, [])

    def test_parse_metadata_response_invalid_json(self):
        result = parse_metadata_response("not json at all")
        self.assertIsNone(result)


class TestFixUtf8Mojibake(unittest.TestCase):
    """Test UTF-8 mojibake repair for MinerU-extracted text."""

    def test_fixes_en_dash(self):
        """â€" → – (en-dash U+2013)."""
        broken = "1.6\u00e2\u20ac\u201c72\u00e2\u20ac\u201c"
        result = fix_utf8_mojibake(broken)
        self.assertEqual(result, "1.6\u201372\u2013")

    def test_fixes_em_dash(self):
        """â€" → — (em-dash U+2014)."""
        broken = "text\u00e2\u20ac\u201dmore"
        result = fix_utf8_mojibake(broken)
        self.assertEqual(result, "text\u2014more")

    def test_fixes_left_double_quote(self):
        """â€œ → " (U+201C)."""
        broken = "\u00e2\u20ac\u0153hello\u00e2\u20ac\u2122"
        result = fix_utf8_mojibake(broken)
        self.assertEqual(result, "\u201chello\u2019")

    def test_fixes_ellipsis(self):
        """â€¦ → … (U+2026)."""
        broken = "and\u00e2\u20ac\u00a6 so on"
        result = fix_utf8_mojibake(broken)
        self.assertEqual(result, "and\u2026 so on")

    def test_preserves_clean_text(self):
        """Clean text should pass through unchanged."""
        clean = "Hello, world! This is fine. 123"
        result = fix_utf8_mojibake(clean)
        self.assertEqual(result, clean)

    def test_preserves_normal_unicode(self):
        """Non-mojibake Unicode (CJK, emoji) should be preserved."""
        text = "中文测试 αβγ ∑∫ ½"
        result = fix_utf8_mojibake(text)
        self.assertEqual(result, text)

    def test_multiple_replacements_in_formula(self):
        """Display formula with mojibake should be fully fixed."""
        broken = "$L = \\frac{buffer}{bandwidth \\times one-hop \u00e2\u20ac\u201c delay \\times 2}$"
        result = fix_utf8_mojibake(broken)
        self.assertIn("\u2013", result)  # en-dash present
        self.assertNotIn("\u00e2\u20ac\u201c", result)  # mojibake removed


class TestImageEmbedding(unittest.TestCase):
    """Test image base64 embedding functions for PDF-to-Markdown conversion."""

    def _create_test_image(self, tmp_dir: str, name: str = "test.jpg") -> str:
        """Create a minimal valid JPEG-like file for testing."""
        path = os.path.join(tmp_dir, name)
        # Minimal JPEG header (not valid image, but sufficient for testing)
        with open(path, "wb") as f:
            f.write(b"\xff\xd8\xff\xe0" + b"test image data" * 10)
        return path

    def test_embed_images_local_path(self):
        """Local file paths should be replaced with base64 data URLs."""
        with tempfile.TemporaryDirectory() as tmp_dir:
            img_path = self._create_test_image(tmp_dir)
            md = f"See figure: ![]({img_path})"
            result = embed_images_as_base64(md)
            self.assertIn("data:image/jpeg;base64,", result)
            self.assertNotIn(img_path, result)

    def test_embed_images_preserves_http_urls(self):
        """HTTP URLs should be left unchanged."""
        md = "![alt](https://example.com/image.png)"
        result = embed_images_as_base64(md)
        self.assertEqual(result, md)

    def test_embed_images_preserves_data_urls(self):
        """Existing data URLs should be left unchanged."""
        md = "![alt](data:image/png;base64,abc123)"
        result = embed_images_as_base64(md)
        self.assertEqual(result, md)

    def test_embed_images_missing_file_keeps_reference(self):
        """References to non-existent files should be preserved."""
        md = "![alt](/nonexistent/path/image.jpg)"
        result = embed_images_as_base64(md)
        self.assertEqual(result, md)

    def test_extract_and_placeholder_images(self):
        """Should extract images and replace with placeholders."""
        with tempfile.TemporaryDirectory() as tmp_dir:
            img_path = self._create_test_image(tmp_dir, "fig1.jpg")
            md = f"Before ![]({img_path}) After"
            result_md, images = extract_and_placeholder_images(md)
            self.assertEqual(len(images), 1)
            self.assertIn("<!--IMG_0-->", result_md)
            self.assertNotIn(img_path, result_md)
            self.assertTrue(images[0].startswith("data:image/jpeg;base64,"))

    def test_extract_multiple_images(self):
        """Should handle multiple images with sequential placeholders."""
        with tempfile.TemporaryDirectory() as tmp_dir:
            img1 = self._create_test_image(tmp_dir, "fig1.jpg")
            img2 = self._create_test_image(tmp_dir, "fig2.jpg")
            md = f"![]({img1}) and ![]({img2})"
            result_md, images = extract_and_placeholder_images(md)
            self.assertEqual(len(images), 2)
            self.assertIn("<!--IMG_0-->", result_md)
            self.assertIn("<!--IMG_1-->", result_md)

    def test_roundtrip_placeholder_restore(self):
        """Full roundtrip: extract → placeholder → restore should match embed."""
        with tempfile.TemporaryDirectory() as tmp_dir:
            img_path = self._create_test_image(tmp_dir, "fig.jpg")
            original = f"Text ![]({img_path}) more text"
            result_md, images = extract_and_placeholder_images(original)
            restored = restore_image_placeholders(result_md, images)
            # Should match direct embedding
            direct = embed_images_as_base64(original)
            self.assertEqual(restored, direct)


if __name__ == "__main__":
    unittest.main()
