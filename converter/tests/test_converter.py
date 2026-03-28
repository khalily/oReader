import json
import unittest
from unittest.mock import patch, MagicMock

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

# Mock openai before importing converter (not installed in test env)
sys.modules['openai'] = MagicMock()

from converter import extract_metadata_prompt, parse_metadata_response


class TestConverter(unittest.TestCase):

    def test_extract_metadata_prompt_contains_required_fields(self):
        prompt = extract_metadata_prompt("# Test Paper\n\nSome content")
        self.assertIn("title", prompt.lower())
        self.assertIn("authors", prompt.lower())
        self.assertIn("abstract", prompt.lower())
        self.assertIn("keywords", prompt.lower())

    def test_parse_metadata_response_valid_json(self):
        response = json.dumps({
            "title": "Test Paper",
            "authors": ["Alice", "Bob"],
            "abstract": "This is abstract",
            "keywords": ["ML", "AI"],
            "published_year": "2024",
            "doi": "10.1234/test"
        })
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


if __name__ == '__main__':
    unittest.main()
