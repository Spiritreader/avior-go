#!/usr/bin/env python3
"""Compare two descriptions through movie_nfo_lib's configured LLM pipeline."""
from __future__ import annotations

import contextlib
import json
import math
import os
import re
import sys
from movie_metadata import llm
from movie_metadata.config import RunConfig

def _metadata_value(text: str, label: str, default: str) -> str:
    match = re.search(rf"^{re.escape(label)}:\s*(.*?)\s*$", text, re.MULTILINE)
    return match.group(1) if match else default

def _metadata_number(text: str, label: str, default: int | float) -> int | float:
    value = _metadata_value(text, label, "")
    if not value:
        return default
    try:
        normalized = value[:-1].strip() if value.endswith("s") else value
        return float(normalized) if isinstance(default, float) else int(normalized)
    except ValueError:
        return default


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
        with contextlib.redirect_stdout(sys.stderr):
            score, explanation = llm.smart_compare_cached(
                left,
                right,
                RunConfig(max_tokens_deep=int(os.environ.get("AVIOR_LLM_MAX_TOKENS", "100"))),
            )
        used_instance = llm.active_client_key
        used_url = (llm.CLIENTS.get(used_instance) or {}).get("base_url", "")
        score = float(score)
        explanation_text = str(explanation or "")
        metadata = {
            "max_tokens": int(os.environ.get("AVIOR_LLM_MAX_TOKENS", "100")),
            "tokens_used": _metadata_number(explanation_text, "Deep model tokens_used", 0),
            "elapsed_seconds": _metadata_number(explanation_text, "Time deepcompare", 0.0),
            "finish_reason": _metadata_value(explanation_text, "Deep model finish reason", ""),
        }
        if not math.isfinite(score) or not 0 <= score <= 1:
            raise ValueError("LLM score must be finite and within [0, 1]")
        json.dump(
            {
                "score": score,
                "same_content": score > threshold,
                "explanation": str(explanation or ""),
                "used_instance": used_instance or "",
                "used_model": used_model or "",
                "used_url": used_url,
                "max_tokens": metadata["max_tokens"],
                "tokens_used": metadata["tokens_used"],
                "elapsed_seconds": metadata["elapsed_seconds"],
                "finish_reason": metadata["finish_reason"],
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
