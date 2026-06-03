"""
Unit tests for app.usecase.grading_usecase — verifies delegation to the repository.
"""
import uuid
import pytest
from app.domain.models import GradingStatus, QuestionGradingStatus, QuestionType
from unittest.mock import AsyncMock, MagicMock

from app.usecase.grading_usecase import GradingUseCase
from tests.conftest import SESSION_ID, ENTERPRISE_ID, ACTOR_ID


class TestGradingUseCase:
    """Each usecase method is a thin delegate — verify correct routing."""

    @pytest.fixture
    def mock_repo(self):
        repo = AsyncMock()
        return repo

    @pytest.fixture
    def mock_candidate_client(self):
        client = AsyncMock()
        client.fetch_candidate = AsyncMock(return_value=None)
        return client

    @pytest.fixture
    def mock_enterprise_client(self):
        client = AsyncMock()
        client.fetch_user = AsyncMock(return_value=None)
        return client

    @pytest.fixture
    def usecase(self, mock_repo, mock_candidate_client, mock_enterprise_client):
        return GradingUseCase(mock_repo, mock_candidate_client, mock_enterprise_client)

    # --- list_graded_students ---

    @pytest.mark.asyncio
    async def test_list_graded_students_delegates(self, usecase, mock_repo):
        mock_repo.list_graded_students.return_value = ([], 0)
        results, total = await usecase.list_graded_students(
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            exam_id=None,
            limit=10,
            offset=0,
        )
        mock_repo.list_graded_students.assert_awaited_once_with(
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            exam_id=None,
            limit=10,
            offset=0,
        )
        assert results == []
        assert total == 0

    # --- get_grade_detail ---

    @pytest.mark.asyncio
    async def test_get_grade_detail_delegates(self, usecase, mock_repo):
        mock_repo.get_by_session.return_value = {"session_id": SESSION_ID}
        result = await usecase.get_grade_detail(uuid.UUID(SESSION_ID))
        mock_repo.get_by_session.assert_awaited_once_with(uuid.UUID(SESSION_ID))
        assert result["session_id"] == SESSION_ID

    @pytest.mark.asyncio
    async def test_get_grade_detail_not_found(self, usecase, mock_repo):
        mock_repo.get_by_session.return_value = None
        result = await usecase.get_grade_detail(uuid.UUID(SESSION_ID))
        assert result is None

    # --- update_grade_manually ---

    @pytest.mark.asyncio
    async def test_update_grade_manually_delegates(self, usecase, mock_repo):
        mock_repo.update_grade_manually.return_value = {
            "session_id": uuid.UUID(SESSION_ID),
            "previous_score": 75.0,
            "new_score": 90.0,
            "new_percentage": 90.0,
            "status": GradingStatus.reviewed.value,
        }
        result = await usecase.update_grade_manually(
            session_id=uuid.UUID(SESSION_ID),
            new_score=90.0,
            actor_id=uuid.UUID(ACTOR_ID),
            actor_role="admin",
            reason="Fix",
            ip_address="127.0.0.1",
        )
        mock_repo.update_grade_manually.assert_awaited_once_with(
            session_id=uuid.UUID(SESSION_ID),
            new_awarded_points=90.0,
            actor_id=uuid.UUID(ACTOR_ID),
            actor_role="admin",
            reason="Fix",
            ip_address="127.0.0.1",
        )
        assert result["new_score"] == 90.0

    # --- get_audit_logs ---

    @pytest.mark.asyncio
    async def test_get_audit_logs_delegates(self, usecase, mock_repo):
        mock_repo.get_audit_logs.return_value = [{"action": "INSERT"}]
        logs = await usecase.get_audit_logs(uuid.UUID(SESSION_ID))
        mock_repo.get_audit_logs.assert_awaited_once_with(uuid.UUID(SESSION_ID))
        assert len(logs) == 1

    @pytest.mark.asyncio
    async def test_get_grade_detail_enriches_candidate_and_grader(
        self, usecase, mock_repo, mock_candidate_client, mock_enterprise_client
    ):
        mock_repo.get_by_session.return_value = {
            "session_id": SESSION_ID,
            "candidate_id": "candidate-123",
            "enterprise_id": "enterprise-123",
            "graded_by": {"id": "user-456", "type": "human"},
        }
        mock_candidate_client.fetch_candidate.return_value = {
            "id": "candidate-123",
            "firstName": "John",
            "lastName": "Doe",
            "email": "john@example.com",
        }
        mock_enterprise_client.fetch_user.return_value = {
            "id": "user-456",
            "firstName": "Jane",
            "lastName": "Smith",
            "email": "jane@example.com",
            "role": "enterprise_admin",
        }

        result = await usecase.get_grade_detail(uuid.UUID(SESSION_ID))
        
        assert result["candidate_info"] == {
            "id": "candidate-123",
            "first_name": "John",
            "last_name": "Doe",
            "email": "john@example.com",
        }
        assert result["graded_by"]["user_details"] == {
            "id": "user-456",
            "first_name": "Jane",
            "last_name": "Smith",
            "email": "jane@example.com",
            "role": "enterprise_admin",
        }
