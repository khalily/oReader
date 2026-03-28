import json
import logging
import os
import re
from dataclasses import dataclass, field

import openai

logger = logging.getLogger(__name__)

METADATA_EXTRACTION_PROMPT = """You are an academic paper metadata extractor. Given the following Markdown content converted from a PDF paper, extract structured metadata.

IMPORTANT: Return ONLY a JSON object, no markdown fences or explanation.

Required fields:
- title: Paper title (string)
- authors: List of author names (array of strings)
- abstract: Paper abstract (string, may be empty)
- keywords: List of keywords/topics (array of strings, 3-8 items)
- published_year: Publication year (string, e.g. "2024")
- doi: DOI if found (string, may be empty)

Markdown content:
{markdown}

JSON response:"""


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
        cleaned = re.sub(r'^```\w*\n?', '', cleaned)
        cleaned = re.sub(r'\n?```$', '', cleaned)

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
    api_key = os.getenv("LLM_API_KEY")
    base_url = os.getenv("LLM_BASE_URL", "https://api.openai.com/v1")
    model = os.getenv("LLM_MODEL", "gpt-4o-mini")

    if not api_key:
        logger.warning("LLM_API_KEY not set, skipping metadata extraction")
        return PaperMetadataResult()

    client = openai.OpenAI(api_key=api_key, base_url=base_url)
    prompt = extract_metadata_prompt(markdown)

    try:
        response = client.chat.completions.create(
            model=model,
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
    api_key = os.getenv("LLM_API_KEY")
    base_url = os.getenv("LLM_BASE_URL", "https://api.openai.com/v1")
    model = os.getenv("LLM_MODEL", "gpt-4o-mini")

    if not api_key:
        return markdown

    client = openai.OpenAI(api_key=api_key, base_url=base_url)

    prompt = f"""Fix formatting issues in this Markdown converted from a PDF academic paper.
Rules:
- Fix broken LaTeX formulas (ensure $...$ and $$...$$ are properly paired)
- Fix table formatting
- Remove page numbers, headers, footers
- Fix paragraph breaks
- Keep all content, don't remove anything important
- Return the corrected markdown only

Markdown:
{markdown[:20000]}"""

    try:
        response = client.chat.completions.create(
            model=model,
            messages=[{"role": "user", "content": prompt}],
            temperature=0.0,
            max_tokens=16000,
        )
        return response.choices[0].message.content or markdown
    except Exception as e:
        logger.error("LLM markdown refinement failed: %s", e)
        return markdown
