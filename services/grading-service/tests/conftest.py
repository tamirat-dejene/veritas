"""
Shared fixtures for the grading-service test suite.
"""
import os
import uuid
from datetime import datetime, timezone
from typing import Any, Dict, List
from unittest.mock import AsyncMock, MagicMock, patch

# pyrefly: ignore [missing-import]
import pytest 

# ---------------------------------------------------------------------------
# Override settings BEFORE any app code imports them
# ---------------------------------------------------------------------------
os.environ.setdefault("PG_VERITAS_HOST", "localhost")
os.environ.setdefault("KAFKA_BROKERS", "localhost:9092")
os.environ.setdefault("GRADING_SECRET_KEY", "test-secret-key-for-unit-tests")
os.environ.setdefault("JWT_SECRET", "test-jwt-secret")
os.environ.setdefault("HF_EVALUATE_URL", "https://fake-hf-space.test/evaluate")
os.environ.setdefault("HF_TIMEOUT_SECONDS", "5")


# ---------------------------------------------------------------------------
# Deterministic UUIDs for repeatable tests
# ---------------------------------------------------------------------------
SESSION_ID = str(uuid.UUID("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeee01"))
EXAM_ID = str(uuid.UUID("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeee02"))
CANDIDATE_ID = str(uuid.UUID("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeee03"))
ENTERPRISE_ID = str(uuid.UUID("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeee04"))
ENROLLMENT_ID = str(uuid.UUID("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeee05"))
EVENT_ID = str(uuid.UUID("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeee06"))
ACTOR_ID = str(uuid.UUID("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeee07"))
QUESTION_ID_1 = str(uuid.UUID("11111111-1111-1111-1111-111111111111"))
QUESTION_ID_2 = str(uuid.UUID("22222222-2222-2222-2222-222222222222"))
QUESTION_ID_3 = str(uuid.UUID("33333333-3333-3333-3333-333333333333"))
SQ_ID_1 = str(uuid.UUID("44444444-4444-4444-4444-444444444441"))
SQ_ID_2 = str(uuid.UUID("44444444-4444-4444-4444-444444444442"))
SQ_ID_3 = str(uuid.UUID("44444444-4444-4444-4444-444444444443"))


# ---------------------------------------------------------------------------
# Event payload fixtures
# ---------------------------------------------------------------------------

def _base_event(**overrides) -> Dict[str, Any]:
    """Build a minimal valid ``exam.session.ready_for_grading`` payload."""
    event: Dict[str, Any] = {
        "event_id": EVENT_ID,
        "event_type": "exam.session.ready_for_grading",
        "version": "3.0",
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "enterprise_id": ENTERPRISE_ID,
        "exam_id": EXAM_ID,
        "session_id": SESSION_ID,
        "candidate_id": CANDIDATE_ID,
        "enrollment_id": ENROLLMENT_ID,
        "status": "submitted",
        "started_at": datetime.now(timezone.utc).isoformat(),
        "submitted_at": datetime.now(timezone.utc).isoformat(),
        "items": [],
    }
    event.update(overrides)
    return event


def _mcq_item(
    question_id: str = QUESTION_ID_1,
    session_question_id: str = SQ_ID_1,
    *,
    points: float = 10.0,
    negative_points: float = 0.0,
    correct_ids: List[str] | None = None,
    selected_ids: List[str] | None = None,
    has_answer: bool = True,
) -> Dict[str, Any]:
    """Build a single MCQ item for event payloads."""
    correct = correct_ids or ["opt-a", "opt-b"]
    item: Dict[str, Any] = {
        "question_id": question_id,
        "session_question_id": session_question_id,
        "question_type": "multiple_choice",
        "content": "Pick the right options",
        "title": "MCQ Question",
        "topic": "general",
        "points": points,
        "negative_points": negative_points,
        "correct_option_ids": correct,
        "has_answer": has_answer,
    }
    if has_answer and selected_ids is not None:
        item["candidate_answer"] = {"selectedOptionIds": selected_ids}
    elif has_answer:
        item["candidate_answer"] = {"selectedOptionIds": correct}
    return item


def _sa_item(
    question_id: str = QUESTION_ID_2,
    session_question_id: str = SQ_ID_2,
    *,
    points: float = 20.0,
    student_text: str | None = "The mitochondria is the powerhouse of the cell.",
    expected_answer: str = "The mitochondria is the powerhouse of the cell.",
    keywords: List[str] | None = None,
    has_answer: bool = True,
) -> Dict[str, Any]:
    """Build a single short-answer item for event payloads."""
    item: Dict[str, Any] = {
        "question_id": question_id,
        "session_question_id": session_question_id,
        "question_type": "short_answer",
        "content": "Describe the mitochondria",
        "title": "SA Question",
        "topic": "biology",
        "points": points,
        "expected_answer": expected_answer,
        "evaluation_criteria": {"keywords": keywords or ["mitochondria", "powerhouse"]},
        "has_answer": has_answer,
    }
    if has_answer and student_text is not None:
        item["candidate_answer"] = {"text": student_text}
    return item


@pytest.fixture
def base_event():
    return _base_event


@pytest.fixture
def mcq_item():
    return _mcq_item


@pytest.fixture
def sa_item():
    return _sa_item


# ---------------------------------------------------------------------------
# Mock asyncpg pool & connection
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
    """Return a mock asyncpg.Pool with a mock connection context manager."""
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

    # Async methods on connection must be AsyncMock
    conn.fetchrow = AsyncMock()
    conn.fetch = AsyncMock()
    conn.execute = AsyncMock()
    conn.fetchval = AsyncMock()

    pool._conn = conn  # expose for test assertions
    return pool
