import base64
import io
import json
import os
import re
import tempfile
from dataclasses import dataclass, field

import openai
from PIL import Image

from logging_config import get_logger  # noqa: E402

logger = get_logger(__name__)


def fix_utf8_mojibake(text: str) -> str:
    """Fix double-encoded UTF-8 mojibake produced by PDF extraction tools.

    Some PDF extractors output text where UTF-8 bytes (e.g. en-dash E2 80 93)
    were incorrectly decoded as Windows-1252, producing 3-char sequences
    like â€" (U+00E2 U+20AC U+201C) instead of – (U+2013).
    """
    replacements = {
        "\u00e2\u20ac\u201c": "\u2013",  # â€" → – (en-dash)
        "\u00e2\u20ac\u201d": "\u2014",  # â€" → — (em-dash)
        "\u00e2\u20ac\u0153": "\u201c",  # â€œ → " (left double quote)
        "\u00e2\u20ac\u02dc": "\u2018",  # â€˜ → ' (left single quote)
        "\u00e2\u20ac\u2122": "\u2019",  # â€™ → ' (right single quote)
        "\u00e2\u20ac\u00a6": "\u2026",  # â€¦ → … (ellipsis)
        "\u00e2\u20ac\u00b8": "\u2032",  # â€¸ → ′ (prime)
        "\u00e2\u20ac\u00b9": "\u2033",  # â€¹ → ″ (double prime)
    }
    for broken, correct in replacements.items():
        text = text.replace(broken, correct)
    return text


# --- Marker PDF conversion ---

_marker_converter = None


def _get_marker_converter():
    """Lazily initialize Marker PdfConverter (model loading is expensive)."""
    global _marker_converter
    if _marker_converter is not None:
        return _marker_converter
    from marker.converters.pdf import PdfConverter
    from marker.models import create_model_dict

    logger.info("Initializing Marker converter and loading models...")
    model_dict = create_model_dict()
    _marker_converter = PdfConverter(artifact_dict=model_dict)
    logger.info("Marker converter initialized")
    return _marker_converter


def pil_image_to_base64(img: Image.Image, filename: str) -> str:
    """Convert a PIL Image to a base64 data URL.

    Args:
        img: PIL Image object
        filename: Original filename (used to determine format)

    Returns:
        data:image/...;base64,... string
    """
    ext = os.path.splitext(filename)[1].lower()
    fmt = "PNG" if ext == ".png" else "JPEG"
    mime = "image/png" if ext == ".png" else "image/jpeg"

    buf = io.BytesIO()
    img.save(buf, format=fmt)
    b64 = base64.b64encode(buf.getvalue()).decode("ascii")
    return f"data:{mime};base64,{b64}"


def images_to_base64(markdown: str, images: list[tuple[str, str]]) -> str:
    """Replace image filename references in markdown with base64 data URLs.

    Args:
        markdown: Markdown text with image references like ![](filename.jpeg)
        images: List of (filename, base64_data_url) tuples

    Returns:
        Markdown with image references replaced by base64 data URLs
    """
    for filename, data_url in images:
        markdown = markdown.replace(f"]({filename})", f"]({data_url})")
    return markdown


def convert_with_marker(
    pdf_content: bytes,
    filename: str,
) -> tuple[str, list[tuple[str, str]]]:
    """Convert PDF to Markdown using Marker.

    Args:
        pdf_content: Raw PDF bytes
        filename: Original filename for logging

    Returns:
        Tuple of (markdown_text, [(image_filename, base64_data_url), ...])
    """
    converter = _get_marker_converter()

    # Marker requires a file path, so write to temp file
    with tempfile.NamedTemporaryFile(suffix=".pdf", delete=False) as tmp:
        tmp.write(pdf_content)
        tmp_path = tmp.name

    try:
        logger.info("Starting Marker conversion", filename=filename)
        rendered = converter(tmp_path)

        markdown = rendered.markdown
        logger.info(
            "Marker conversion complete",
            markdown_length=len(markdown),
            image_count=len(rendered.images),
        )

        # Convert PIL images to base64 data URLs
        image_data: list[tuple[str, str]] = []
        for img_name, pil_img in rendered.images.items():
            data_url = pil_image_to_base64(pil_img, img_name)
            image_data.append((img_name, data_url))

        return markdown, image_data

    finally:
        os.unlink(tmp_path)


# --- LLM metadata extraction (unchanged) ---

_openai_client = None


def _get_openai_client() -> openai.OpenAI | None:
    global _openai_client
    if _openai_client is not None:
        return _openai_client
    api_key = os.getenv("LLM_API_KEY")
    if not api_key:
        return None
    base_url = os.getenv("LLM_BASE_URL", "https://api.openai.com/v1")
    _openai_client = openai.OpenAI(api_key=api_key, base_url=base_url)
    return _openai_client


def _get_llm_model() -> str:
    return os.getenv("LLM_MODEL", "gpt-4o-mini")


METADATA_EXTRACTION_PROMPT = (
    "You are an academic paper metadata extractor. "
    "Given the following Markdown content converted from a PDF paper, "
    "extract structured metadata.\n"
    "\n"
    "IMPORTANT: Return ONLY a JSON object, no markdown fences or "
    "explanation.\n"
    "\n"
    "Required fields:\n"
    "- title: Paper title (string)\n"
    "- authors: List of author names (array of strings)\n"
    "- abstract: Paper abstract (string, may be empty)\n"
    "- keywords: List of keywords/topics (array of strings, 3-8 items)\n"
    '- published_year: Publication year (string, e.g. "2024")\n'
    "- doi: DOI if found (string, may be empty)\n"
    "\n"
    "Markdown content:\n"
    "{markdown}\n"
    "\n"
    "JSON response:"
)


@dataclass
class PaperMetadataResult:
    title: str = ""
    authors: list = field(default_factory=list)
    abstract: str = ""
    keywords: list = field(default_factory=list)
    published_year: str = ""
    doi: str = ""


def extract_metadata_prompt(markdown: str) -> str:
    return METADATA_EXTRACTION_PROMPT.format(markdown=markdown[:15000])


def parse_metadata_response(response: str) -> PaperMetadataResult | None:
    """Parse LLM response into PaperMetadataResult."""
    # Strip markdown code fences if present
    cleaned = response.strip()
    if cleaned.startswith("```"):
        cleaned = re.sub(r"^```\w*\n?", "", cleaned)
        cleaned = re.sub(r"\n?```$", "", cleaned)

    try:
        data = json.loads(cleaned)
    except json.JSONDecodeError:
        logger.error("Failed to parse metadata JSON", response_preview=cleaned[:200])
        return None

    return PaperMetadataResult(
        title=data.get("title", ""),
        authors=data.get("authors", []),
        abstract=data.get("abstract", ""),
        keywords=data.get("keywords", []),
        published_year=str(data.get("published_year", "")),
        doi=data.get("doi", ""),
    )


def extract_metadata(markdown: str) -> PaperMetadataResult:
    """Call LLM to extract metadata from markdown content."""
    client = _get_openai_client()
    if client is None:
        logger.warning("LLM_API_KEY not set, skipping metadata extraction")
        return PaperMetadataResult()

    prompt = extract_metadata_prompt(markdown)

    try:
        response = client.chat.completions.create(
            model=_get_llm_model(),
            messages=[{"role": "user", "content": prompt}],
            temperature=0.0,
            max_tokens=2000,
        )
        content = response.choices[0].message.content or ""
        result = parse_metadata_response(content)
        if result is not None:
            logger.info(
                "Metadata extracted",
                title=result.title[:50] if result.title else "",
                author_count=len(result.authors),
            )
            return result
        return PaperMetadataResult()
    except Exception as e:
        logger.error("LLM metadata extraction failed", error=str(e))
        return PaperMetadataResult()
