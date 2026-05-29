"""
Unit tests for the data access layer in proctoring-service.
"""
import uuid
from datetime import datetime, timezone

# pyrefly: ignore [missing-import]
import pytest

from app.domain.enums import EventType, Severity
from app.repository.event_repository import EventRepository
from app.repository.score_repository import ScoreRepository
from tests.conftest import CANDIDATE_ID, ENTERPRISE_ID, EVENT_ID, SESSION_ID, FakeRecord


# ===================================================================
# EventRepository Tests
# ===================================================================

class TestEventRepository:

    @pytest.mark.asyncio
    async def test_create_event_success(self, mock_pool):
        # Mock database row returned by fetchrow
        mock_row = FakeRecord(
            id=uuid.UUID(EVENT_ID),
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            event_type=EventType.TAB_SWITCH.value,
            severity=Severity.MEDIUM.value,
            metadata='{"prev_tab": "exam", "next_tab": "google"}',
            occurred_at=datetime.now(timezone.utc),
            created_at=datetime.now(timezone.utc),
        )
        mock_pool.fetchrow.return_value = mock_row

        repo = EventRepository(mock_pool)
        event = await repo.create(
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            event_type=EventType.TAB_SWITCH.value,
            severity=Severity.MEDIUM.value,
            metadata={"prev_tab": "exam", "next_tab": "google"},
            occurred_at=mock_row["occurred_at"],
        )

        assert event.id == uuid.UUID(EVENT_ID)
        assert event.event_type == EventType.TAB_SWITCH
        assert event.severity == Severity.MEDIUM
        assert event.metadata == {"prev_tab": "exam", "next_tab": "google"}
        assert event.occurred_at == mock_row["occurred_at"]
        mock_pool.fetchrow.assert_called_once()

    @pytest.mark.asyncio
    async def test_list_by_session(self, mock_pool):
        mock_row = FakeRecord(
            id=uuid.UUID(EVENT_ID),
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            event_type=EventType.MOUSE_INACTIVE.value,
            severity=Severity.LOW.value,
            metadata={},
            occurred_at=datetime.now(timezone.utc),
            created_at=datetime.now(timezone.utc),
        )
        mock_pool.fetch.return_value = [mock_row]

        repo = EventRepository(mock_pool)
        events = await repo.list_by_session(uuid.UUID(SESSION_ID))

        assert len(events) == 1
        assert events[0].id == uuid.UUID(EVENT_ID)
        assert events[0].event_type == EventType.MOUSE_INACTIVE
        mock_pool.fetch.assert_called_once_with(
            "SELECT * FROM proctoring_events WHERE session_id = $1 ORDER BY occurred_at ASC",
            uuid.UUID(SESSION_ID),
        )

    @pytest.mark.asyncio
    async def test_count_by_session(self, mock_pool):
        mock_pool.fetchrow.return_value = FakeRecord(cnt=42)

        repo = EventRepository(mock_pool)
        count = await repo.count_by_session(uuid.UUID(SESSION_ID))

        assert count == 42
        mock_pool.fetchrow.assert_called_once_with(
            "SELECT COUNT(*) AS cnt FROM proctoring_events WHERE session_id = $1",
            uuid.UUID(SESSION_ID),
        )


# ===================================================================
# ScoreRepository Tests
# ===================================================================

class TestScoreRepository:

    @pytest.mark.asyncio
    async def test_upsert_success(self, mock_pool):
        mock_row = FakeRecord(
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            cheating_score=12.5,
            event_count=3,
            last_computed_at=datetime.now(timezone.utc),
        )
        mock_pool.fetchrow.return_value = mock_row

        repo = ScoreRepository(mock_pool)
        score = await repo.upsert(
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            cheating_score=12.5,
            event_count=3,
        )

        assert score.session_id == uuid.UUID(SESSION_ID)
        assert score.cheating_score == 12.5
        assert score.event_count == 3
        mock_pool.fetchrow.assert_called_once()

    @pytest.mark.asyncio
    async def test_get_by_session_found(self, mock_pool):
        mock_row = FakeRecord(
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            cheating_score=0.0,
            event_count=0,
            last_computed_at=datetime.now(timezone.utc),
        )
        mock_pool.fetchrow.return_value = mock_row

        repo = ScoreRepository(mock_pool)
        score = await repo.get_by_session(uuid.UUID(SESSION_ID))

        assert score is not None
        assert score.session_id == uuid.UUID(SESSION_ID)
        assert score.cheating_score == 0.0
        mock_pool.fetchrow.assert_called_once_with(
            "SELECT * FROM proctoring_session_scores WHERE session_id = $1",
            uuid.UUID(SESSION_ID),
        )

    @pytest.mark.asyncio
    async def test_get_by_session_not_found(self, mock_pool):
        mock_pool.fetchrow.return_value = None

        repo = ScoreRepository(mock_pool)
        score = await repo.get_by_session(uuid.UUID(SESSION_ID))

        assert score is None
