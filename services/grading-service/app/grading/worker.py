"""
Grading Worker — Kafka consumer for ``candidate.exam.ready_for_grading`` events.

This module is the event-driven entry point. It:
  1. Subscribes to the ``candidate.exam.ready_for_grading`` Kafka topic.
  2. Deserialises each message and delegates to the grading pipeline.
  3. Persists the result using the secure GradingRepository.

Runs as a long-lived asyncio background task spawned during FastAPI lifespan.
"""
from __future__ import annotations

import asyncio
import json
import logging
from typing import Any
import asyncpg

from aiokafka import AIOKafkaConsumer

from app.config import settings
from app.grading.grader import grade_exam, ExamGradeReport
from app.repository.grading_repository import GradingRepository

logger = logging.getLogger("grading.worker")

# ---------------------------------------------------------------------------
# Kafka topic
# ---------------------------------------------------------------------------

GRADING_TOPIC = "candidate.exam.ready_for_grading"
CONSUMER_GROUP = "grading-service-group"


# ---------------------------------------------------------------------------
# Public entry point (also usable in tests / manual invocations)
# ---------------------------------------------------------------------------

async def process_incoming_event(payload: dict[str, Any], pool: asyncpg.Pool) -> ExamGradeReport:
    """
    Top-level handler: parse → grade → persist to database.

    Can be called directly (e.g. from an HTTP endpoint or test harness)
    by passing a database pool.
    """
    report = await grade_exam(payload)
    
    # Secure database storage step
    repo = GradingRepository(pool)
    logger.info("Saving grading report securely to database for session=%s", report.session_id)
    await repo.save_grading_report(report, graded_by="system")
    
    return report


# ---------------------------------------------------------------------------
# Kafka consumer loop
# ---------------------------------------------------------------------------

async def run_grading_consumer(pool: asyncpg.Pool) -> None:
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
                await _handle_message(msg, pool)

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


async def _handle_message(msg: Any, pool: asyncpg.Pool) -> None:
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

        await process_incoming_event(payload, pool)

    except json.JSONDecodeError as exc:
        logger.error("Failed to decode Kafka message as JSON: %s", exc)
    except Exception as exc:
        logger.exception(
            "Unhandled error processing grading message: %s", exc,
        )
