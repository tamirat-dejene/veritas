"""
Grading Worker — Kafka consumer for ``candidate.exam.ready_for_grading`` events.

This module is the event-driven entry point. It:
  1. Subscribes to the ``candidate.exam.ready_for_grading`` Kafka topic.
  2. Deserialises each message and delegates to the grading pipeline.
  3. Persists the result (simulated DB save for now).

Runs as a long-lived asyncio background task spawned during FastAPI lifespan.
"""
from __future__ import annotations

import asyncio
import json
import logging
from dataclasses import asdict
from typing import Any

from aiokafka import AIOKafkaConsumer

from app.config import settings
from app.grading.grader import grade_exam, ExamGradeReport

logger = logging.getLogger("grading.worker")

# ---------------------------------------------------------------------------
# Kafka topic
# ---------------------------------------------------------------------------

GRADING_TOPIC = "candidate.exam.ready_for_grading"
CONSUMER_GROUP = "grading-service-group"


# ---------------------------------------------------------------------------
# Simulated persistence
# ---------------------------------------------------------------------------

async def _persist_grade_report(report: ExamGradeReport) -> None:
    """
    Simulate saving the grading report to the database.

    In production this would INSERT into a ``grade_results`` table and
    optionally publish a ``grading.completed`` Kafka event for downstream
    services (notifications, analytics, etc.).
    """
    logger.info(
        "[DB-SAVE] Persisting grade report for session=%s  "
        "candidate=%s  exam=%s  score=%.2f / %.2f (%.1f%%)",
        report.session_id,
        report.candidate_id,
        report.exam_id,
        report.total_awarded_points,
        report.total_max_points,
        report.percentage,
    )

    # Pretty-print the per-question breakdown
    for qr in report.question_results:
        logger.info(
            "  ├─ [%s] q=%s  %-20s  %6.2f / %6.2f  (%s)",
            qr.question_type.upper()[:3],
            qr.question_id[:12] + "…",
            qr.title[:20],
            qr.awarded_points,
            qr.max_points,
            qr.status,
        )

    logger.info(
        "[DB-SAVE] Grade report saved successfully for session=%s.",
        report.session_id,
    )


# ---------------------------------------------------------------------------
# Public entry point (also usable in tests / manual invocations)
# ---------------------------------------------------------------------------

async def process_incoming_event(payload: dict[str, Any]) -> ExamGradeReport:
    """
    Top-level handler: parse → grade → persist.

    Can be called directly (e.g. from an HTTP endpoint or test harness)
    without requiring Kafka.
    """
    report = await grade_exam(payload)
    await _persist_grade_report(report)
    return report


# ---------------------------------------------------------------------------
# Kafka consumer loop
# ---------------------------------------------------------------------------

async def run_grading_consumer() -> None:
    """
    Long-running Kafka consumer coroutine.

    Follows the same resilient pattern used by the proctoring-service consumer:
      • Auto-retry on transient errors with 5-second backoff.
      • Clean shutdown on ``asyncio.CancelledError``.
    """
    while True:
        consumer = AIOKafkaConsumer(
            GRADING_TOPIC,
            bootstrap_servers=settings.KAFKA_BROKERS,
            group_id=CONSUMER_GROUP,
            auto_offset_reset="earliest",
            enable_auto_commit=True,
        )
        try:
            await consumer.start()
            logger.info(
                "Kafka consumer started — topic: %s  group: %s",
                GRADING_TOPIC,
                CONSUMER_GROUP,
            )

            async for msg in consumer:
                await _handle_message(msg)

        except asyncio.CancelledError:
            logger.info("Kafka consumer received shutdown signal.")
            break
        except Exception as exc:
            logger.error(
                "Kafka consumer error: %s — retrying in 5s", exc,
            )
            await asyncio.sleep(5)
        finally:
            try:
                await consumer.stop()
            except Exception:
                pass


async def _handle_message(msg) -> None:
    """Process a single Kafka message."""
    try:
        payload: dict[str, Any] = json.loads(msg.value)

        event_type = payload.get("event_type", "unknown")
        event_id = payload.get("event_id", "n/a")

        if event_type != "candidate.exam.ready_for_grading":
            logger.warning(
                "Ignoring unexpected event_type=%s  event_id=%s",
                event_type,
                event_id,
            )
            return

        logger.info(
            "Received grading event  event_id=%s  session=%s",
            event_id,
            payload.get("session_id", "n/a"),
        )

        await process_incoming_event(payload)

    except json.JSONDecodeError as exc:
        logger.error("Failed to decode Kafka message as JSON: %s", exc)
    except Exception as exc:
        logger.exception(
            "Unhandled error processing grading message: %s", exc,
        )
