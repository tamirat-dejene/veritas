"""
Grading Worker — Kafka consumer for ``exam.session.ready_for_grading`` events.

This module is the event-driven entry point. It:
  1. Subscribes to the ``exam.session.ready_for_grading`` Kafka topic.
  2. Validates that the event is version "3.0" (rejects older fat events).
  3. Calls the candidate-service internal HTTP endpoint to fetch the full
     grading payload (questions + answers + evaluation criteria).
  4. Delegates to the grading pipeline.
  5. Persists the result using the secure GradingRepository.

Runs as a long-lived asyncio background task spawned during FastAPI lifespan.
"""
from __future__ import annotations

import asyncio
import json
import logging
import time
from typing import Any
from uuid import UUID
import asyncpg

from aiokafka import AIOKafkaConsumer
from pydantic import ValidationError

from app.config import settings
from app.grading.candidate_client import CandidateServiceClient, CandidateServiceError
from app.grading.grader import grade_exam, ExamGradeReport
from app.grading.models import ExamReadyForGradingEvent
from app.repository.grading_repository import GradingRepository
from app.grading.producer import KafkaProducer

logger = logging.getLogger("grading.worker")

# ---------------------------------------------------------------------------
# Kafka topic
# ---------------------------------------------------------------------------

GRADING_TOPIC = "exam.session.ready_for_grading"
COMPLETED_TOPIC = "grading.session.completed"
CONSUMER_GROUP = "grading-service-group"
ACCEPTED_VERSION = "3.0"


# ---------------------------------------------------------------------------
# Public entry point (also usable in tests / manual invocations)
# ---------------------------------------------------------------------------

async def process_incoming_event(
    payload: dict[str, Any],
    pool: asyncpg.Pool,
    candidate_client: CandidateServiceClient,
    producer: KafkaProducer | None = None,
) -> ExamGradeReport:
    """
    Top-level handler: parse slim trigger → fetch grading payload → grade → persist.

    Can be called directly (e.g. from an HTTP endpoint or test harness)
    by passing a database pool and a pre-started CandidateServiceClient.
    """
    t_start = time.monotonic()

    # 1. Parse and validate the slim trigger event
    event = ExamReadyForGradingEvent.model_validate(payload)

    # 2. Reject legacy fat events (version != "3.0")
    if event.version != ACCEPTED_VERSION:
        logger.warning(
            "Rejecting event with unsupported version=%s  event_id=%s  session=%s",
            event.version,
            event.event_id,
            event.session_id,
        )
        raise ValueError(
            f"Unsupported event version '{event.version}'. "
            f"Only version '{ACCEPTED_VERSION}' is accepted."
        )

    repo = GradingRepository(pool)

    # 3. Mark this session as 'pending' so status polls return a meaningful
    #    response while grading is still in progress instead of a 404.
    await repo.create_pending_result(
        session_id=UUID(event.session_id),
        exam_id=UUID(event.exam_id),
        candidate_id=UUID(event.candidate_id),
        enterprise_id=UUID(event.enterprise_id),
        enrollment_id=UUID(event.enrollment_id),
    )

    # 4. Fetch full grading payload from candidate-service internal endpoint
    logger.info(
        "Fetching grading payload  event_id=%s  session=%s",
        event.event_id,
        event.session_id,
    )
    grading_payload = await candidate_client.fetch_grading_payload(
        session_id=event.session_id,
        enterprise_id=event.enterprise_id,
    )

    # 5. Grade the exam
    report = await grade_exam(event_id=event.event_id, payload=grading_payload)

    # 6. Secure database storage (updates the pending row → graded)
    logger.info("Saving grading report securely to database for session=%s", report.session_id)
    await repo.save_grading_report(report, graded_by="system")

    elapsed_ms = (time.monotonic() - t_start) * 1000
    logger.info(
        "Grading pipeline complete  event_id=%s  session=%s  "
        "score=%.2f/%.2f (%.1f%%)  elapsed_ms=%.0f",
        event.event_id,
        event.session_id,
        report.total_awarded_points,
        report.total_max_points,
        report.percentage,
        elapsed_ms,
    )

    # 7. Publish grading completed event to Kafka
    if producer:
        try:
            logger.info("Publishing grading completed event for session=%s", report.session_id)
            await producer.publish(
                COMPLETED_TOPIC,
                {
                    "event_id": str(event.event_id),
                    "event_type": COMPLETED_TOPIC,
                    "version": "1.0",
                    "timestamp": int(time.time() * 1000),
                    "session_id": str(report.session_id),
                    "candidate_id": str(report.candidate_id),
                    "enterprise_id": str(report.enterprise_id),
                    "exam_id": str(report.exam_id),
                    "candidate_name": grading_payload.candidate_name,
                    "candidate_email": grading_payload.candidate_email,
                    "exam_title": grading_payload.exam_title,
                    "total_awarded_points": float(report.total_awarded_points),
                    "total_max_points": float(report.total_max_points),
                    "percentage": float(report.percentage),
                }
            )
        except Exception as exc:
            logger.error(
                "Failed to publish grading completed event for session=%s: %s",
                report.session_id,
                exc,
            )

    return report


# ---------------------------------------------------------------------------
# Kafka consumer loop
# ---------------------------------------------------------------------------

async def run_grading_consumer(
    pool: asyncpg.Pool,
    candidate_client: CandidateServiceClient,
    producer: KafkaProducer,
) -> None:
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
                await _handle_message(msg, pool, candidate_client, producer)

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


async def _handle_message(
    msg: Any,
    pool: asyncpg.Pool,
    candidate_client: CandidateServiceClient,
    producer: KafkaProducer | None = None,
) -> None:
    """Process a single Kafka message."""
    try:
        payload: dict[str, Any] = json.loads(msg.value)

        event_type = payload.get("event_type", "unknown")
        event_id = payload.get("event_id", "n/a")
        version = payload.get("version", "unknown")
        session_id = payload.get("session_id", "n/a")

        if event_type != "exam.session.ready_for_grading":
            logger.warning(
                "Ignoring unexpected event_type=%s  event_id=%s  "
                "topic=%s  partition=%s  offset=%s",
                event_type,
                event_id,
                msg.topic,
                msg.partition,
                msg.offset,
            )
            return

        if version != ACCEPTED_VERSION:
            logger.warning(
                "Ignoring event with unsupported version=%s  event_id=%s  session=%s  "
                "(only version %s is accepted)",
                version,
                event_id,
                session_id,
                ACCEPTED_VERSION,
            )
            return

        logger.info(
            "Received grading trigger  event_id=%s  session=%s  version=%s  "
            "topic=%s  partition=%s  offset=%s",
            event_id,
            session_id,
            version,
            msg.topic,
            msg.partition,
            msg.offset,
        )

        await process_incoming_event(payload, pool, candidate_client, producer)

    except json.JSONDecodeError as exc:
        logger.error(
            "Failed to decode Kafka message as JSON  "
            "topic=%s  partition=%s  offset=%s  error=%s",
            msg.topic,
            msg.partition,
            msg.offset,
            exc,
        )
    except ValidationError as exc:
        logger.error(
            "Event payload failed schema validation  session=%s  errors=%s",
            payload.get("session_id", "n/a") if isinstance(payload, dict) else "n/a",
            exc.errors(),
        )
    except CandidateServiceError as exc:
        logger.error(
            "Failed to fetch grading payload from candidate-service  session=%s  error=%s",
            payload.get("session_id", "n/a") if isinstance(payload, dict) else "n/a",
            exc,
        )
    except Exception as exc:
        logger.exception(
            "Unhandled error processing grading message  "
            "topic=%s  partition=%s  offset=%s  error=%s",
            msg.topic,
            msg.partition,
            msg.offset,
            exc,
        )
