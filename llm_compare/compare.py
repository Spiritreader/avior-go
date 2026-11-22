#!/usr/bin/env python3
"""Compare two descriptions through movie_nfo_lib's configured LLM pipeline."""
from __future__ import annotations

import json
import math
import sys

from movie_metadata.config import RunConfig
from movie_metadata.llm import smart_compare_cached


def main() -> int:
    try:
        request = json.load(sys.stdin)
        left = request.get("left", "")
        right = request.get("right", "")
        threshold = float(request.get("threshold", 0.7))
        if not isinstance(left, str) or not isinstance(right, str):
            raise ValueError("left and right must be strings")
        if not math.isfinite(threshold) or not 0 <= threshold <= 1:
            raise ValueError("threshold must be finite and within [0, 1]")
        score, explanation = smart_compare_cached(left, right, RunConfig())
        score = float(score)
        if not math.isfinite(score) or not 0 <= score <= 1:
            raise ValueError("LLM score must be finite and within [0, 1]")
        json.dump(
            {
                "score": score,
                "same_content": score > threshold,
                "explanation": str(explanation or ""),
            },
            sys.stdout,
            ensure_ascii=False,
        )
        sys.stdout.write("\n")
        return 0
    except Exception as exc:
        print(f"description comparison failed: {type(exc).__name__}: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
