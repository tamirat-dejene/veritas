"""
Unit tests for app.repository.grading_repository — data layer with mocked asyncpg.
"""
import uuid
import pytest
from datetime import datetime, timezone
from unittest.mock import AsyncMock, MagicMock, patch

from app.repository.grading_repository import GradingRepository, DataTamperingError
from app.grading.grader import ExamGradeReport, QuestionResult
from app.grading.security import calculate_row_checksum
from tests.conftest import (
    SESSION_ID,
    EXAM_ID,
    CANDIDATE_ID,
    ENTERPRISE_ID,
    ENROLLMENT_ID,
    EVENT_ID,
    ACTOR_ID,
    QUESTION_ID_1,
    SQ_ID_1,
    FakeRecord,
)


# ===================================================================
# Helpers
# ===================================================================


def _sample_report(**overrides) -> ExamGradeReport:
    """Build a minimal ExamGradeReport for testing persistence."""
    defaults = dict(
        event_id=EVENT_ID,
        session_id=SESSION_ID,
        candidate_id=CANDIDATE_ID,
        exam_id=EXAM_ID,
        enterprise_id=ENTERPRISE_ID,
        enrollment_id=ENROLLMENT_ID,
        total_max_points=100.0,
        total_awarded_points=75.0,
        question_results=[
            QuestionResult(
                question_id=QUESTION_ID_1,
                session_question_id=SQ_ID_1,
                question_type="multiple_choice",
                title="Q1",
                content="What is Python?",
                candidate_answer={"selectedOptionIds": ["opt_1"]},
                max_points=100.0,
                awarded_points=75.0,
                status="correct",
            )
        ],
    )
    defaults.update(overrides)
    return ExamGradeReport(**defaults)


def _make_db_row(
    version: int = 1,
    total_max_points: float = 100.0,
    total_awarded_points: float = 75.0,
    percentage: float = 75.0,
    valid_checksum: bool = True,
) -> FakeRecord:
    """Build a FakeRecord mimicking an asyncpg result row."""
    checksum = calculate_row_checksum(
        session_id=SESSION_ID,
        candidate_id=CANDIDATE_ID,
        total_max_points=total_max_points,
        total_awarded_points=total_awarded_points,
        percentage=percentage,
        version=version,
    )
    if not valid_checksum:
        checksum = "tampered_" + checksum[9:]
    return FakeRecord(
        id=uuid.uuid4(),
        session_id=uuid.UUID(SESSION_ID),
        exam_id=uuid.UUID(EXAM_ID),
        candidate_id=uuid.UUID(CANDIDATE_ID),
        enterprise_id=uuid.UUID(ENTERPRISE_ID),
        enrollment_id=uuid.UUID(ENROLLMENT_ID),
        total_max_points=total_max_points,
        total_awarded_points=total_awarded_points,
        percentage=percentage,
        status="graded",
        graded_by="system",
        row_checksum=checksum,
        version=version,
        created_at=datetime.now(timezone.utc),
        updated_at=datetime.now(timezone.utc),
    )


# ===================================================================
# save_grading_report
# ===================================================================


class TestSaveGradingReport:
    """Tests for saving exam grading reports."""

    @pytest.mark.asyncio
    async def test_insert_new_report(self, mock_pool):
        conn = mock_pool._conn
        conn.fetchrow.return_value = None  # no existing record
        new_id = uuid.uuid4()
        conn.fetchval.return_value = new_id

        repo = GradingRepository(mock_pool)
        result_id = await repo.save_grading_report(_sample_report())

        assert result_id == new_id
        # Should have called fetchval (INSERT) and execute (question results)
        conn.fetchval.assert_called_once()
        conn.execute.assert_called()

    @pytest.mark.asyncio
    async def test_update_existing_report(self, mock_pool):
        conn = mock_pool._conn
        existing_row = _make_db_row(version=1)
        conn.fetchrow.return_value = existing_row
        conn.execute.return_value = "UPDATE 1"

        repo = GradingRepository(mock_pool)
        result_id = await repo.save_grading_report(_sample_report())

        assert result_id == existing_row["id"]

    @pytest.mark.asyncio
    async def test_tampered_existing_row_raises(self, mock_pool):
        conn = mock_pool._conn
        tampered_row = _make_db_row(valid_checksum=False)
        conn.fetchrow.return_value = tampered_row

        repo = GradingRepository(mock_pool)
        with pytest.raises(DataTamperingError):
            await repo.save_grading_report(_sample_report())

    @pytest.mark.asyncio
    async def test_optimistic_lock_conflict_raises(self, mock_pool):
        conn = mock_pool._conn
        conn.fetchrow.return_value = _make_db_row(version=1)
        conn.execute.return_value = "UPDATE 0"  # version conflict

        repo = GradingRepository(mock_pool)
        with pytest.raises(RuntimeError, match="Optimistic locking conflict"):
            await repo.save_grading_report(_sample_report())


# ===================================================================
# list_graded_students
# ===================================================================


class TestListGradedStudents:

    @pytest.mark.asyncio
    async def test_empty_result(self, mock_pool):
        mock_pool.fetchval.return_value = 0

        repo = GradingRepository(mock_pool)
        results, total = await repo.list_graded_students(
            enterprise_id=uuid.UUID(ENTERPRISE_ID)
        )
        assert results == []
        assert total == 0

    @pytest.mark.asyncio
    async def test_returns_results_with_tamper_flag(self, mock_pool):
        mock_pool.fetchval.return_value = 1
        row = _make_db_row()
        mock_pool.fetch.return_value = [row]

        repo = GradingRepository(mock_pool)
        results, total = await repo.list_graded_students(
            enterprise_id=uuid.UUID(ENTERPRISE_ID)
        )
        assert total == 1
        assert len(results) == 1
        assert results[0]["is_tampered"] is False

    @pytest.mark.asyncio
    async def test_flags_tampered_row(self, mock_pool):
        mock_pool.fetchval.return_value = 1
        row = _make_db_row(valid_checksum=False)
        mock_pool.fetch.return_value = [row]

        repo = GradingRepository(mock_pool)
        results, _ = await repo.list_graded_students(
            enterprise_id=uuid.UUID(ENTERPRISE_ID)
        )
        assert results[0]["is_tampered"] is True


# ===================================================================
# get_by_session
# ===================================================================


class TestGetBySession:

    @pytest.mark.asyncio
    async def test_not_found(self, mock_pool):
        mock_pool.fetchrow.return_value = None
        repo = GradingRepository(mock_pool)
        result = await repo.get_by_session(uuid.UUID(SESSION_ID))
        assert result is None

    @pytest.mark.asyncio
    async def test_returns_record_with_questions(self, mock_pool):
        row = _make_db_row()
        mock_pool.fetchrow.return_value = row
        mock_pool.fetch.return_value = [
            FakeRecord(
                question_id=uuid.UUID(QUESTION_ID_1),
                session_question_id=uuid.UUID(SQ_ID_1),
                question_type="multiple_choice",
                title="Q1",
                max_points=10.0,
                awarded_points=10.0,
                status="correct",
            )
        ]

        repo = GradingRepository(mock_pool)
        result = await repo.get_by_session(uuid.UUID(SESSION_ID))

        assert result is not None
        assert result["is_tampered"] is False
        assert len(result["question_results"]) == 1


# ===================================================================
# update_grade_manually
# ===================================================================


class TestUpdateGradeManually:

    @pytest.mark.asyncio
    async def test_successful_override(self, mock_pool):
        conn = mock_pool._conn
        row = _make_db_row(version=1, total_max_points=100.0, total_awarded_points=75.0, percentage=75.0)
        conn.fetchrow.return_value = row
        conn.execute.return_value = "UPDATE 1"

        repo = GradingRepository(mock_pool)
        result = await repo.update_grade_manually(
            session_id=uuid.UUID(SESSION_ID),
            new_awarded_points=90.0,
            actor_id=uuid.UUID(ACTOR_ID),
            actor_role="admin",
            reason="Correcting score",
        )

        assert result["previous_score"] == 75.0
        assert result["new_score"] == 90.0
        assert result["new_percentage"] == 90.0
        assert result["status"] == "reviewed"

    @pytest.mark.asyncio
    async def test_not_found_raises(self, mock_pool):
        conn = mock_pool._conn
        conn.fetchrow.return_value = None

        repo = GradingRepository(mock_pool)
        with pytest.raises(ValueError, match="not found"):
            await repo.update_grade_manually(
                session_id=uuid.UUID(SESSION_ID),
                new_awarded_points=90.0,
                actor_id=uuid.UUID(ACTOR_ID),
                actor_role="admin",
                reason="Test",
            )

    @pytest.mark.asyncio
    async def test_tampered_row_blocks_update(self, mock_pool):
        conn = mock_pool._conn
        conn.fetchrow.return_value = _make_db_row(valid_checksum=False)

        repo = GradingRepository(mock_pool)
        with pytest.raises(DataTamperingError):
            await repo.update_grade_manually(
                session_id=uuid.UUID(SESSION_ID),
                new_awarded_points=90.0,
                actor_id=uuid.UUID(ACTOR_ID),
                actor_role="admin",
                reason="Test",
            )


# ===================================================================
# get_audit_logs
# ===================================================================


class TestGetAuditLogs:

    @pytest.mark.asyncio
    async def test_no_grading_result_returns_empty(self, mock_pool):
        mock_pool.fetchval.return_value = None
        repo = GradingRepository(mock_pool)
        logs = await repo.get_audit_logs(uuid.UUID(SESSION_ID))
        assert logs == []

    @pytest.mark.asyncio
    async def test_returns_audit_rows(self, mock_pool):
        grading_id = uuid.uuid4()
        mock_pool.fetchval.return_value = grading_id
        mock_pool.fetch.return_value = [
            FakeRecord(
                id=uuid.uuid4(),
                action="UPDATE",
                actor_id=uuid.UUID(ACTOR_ID),
                actor_role="admin",
                old_values={"total_awarded_points": 75.0},
                new_values={"total_awarded_points": 90.0},
                changed_fields=["total_awarded_points"],
                ip_address="127.0.0.1",
                reason="Correction",
                created_at=datetime.now(timezone.utc),
            )
        ]

        repo = GradingRepository(mock_pool)
        logs = await repo.get_audit_logs(uuid.UUID(SESSION_ID))
        assert len(logs) == 1
        assert logs[0]["action"] == "UPDATE"
