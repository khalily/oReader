import json
import logging
import os
import re
from dataclasses import dataclass, field

import openai

logger = logging.getLogger(__name__)


def fix_utf8_mojibake(text: str) -> str:
    """Fix double-encoded UTF-8 mojibake produced by MinerU PDF extraction.

    MinerU sometimes outputs text where UTF-8 bytes (e.g. en-dash E2 80 93)
    were incorrectly decoded as Windows-1252, producing 3-char sequences
    like â€" (U+00E2 U+20AC U+201C) instead of – (U+2013).
    """
    # UTF-8 bytes → Windows-1252 misinterpretation → correct Unicode
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


# Q6: Reuse OpenAI client instead of creating per-request
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
        logger.error("Failed to parse metadata JSON: %s", cleaned[:200])
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
        return result if result is not None else PaperMetadataResult()
    except Exception as e:
        logger.error("LLM metadata extraction failed: %s", e)
        return PaperMetadataResult()


def refine_markdown(markdown: str) -> str:
    """Call LLM to fix formatting issues in the markdown."""
    client = _get_openai_client()
    if client is None:
        return markdown

    prompt = (
        "Fix formatting issues in this Markdown converted from a PDF academic paper.\n"
        "Rules:\n"
        "- Restore mathematical formulas as LaTeX: use $...$ for inline and $$...$$ for display math.\n"
        "  The PDF-to-text conversion often destroys formulas. Look for patterns like\n"
        "  garbled characters near math operators, subscripts/superscripts rendered as plain text,\n"
        "  or symbols like ×, ≤, ≥, →, α, β, γ that should be LaTeX commands.\n"
        "  Examples: '1.6×' → '$1.6\\times$', 'awin' → '$a_{win}$',\n"
        "  'Dmax' → '$D_{max}$', '2 s' → '$2\\mu s$'.\n"
        "- Fix broken LaTeX formulas (ensure $...$ and $$...$$ are properly paired)\n"
        "- Convert any HTML tables to GFM pipe tables (| col1 | col2 |) where possible.\n"
        "  If a table is too complex for GFM, keep it as HTML.\n"
        "- Remove page numbers, headers, footers\n"
        "- Fix paragraph breaks\n"
        "- Keep all content, don't remove anything important\n"
        "- Keep all <!--IMG_N--> placeholders exactly as they are (image references)\n"
        "- Return the corrected markdown only\n"
        "\n"
        f"Markdown:\n{markdown[:30000]}"
    )

    try:
        response = client.chat.completions.create(
            model=_get_llm_model(),
            messages=[{"role": "user", "content": prompt}],
            temperature=0.0,
            max_tokens=32000,
        )
        return response.choices[0].message.content or markdown
    except Exception as e:
        logger.error("LLM markdown refinement failed: %s", e)
        return markdown
