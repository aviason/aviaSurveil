#!/usr/bin/env python3
"""Extract bounded, searchable text from a regulatory PDF."""

from __future__ import annotations

import json
import pathlib
import shutil
import subprocess
import sys
import tempfile

from pypdf import PdfReader


def pdftotext_candidates() -> list[pathlib.Path]:
    candidates: list[pathlib.Path] = []
    discovered = shutil.which("pdftotext")
    if discovered:
        candidates.append(pathlib.Path(discovered))
    candidates.append(
        pathlib.Path.home()
        / ".cache/codex-runtimes/codex-primary-runtime/dependencies/native/poppler/poppler/bin/pdftotext"
    )
    return [candidate for candidate in candidates if candidate.is_file()]


def extract_with_poppler(
    input_path: pathlib.Path, output_path: pathlib.Path, page_count: int
) -> dict[str, int | str | list[int]] | None:
    for executable in pdftotext_candidates():
        temporary_path: pathlib.Path | None = None
        try:
            with tempfile.NamedTemporaryFile(
                prefix="avia-pdf-text-",
                suffix=".txt",
                delete=False,
            ) as temporary:
                temporary_path = pathlib.Path(temporary.name)
            subprocess.run(
                [str(executable), "-layout", str(input_path), str(temporary_path)],
                check=True,
                capture_output=True,
                timeout=900,
            )
            text = temporary_path.read_text(encoding="utf-8", errors="replace")
            page_text = text.split("\f")
            if len(page_text) > page_count:
                page_text = page_text[:page_count]
            if len(page_text) < page_count:
                page_text.extend([""] * (page_count - len(page_text)))
            write_page_sections(output_path, page_text)
            missing_pages = [
                page_number
                for page_number, page in enumerate(page_text, start=1)
                if not page.strip()
            ]
            return {
                "extractionStrategyVersion": 2,
                "engine": "poppler-pdftotext",
                "pageCount": page_count,
                "pagesWithExtractedText": page_count - len(missing_pages),
                "pagesWithoutExtractedText": missing_pages,
                "characterCount": sum(len(page) for page in page_text),
            }
        except (OSError, subprocess.SubprocessError):
            continue
        finally:
            if temporary_path is not None:
                temporary_path.unlink(missing_ok=True)
    return None


def write_page_sections(output_path: pathlib.Path, pages: list[str]) -> None:
    with output_path.open("w", encoding="utf-8", newline="\n") as stream:
        for page_number, text in enumerate(pages, start=1):
            stream.write(f"\n\n--- PAGE {page_number} [SEARCHABLE] ---\n\n")
            stream.write(text)


def main() -> int:
    if len(sys.argv) != 3:
        raise SystemExit("usage: extract-pdf-text.py INPUT.pdf OUTPUT.txt")

    input_path = pathlib.Path(sys.argv[1]).resolve()
    output_path = pathlib.Path(sys.argv[2]).resolve()
    if input_path.suffix.lower() != ".pdf":
        raise SystemExit("input must be a PDF")

    reader = PdfReader(str(input_path), strict=False)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    poppler_result = extract_with_poppler(input_path, output_path, len(reader.pages))
    if poppler_result is not None:
        print(json.dumps(poppler_result, sort_keys=True))
        return 0

    pages: list[str] = []
    for page in reader.pages:
        try:
            pages.append(page.extract_text() or "")
        except Exception:
            # A failed text read remains an OCR candidate; the manifest preserves
            # the exact page number in pagesWithoutExtractedText.
            pages.append("")
    write_page_sections(output_path, pages)
    missing_pages = [
        page_number
        for page_number, text in enumerate(pages, start=1)
        if not text.strip()
    ]

    print(
        json.dumps(
            {
                "extractionStrategyVersion": 2,
                "engine": "pypdf-fallback",
                "pageCount": len(reader.pages),
                "pagesWithExtractedText": len(pages) - len(missing_pages),
                "pagesWithoutExtractedText": missing_pages,
                "characterCount": sum(len(text) for text in pages),
            },
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
