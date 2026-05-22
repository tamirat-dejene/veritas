"""
HTTP client for the Hugging Face AI grading space.

This module owns ALL outbound network logic:
  • Building the batch payload
  • Sending the single POST request
  • Parsing the response

It is fully decoupled from event-parsing (models.py) and grading logic (grader.py).
"""
from __future__ import annotations

import logging
from typing import Any

import httpx

from app.config import settings

logger = logging.getLogger("grading.ai_client")

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

# Pulled from centralized Settings (configurable via env vars).
# HF_EVALUATE_URL and HF_TIMEOUT_SECONDS live in app.config.Settings.


# ---------------------------------------------------------------------------
# Payload helpers
# ---------------------------------------------------------------------------

def build_batch_payload(
    items: list[dict[str, Any]],
) -> dict[str, Any]:
    """
    Assemble the batch payload expected by the HF space.

    Each *item* dict must contain:
        question_id, student_text, expected_answer, keywords
    """
    return {"items": items}


# ---------------------------------------------------------------------------
# Network call
# ---------------------------------------------------------------------------

async def evaluate_short_answers(
    batch_items: list[dict[str, Any]],
) -> dict[str, float]:
    """
    Send a **single** batched POST to the Hugging Face grading space
    and return a mapping ``{question_id: score_percentage}``.

    Raises no exceptions to the caller — any failure is logged and an
    empty dict is returned so the grading pipeline can still finish
    (affected questions receive 0 points).
    """
    if not batch_items:
        logger.info("No short-answer items to evaluate — skipping AI call.")
        return {}

    payload = build_batch_payload(batch_items)

    headers = {}
    if settings.HF_TOKEN:
        headers["Authorization"] = f"Bearer {settings.HF_TOKEN}"

    try:
        async with httpx.AsyncClient(timeout=settings.HF_TIMEOUT_SECONDS) as client:
            logger.info(
                "Sending batch of %d short-answer items to HF space (%s)",
                len(batch_items),
                settings.HF_EVALUATE_URL,
            )
            response = await client.post(
                settings.HF_EVALUATE_URL, json=payload, headers=headers
            )
            response.raise_for_status()

            body = response.json()
            graded: list[dict[str, Any]] = body.get("graded_items", [])

            scores: dict[str, float] = {
                item["question_id"]: float(item["score_percentage"])
                for item in graded
            }
            logger.info(
                "Received AI scores for %d / %d items.",
                len(scores),
                len(batch_items),
            )
            return scores

    except httpx.TimeoutException:
        logger.error(
            "HF space request timed out after %.0fs — "
            "all short-answer questions will receive 0 points.",
            settings.HF_TIMEOUT_SECONDS,
        )
    except httpx.HTTPStatusError as exc:
        logger.error(
            "HF space returned HTTP %d: %s",
            exc.response.status_code,
            exc.response.text[:500],
        )
    except httpx.RequestError as exc:
        logger.error(
            "Network error contacting HF space: %s", exc,
        )
    except (KeyError, ValueError, TypeError) as exc:
        logger.error(
            "Unexpected response format from HF space: %s", exc,
        )
    except Exception as exc:
        logger.exception("Unhandled error during AI evaluation: %s", exc)

    # Graceful degradation: return empty scores so pipeline completes.
    return {}
