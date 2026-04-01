import json
import os
import sys
import unittest
from unittest.mock import MagicMock

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))  # noqa: E402

# Save real modules before mocking (to avoid polluting other test files)
_saved_modules = {
    "openai": sys.modules.get("openai"),
    "paper_pb2": sys.modules.get("paper_pb2"),
    "paper_pb2_grpc": sys.modules.get("paper_pb2_grpc"),
}

# Mock openai before importing converter (not installed in test env)
sys.modules["openai"] = MagicMock()
# Mock proto modules (generated code, not available in test env)
sys.modules["paper_pb2"] = MagicMock()
sys.modules["paper_pb2_grpc"] = MagicMock()

from converter import (  # noqa: E402
    extract_metadata_prompt,
    fix_utf8_mojibake,
    images_to_base64,
    parse_metadata_response,
    sanitize_markdown,
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
    """Test UTF-8 mojibake repair for PDF-extracted text."""

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


class TestSanitizeMarkdown(unittest.TestCase):
    """Test HTML tag cleanup from Marker output."""

    def test_removes_empty_page_anchor_spans(self):
        """<span id="page-X-Y"></span> should be removed entirely."""
        md = 'Some text<span id="page-13-21"></span> more text'
        result = sanitize_markdown(md)
        self.assertEqual(result, "Some text more text")

    def test_removes_page_anchor_with_whitespace(self):
        """Page anchors with internal whitespace should be removed."""
        md = '<span id="page-0-0">  </span>Content'
        result = sanitize_markdown(md)
        self.assertEqual(result, "Content")

    def test_unwraps_sup_tags(self):
        """<sup>1</sup> should become plain text 1."""
        md = "RDMA<sup>1</sup> is a technology"
        result = sanitize_markdown(md)
        self.assertEqual(result, "RDMA1 is a technology")

    def test_unwraps_sub_tags(self):
        """<sub>2</sub> should become plain text 2."""
        md = "H<sub>2</sub>O is water"
        result = sanitize_markdown(md)
        self.assertEqual(result, "H2O is water")

    def test_converts_br_to_newline(self):
        """<br> should become newline."""
        md = "Line one<br>Line two"
        result = sanitize_markdown(md)
        self.assertEqual(result, "Line one\nLine two")

    def test_converts_br_self_closing(self):
        """<br/> and <br /> should also become newline."""
        md = "Line one<br/>Line two<br />Line three"
        result = sanitize_markdown(md)
        self.assertEqual(result, "Line one\nLine two\nLine three")

    def test_removes_remaining_span_tags(self):
        """Non-anchor span tags should be stripped, content preserved."""
        md = '<span class="bold">Important</span> point'
        result = sanitize_markdown(md)
        self.assertEqual(result, "Important point")

    def test_combined_html_cleanup(self):
        """Real-world Marker output with multiple HTML tags."""
        md = (
            "# Introduction\n"
            '<span id="page-1-0"></span>\n'
            "RDMA<sup>1</sup> allows H<sub>2</sub>O<br>\n"
            'Next line<span class="foo">styled</span>.'
        )
        result = sanitize_markdown(md)
        self.assertNotIn("<span", result)
        self.assertNotIn("</span>", result)
        self.assertNotIn("<sup>", result)
        self.assertNotIn("<br>", result)
        self.assertIn("RDMA1", result)
        self.assertIn("H2O", result)
        self.assertIn("\n", result)

    def test_preserves_plain_markdown(self):
        """Clean Markdown should pass through unchanged."""
        md = "# Title\n\nParagraph with **bold** and *italic*.\n\n- List item"
        result = sanitize_markdown(md)
        self.assertEqual(result, md)


class TestImagesToBase64(unittest.TestCase):
    """Test Marker image reference to base64 data URL conversion."""

    def test_replaces_local_image_references(self):
        """Image filenames should be replaced with base64 data URLs."""
        md = "See figure: ![](_page_0_Figure_0.jpeg)"
        images = [("_page_0_Figure_0.jpeg", "data:image/jpeg;base64,abc123")]
        result = images_to_base64(md, images)
        self.assertIn("data:image/jpeg;base64,abc123", result)
        self.assertNotIn("_page_0_Figure_0.jpeg", result)

    def test_preserves_text_without_images(self):
        """Markdown without images should pass through unchanged."""
        md = "Just plain text with $formula$"
        result = images_to_base64(md, [])
        self.assertEqual(result, md)

    def test_handles_multiple_images(self):
        """Should replace all image references."""
        md = "![fig1](_page_0_Figure_0.jpeg) and ![fig2](_page_0_Figure_1.jpeg)"
        images = [
            ("_page_0_Figure_0.jpeg", "data:image/jpeg;base64,aaa"),
            ("_page_0_Figure_1.jpeg", "data:image/png;base64,bbb"),
        ]
        result = images_to_base64(md, images)
        self.assertIn("data:image/jpeg;base64,aaa", result)
        self.assertIn("data:image/png;base64,bbb", result)

    def test_preserves_alt_text(self):
        """Should preserve alt text in image references."""
        md = "![Architecture Diagram](_page_5_Figure_2.jpeg)"
        images = [("_page_5_Figure_2.jpeg", "data:image/jpeg;base64,xyz")]
        result = images_to_base64(md, images)
        self.assertIn("[Architecture Diagram]", result)


def teardown_module():
    """Restore real modules after all tests in this file complete."""
    for name, saved in _saved_modules.items():
        if saved is not None:
            sys.modules[name] = saved
        elif name in sys.modules:
            del sys.modules[name]


if __name__ == "__main__":
    unittest.main()
