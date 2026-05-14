"""
Kafka consumer — subscribes to candidate.exam.submitted and publishes
the final cheating score with is_final=True once the session is closed.

Runs as a background asyncio task during FastAPI lifespan.
"""
import asyncio
import json
import logging
from uuid import UUID

from aiokafka import AIOKafkaConsumer

from app.config import settings

logger = logging.getLogger("proctoring.consumer")


async def run_consumer(event_usecase, producer) -> None:
    """
    Long-running coroutine. Should be started as an asyncio task.
    Retries automatically on consumer errors with a 5-second backoff.
    """
    while True:
        consumer = AIOKafkaConsumer(
            "candidate.exam.submitted",
            bootstrap_servers=settings.KAFKA_BROKERS,
            group_id="proctoring-service-group",
            auto_offset_reset="earliest",
            enable_auto_commit=True,
        )
        try:
            await consumer.start()
            logger.info("Kafka consumer started — topic: candidate.exam.submitted")
            async for msg in consumer:
                await _handle_message(msg, event_usecase, producer)
        except asyncio.CancelledError:
            logger.info("Kafka consumer shutting down")
            break
        except Exception as exc:
            logger.error("Kafka consumer error: %s — retrying in 5s", exc)
            await asyncio.sleep(5)
        finally:
            try:
                await consumer.stop()
            except Exception:
                pass


async def _handle_message(msg, event_usecase, producer) -> None:
    try:
        payload = json.loads(msg.value)
        session_id = UUID(payload["session_id"])
        enterprise_id = payload.get("enterprise_id", "")

        score = await event_usecase.get_score(session_id)
        if score:
            await producer.publish("proctoring.cheating_score.updated", {
                "session_id": str(session_id),
                "enterprise_id": enterprise_id,
                "cheating_score": score.cheating_score,
                "event_count": score.event_count,
                "is_final": True,
            })
            logger.info(
                "Published final cheating score %.2f for session %s",
                score.cheating_score, session_id,
            )
    except Exception as exc:
        logger.error("Failed to process candidate.exam.submitted message: %s", exc)
