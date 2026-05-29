"""
Shared fixtures for the proctoring-service test suite.
"""
import os
import uuid
from datetime import datetime, timezone
from typing import Any, Dict
from unittest.mock import AsyncMock, MagicMock, patch

# pyrefly: ignore [missing-import]
import pytest

# ---------------------------------------------------------------------------
# Override settings BEFORE any app code imports them
# ---------------------------------------------------------------------------
os.environ.setdefault("PG_VERITAS_HOST", "localhost")
os.environ.setdefault("KAFKA_BROKERS", "localhost:9092")
os.environ.setdefault("REDIS_HOST", "localhost")
os.environ.setdefault("JWT_SECRET", "test-jwt-secret")
os.environ.setdefault("CANDIDATE_SERVICE_URL", "http://localhost:8084")
os.environ.setdefault("FACE_API_URL", "http://localhost:8000/face")


# ---------------------------------------------------------------------------
# Deterministic UUIDs for repeatable tests
# ---------------------------------------------------------------------------
SESSION_ID = str(uuid.UUID("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeee01"))
CANDIDATE_ID = str(uuid.UUID("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeee02"))
ENTERPRISE_ID = str(uuid.UUID("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeee03"))
EVENT_ID = str(uuid.UUID("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeee04"))


# ---------------------------------------------------------------------------
# FakeRecord for asyncpg mock returns
# ---------------------------------------------------------------------------
class FakeRecord(dict):
    """dict subclass that also supports attribute access like asyncpg.Record."""
    def __getattr__(self, key):
        try:
            return self[key]
        except KeyError:
            raise AttributeError(key)


@pytest.fixture
def mock_pool():
    """Return a mock asyncpg.Pool with fetch, fetchrow, fetchval and execute methods."""
    pool = MagicMock()
    conn = MagicMock()

    class AsyncContextManagerMock:
        def __init__(self, value):
            self.value = value
        async def __aenter__(self):
            return self.value
        async def __aexit__(self, exc_type, exc_val, exc_tb):
            return False

    pool.acquire = MagicMock(return_value=AsyncContextManagerMock(conn))
    conn.transaction = MagicMock(return_value=AsyncContextManagerMock(None))

    # Async methods on pool must be AsyncMock
    pool.fetchrow = AsyncMock()
    pool.fetch = AsyncMock()
    pool.execute = AsyncMock()
    pool.fetchval = AsyncMock()
    pool.close = AsyncMock()

    # Async methods on connection must be AsyncMock
    conn.fetchrow = AsyncMock()
    conn.fetch = AsyncMock()
    conn.execute = AsyncMock()
    conn.fetchval = AsyncMock()

    pool._conn = conn  # expose for test assertions
    return pool


@pytest.fixture
def mock_redis():
    """Return a mock Redis client."""
    client = AsyncMock()
    client.get = AsyncMock(return_value=None)
    client.set = AsyncMock(return_value=True)
    client.close = AsyncMock()
    return client


@pytest.fixture
def mock_kafka_producer():
    """Return a mock KafkaProducer wrapper."""
    producer = AsyncMock()
    producer.start = AsyncMock()
    producer.stop = AsyncMock()
    producer.publish = AsyncMock()
    return producer


@pytest.fixture
def mock_face_detector():
    """Return a mock FaceDetector port."""
    detector = AsyncMock()
    detector.detect = AsyncMock()
    detector.compare = AsyncMock()
    return detector


@pytest.fixture
def mock_candidate_client():
    """Return a mock CandidateServiceClient."""
    client = AsyncMock()
    client.get_face_reference_data = AsyncMock()
    return client
