"""Tests for Marker-based PDF to Markdown conversion.

Covers:
  - convert_with_marker: PDF bytes → markdown + images
  - images_to_base64: PIL images → base64 data URL replacement
  - pil_image_to_base64: PIL Image → data URL conversion
"""

import base64
import os
import sys
import unittest
from unittest.mock import MagicMock, patch

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))

# Mock proto modules (generated code, not available in test env)
_saved_modules = {
    "paper_pb2": sys.modules.get("paper_pb2"),
    "paper_pb2_grpc": sys.modules.get("paper_pb2_grpc"),
}
sys.modules["paper_pb2"] = MagicMock()
sys.modules["paper_pb2_grpc"] = MagicMock()


class TestConvertWithMarker(unittest.TestCase):
    """Test the convert_with_marker function."""

    @patch("converter._get_marker_converter")
    def test_returns_markdown_and_images(self, mock_get_converter):
        """convert_with_marker should return (markdown, [(name, base64_data), ...])."""
        from converter import convert_with_marker

        # Setup mock converter that returns rendered result
        mock_img = MagicMock()
        mock_rendered = MagicMock()
        mock_rendered.markdown = "# Test Paper\n\nSome content with formula $E=mc^2$"
        mock_rendered.images = {"fig1.jpeg": mock_img}

        mock_converter_instance = MagicMock()
        mock_converter_instance.return_value = mock_rendered
        mock_get_converter.return_value = mock_converter_instance

        markdown, images = convert_with_marker(b"%PDF-1.4 test content", "test.pdf")

        self.assertIn("Test Paper", markdown)
        self.assertIn("$E=mc^2$", markdown)
        self.assertIsInstance(images, list)
        # Should have converted the PIL mock image to base64
        self.assertEqual(len(images), 1)
        self.assertEqual(images[0][0], "fig1.jpeg")

    @patch("converter._get_marker_converter")
    def test_handles_pdf_without_images(self, mock_get_converter):
        """Should handle PDFs that produce no images."""
        from converter import convert_with_marker

        mock_rendered = MagicMock()
        mock_rendered.markdown = "Just text, no images"
        mock_rendered.images = {}

        mock_converter_instance = MagicMock()
        mock_converter_instance.return_value = mock_rendered
        mock_get_converter.return_value = mock_converter_instance

        markdown, images = convert_with_marker(b"%PDF-1.4 test", "test.pdf")

        self.assertEqual(markdown, "Just text, no images")
        self.assertEqual(len(images), 0)

    @patch("converter._get_marker_converter")
    def test_raises_on_conversion_failure(self, mock_get_converter):
        """Should raise if Marker conversion fails."""
        from converter import convert_with_marker

        mock_converter_instance = MagicMock()
        mock_converter_instance.side_effect = RuntimeError("Conversion failed")
        mock_get_converter.return_value = mock_converter_instance

        with self.assertRaises(RuntimeError):
            convert_with_marker(b"%PDF-1.4 test", "test.pdf")


class TestImagesToBase64(unittest.TestCase):
    """Test conversion of Marker image references to base64 data URLs."""

    def test_replaces_image_references(self):
        """Should replace ![](fig1.jpeg) with base64 data URL."""
        from converter import images_to_base64

        markdown = "See figure: ![](_page_0_Figure_0.jpeg)"
        images = [("_page_0_Figure_0.jpeg", "data:image/jpeg;base64,abc123")]

        result = images_to_base64(markdown, images)
        self.assertIn("data:image/jpeg;base64,abc123", result)
        self.assertNotIn("_page_0_Figure_0.jpeg", result)

    def test_preserves_text_without_images(self):
        """Markdown without images should pass through unchanged."""
        from converter import images_to_base64

        markdown = "Just plain text with $formula$"
        result = images_to_base64(markdown, [])
        self.assertEqual(result, markdown)

    def test_handles_multiple_images(self):
        """Should replace all image references."""
        from converter import images_to_base64

        markdown = "![fig1](_page_0_Figure_0.jpeg) and ![fig2](_page_0_Figure_1.jpeg)"
        images = [
            ("_page_0_Figure_0.jpeg", "data:image/jpeg;base64,aaa"),
            ("_page_0_Figure_1.jpeg", "data:image/png;base64,bbb"),
        ]

        result = images_to_base64(markdown, images)
        self.assertIn("data:image/jpeg;base64,aaa", result)
        self.assertIn("data:image/png;base64,bbb", result)

    def test_preserves_alt_text(self):
        """Should preserve alt text in image references."""
        from converter import images_to_base64

        markdown = "![Architecture Diagram](_page_5_Figure_2.jpeg)"
        images = [("_page_5_Figure_2.jpeg", "data:image/jpeg;base64,xyz")]

        result = images_to_base64(markdown, images)
        self.assertIn("[Architecture Diagram]", result)


class TestMarkerImageConversion(unittest.TestCase):
    """Test PIL Image to base64 conversion helper."""

    def test_pil_image_to_base64_jpeg(self):
        """Should convert PIL Image to JPEG base64 data URL."""
        from PIL import Image

        from converter import pil_image_to_base64

        img = Image.new("RGB", (10, 10), color="red")
        data_url = pil_image_to_base64(img, "test.jpeg")

        self.assertTrue(data_url.startswith("data:image/jpeg;base64,"))
        # Verify the base64 part is valid
        b64_part = data_url.split(",", 1)[1]
        decoded = base64.b64decode(b64_part)
        self.assertGreater(len(decoded), 0)

    def test_pil_image_to_base64_png(self):
        """Should convert PIL Image to PNG base64 data URL."""
        from PIL import Image

        from converter import pil_image_to_base64

        img = Image.new("RGBA", (10, 10), color=(255, 0, 0, 128))
        data_url = pil_image_to_base64(img, "test.png")

        self.assertTrue(data_url.startswith("data:image/png;base64,"))


class TestMarkerConverterOCRDisabled(unittest.TestCase):
    """Test that Marker converter is initialized with disable_ocr=True.

    This is the primary performance optimization: academic papers are text-based
    PDFs and do not need OCR. Skipping OCR reduces conversion from 30+ minutes
    to 1-2 minutes on CPU.
    """

    def setUp(self):
        import converter

        # Reset singleton so _get_marker_converter re-initializes
        self._saved = converter._marker_converter
        converter._marker_converter = None

        # Create mock marker modules so `from marker.X import Y` works
        import types

        self._mock_config_parser_cls = MagicMock()
        mod_parser = types.ModuleType("marker.config.parser")
        mod_parser.ConfigParser = self._mock_config_parser_cls

        self._mock_pdf_converter_cls = MagicMock()
        mod_converter = types.ModuleType("marker.converters.pdf")
        mod_converter.PdfConverter = self._mock_pdf_converter_cls

        self._mock_create_model_dict = MagicMock(return_value={"mock": True})
        mod_models = types.ModuleType("marker.models")
        mod_models.create_model_dict = self._mock_create_model_dict

        # Inject into sys.modules so lazy imports in _get_marker_converter resolve
        self._mocked_keys = [
            "marker.config",
            "marker.config.parser",
            "marker.converters",
            "marker.converters.pdf",
            "marker.models",
        ]
        for key in self._mocked_keys:
            sys.modules.setdefault(key, types.ModuleType(key))
        sys.modules["marker.config.parser"] = mod_parser
        sys.modules["marker.converters.pdf"] = mod_converter
        sys.modules["marker.models"] = mod_models

    def tearDown(self):
        import converter

        converter._marker_converter = self._saved
        for key in self._mocked_keys:
            sys.modules.pop(key, None)

    def test_config_parser_called_with_disable_ocr(self):
        """ConfigParser should be initialized with {'disable_ocr': True}."""
        from converter import _get_marker_converter

        # Setup mock parser instance
        mock_parser = MagicMock()
        mock_parser.generate_config_dict.return_value = {}
        mock_parser.get_processors.return_value = []
        mock_parser.get_renderer.return_value = None
        mock_parser.get_llm_service.return_value = None
        self._mock_config_parser_cls.return_value = mock_parser

        _get_marker_converter()

        self._mock_config_parser_cls.assert_called_once_with({"disable_ocr": True, "output_format": "markdown"})

    def test_pdf_converter_receives_config(self):
        """PdfConverter should receive config from ConfigParser.generate_config_dict()."""
        from converter import _get_marker_converter

        mock_parser = MagicMock()
        mock_parser.generate_config_dict.return_value = {"disable_ocr": True, "output_format": "markdown"}
        mock_parser.get_processors.return_value = "procs"
        mock_parser.get_renderer.return_value = "renderer"
        mock_parser.get_llm_service.return_value = "llm"
        self._mock_config_parser_cls.return_value = mock_parser

        _get_marker_converter()

        self._mock_pdf_converter_cls.assert_called_once_with(
            config={"disable_ocr": True, "output_format": "markdown"},
            artifact_dict={"mock": True},
            processor_list="procs",
            renderer="renderer",
            llm_service="llm",
        )


class TestExistingFunctions(unittest.TestCase):
    """Test that existing functions still work after refactor."""

    def test_fix_utf8_mojibake_preserved(self):
        """fix_utf8_mojibake should still work."""
        from converter import fix_utf8_mojibake

        broken = "1.6\u00e2\u20ac\u201c72\u00e2\u20ac\u201c"
        result = fix_utf8_mojibake(broken)
        self.assertEqual(result, "1.6\u201372\u2013")

    def test_extract_metadata_prompt_works(self):
        """extract_metadata_prompt should still work."""
        from converter import extract_metadata_prompt

        prompt = extract_metadata_prompt("# Test\nContent")
        self.assertIn("Test", prompt)
        self.assertIn("title", prompt.lower())

    def test_parse_metadata_response_works(self):
        """parse_metadata_response should still work."""
        import json

        from converter import parse_metadata_response

        response = json.dumps({"title": "Test", "authors": ["A"], "keywords": []})
        result = parse_metadata_response(response)
        self.assertEqual(result.title, "Test")


def teardown_module():
    """Restore real modules after all tests in this file complete."""
    for name, saved in _saved_modules.items():
        if saved is not None:
            sys.modules[name] = saved
        elif name in sys.modules:
            del sys.modules[name]


if __name__ == "__main__":
    unittest.main()
