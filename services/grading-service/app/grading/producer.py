"""
Kafka producer wrapper — thin async wrapper around aiokafka AIOKafkaProducer.
Publishes JSON-encoded messages to the configured broker.
"""
import json
from typing import Any

from aiokafka import AIOKafkaProducer

from app.config import settings


class KafkaProducer:
    def __init__(self):
        self._producer: AIOKafkaProducer | None = None

    async def start(self) -> None:
        self._producer = AIOKafkaProducer(
            bootstrap_servers=settings.KAFKA_BROKERS,
            value_serializer=lambda v: json.dumps(v, default=str).encode("utf-8"),
        )
        await self._producer.start()

    async def stop(self) -> None:
        if self._producer:
            await self._producer.stop()

    async def publish(self, topic: str, payload: dict[str, Any]) -> None:
        """Publish a JSON payload to the given Kafka topic."""
        if self._producer is None:
            raise RuntimeError("Kafka producer is not started")
        await self._producer.send_and_wait(topic, payload)
