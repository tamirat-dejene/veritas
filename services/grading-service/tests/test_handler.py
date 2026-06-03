"""
Unit tests for app.handler.grading_handler — FastAPI HTTP endpoints.

Uses fastapi.testclient.TestClient to exercise the routes without a real
database or Kafka connection (lifespan is replaced by a no-op).
"""
import uuid
import pytest
from app.domain.models import GradingStatus, QuestionGradingStatus, QuestionType
from datetime import datetime, timezone
from contextlib import asynccontextmanager
from unittest.mock import AsyncMock, MagicMock, patch

from fastapi import FastAPI
from fastapi.testclient import TestClient

from app.handler import grading_handler
from app.middleware.context import IdentityMiddleware
from app.repository.grading_repository import DataTamperingError
from tests.conftest import (
    SESSION_ID,
    EXAM_ID,
    CANDIDATE_ID,
    ENTERPRISE_ID,
    ENROLLMENT_ID,
    ACTOR_ID,
    QUESTION_ID_1,
    SQ_ID_1,
)


# ===================================================================
# Test app factory (no lifespan / DB / Kafka)
# ===================================================================

def _create_test_app(grading_uc: AsyncMock) -> FastAPI:
    """Minimal FastAPI app wired to the grading handler with a mock use case."""

    @asynccontextmanager
    async def _noop_lifespan(app: FastAPI):
        yield

    app = FastAPI(lifespan=_noop_lifespan)
    app.add_middleware(IdentityMiddleware)
    app.include_router(grading_handler.router)
    app.state.grading_uc = grading_uc
    return app


def _headers(
    enterprise_id: str = ENTERPRISE_ID,
    user_id: str = ACTOR_ID,
    user_role: str = "admin",
) -> dict:
    return {
        "X-Enterprise-ID": enterprise_id,
        "X-User-ID": user_id,
        "X-User-Role": user_role,
    }


# ===================================================================
# list_graded_students
# ===================================================================


class TestListGradedStudents:

    def _setup(self, return_value=([], 0)):
        uc = AsyncMock()
        uc.list_graded_students.return_value = return_value
        client = TestClient(_create_test_app(uc))
        return client, uc

    def test_returns_empty_list(self):
        client, _ = self._setup()
        resp = client.get("/grading/results", headers=_headers())
        assert resp.status_code == 200
        body = resp.json()
        assert body["results"] == []
        assert body["total"] == 0

    def test_missing_enterprise_id_401(self):
        client, _ = self._setup()
        resp = client.get("/grading/results")
        assert resp.status_code == 401

    def test_pagination_params(self):
        client, uc = self._setup()
        client.get(
            "/grading/results?limit=5&offset=10",
            headers=_headers(),
        )
        call_kwargs = uc.list_graded_students.call_args.kwargs
        assert call_kwargs["limit"] == 5
        assert call_kwargs["offset"] == 10


# ===================================================================
# get_grade_detail
# ===================================================================


class TestGetGradeDetail:

    def _setup(self, detail=None):
        uc = AsyncMock()
        uc.get_grade_detail.return_value = detail
        client = TestClient(_create_test_app(uc))
        return client, uc

    def test_not_found(self):
        client, _ = self._setup(None)
        resp = client.get(f"/grading/results/{SESSION_ID}", headers=_headers())
        assert resp.status_code == 404

    def test_wrong_enterprise_returns_403(self):
        detail = {
            "id": uuid.uuid4(),
            "session_id": uuid.UUID(SESSION_ID),
            "exam_id": uuid.UUID(EXAM_ID),
            "candidate_id": uuid.UUID(CANDIDATE_ID),
            "enterprise_id": uuid.uuid4(),  # different enterprise
            "enrollment_id": uuid.UUID(ENROLLMENT_ID),
            "total_max_points": 100,
            "total_awarded_points": 80,
            "percentage": 80.0,
            "status": GradingStatus.graded.value,
            "graded_by": "system",
            "is_tampered": False,
            "version": 1,
            "created_at": datetime.now(timezone.utc),
            "updated_at": datetime.now(timezone.utc),
            "question_results": [],
        }
        client, _ = self._setup(detail)
        resp = client.get(f"/grading/results/{SESSION_ID}", headers=_headers())
        assert resp.status_code == 403

    def test_success(self):
        detail = {
            "id": uuid.uuid4(),
            "session_id": uuid.UUID(SESSION_ID),
            "exam_id": uuid.UUID(EXAM_ID),
            "candidate_id": uuid.UUID(CANDIDATE_ID),
            "enterprise_id": uuid.UUID(ENTERPRISE_ID),
            "enrollment_id": uuid.UUID(ENROLLMENT_ID),
            "total_max_points": 100,
            "total_awarded_points": 85,
            "percentage": 85.0,
            "graded_by": {"id": "system", "type": "system"},
            "status": GradingStatus.graded.value,
            "is_tampered": False,
            "version": 1,
            "created_at": datetime.now(timezone.utc),
            "updated_at": datetime.now(timezone.utc),
            "question_results": [
                {
                    "question_id": uuid.UUID(QUESTION_ID_1),
                    "session_question_id": uuid.UUID(SQ_ID_1),
                    "question_type": QuestionType.MCQ,
                    "title": "Q1",
                    "content": "What is 2+2",
                    "max_points": 100.0,
                    "awarded_points": 85.0,
                    "status": QuestionGradingStatus.correct.value,
                }
            ],
        }
        client, _ = self._setup(detail)
        resp = client.get(f"/grading/results/{SESSION_ID}", headers=_headers())
        assert resp.status_code == 200
        body = resp.json()
        assert body["percentage"] == 85.0
        assert len(body["question_results"]) == 1

    def test_tamper_detected_returns_409(self):
        uc = AsyncMock()
        uc.get_grade_detail.side_effect = DataTamperingError(uuid.UUID(SESSION_ID))
        client = TestClient(_create_test_app(uc))
        resp = client.get(f"/grading/results/{SESSION_ID}", headers=_headers())
        assert resp.status_code == 409


# ===================================================================
# override_grade
# ===================================================================


class TestOverrideGrade:

    def _make_detail(self):
        return {
            "id": uuid.uuid4(),
            "session_id": uuid.UUID(SESSION_ID),
            "exam_id": uuid.UUID(EXAM_ID),
            "candidate_id": uuid.UUID(CANDIDATE_ID),
            "enterprise_id": uuid.UUID(ENTERPRISE_ID),
            "enrollment_id": uuid.UUID(ENROLLMENT_ID),
            "total_max_points": 100,
            "total_awarded_points": 75,
            "percentage": 75.0,
            "status": GradingStatus.graded.value,
            "graded_by": "system",
            "is_tampered": False,
            "version": 1,
            "created_at": datetime.now(timezone.utc),
            "updated_at": datetime.now(timezone.utc),
            "question_results": [],
        }

    def test_successful_override(self):
        uc = AsyncMock()
        uc.get_grade_detail.return_value = self._make_detail()
        uc.update_grade_manually.return_value = {
            "session_id": uuid.UUID(SESSION_ID),
            "previous_score": 75.0,
            "new_score": 90.0,
            "new_percentage": 90.0,
            "status": GradingStatus.reviewed.value,
        }
        client = TestClient(_create_test_app(uc))
        resp = client.post(
            f"/grading/results/{SESSION_ID}/override",
            json={"new_score": 90.0, "reason": "Correcting error in grading"},
            headers=_headers(),
        )
        assert resp.status_code == 200
        body = resp.json()
        assert body["new_score"] == 90.0
        assert body["status"] == GradingStatus.reviewed.value

    def test_not_found_returns_404(self):
        uc = AsyncMock()
        uc.get_grade_detail.return_value = None
        client = TestClient(_create_test_app(uc))
        resp = client.post(
            f"/grading/results/{SESSION_ID}/override",
            json={"new_score": 90.0, "reason": "Correcting error"},
            headers=_headers(),
        )
        assert resp.status_code == 404

    def test_invalid_body_returns_422(self):
        uc = AsyncMock()
        client = TestClient(_create_test_app(uc))
        resp = client.post(
            f"/grading/results/{SESSION_ID}/override",
            json={"new_score": -5.0, "reason": "Bad"},  # negative score, reason too short
            headers=_headers(),
        )
        assert resp.status_code == 422

    def test_tamper_detected_returns_409(self):
        uc = AsyncMock()
        uc.get_grade_detail.return_value = self._make_detail()
        uc.update_grade_manually.side_effect = DataTamperingError(uuid.UUID(SESSION_ID))
        client = TestClient(_create_test_app(uc))
        resp = client.post(
            f"/grading/results/{SESSION_ID}/override",
            json={"new_score": 90.0, "reason": "Correcting error in grading"},
            headers=_headers(),
        )
        assert resp.status_code == 409


# ===================================================================
# get_audit_logs
# ===================================================================


class TestGetAuditLogs:

    def _make_detail(self):
        return {
            "id": uuid.uuid4(),
            "session_id": uuid.UUID(SESSION_ID),
            "exam_id": uuid.UUID(EXAM_ID),
            "candidate_id": uuid.UUID(CANDIDATE_ID),
            "enterprise_id": uuid.UUID(ENTERPRISE_ID),
            "enrollment_id": uuid.UUID(ENROLLMENT_ID),
            "total_max_points": 100,
            "total_awarded_points": 75,
            "percentage": 75.0,
            "status": GradingStatus.graded.value,
            "graded_by": "system",
            "is_tampered": False,
            "version": 1,
            "created_at": datetime.now(timezone.utc),
            "updated_at": datetime.now(timezone.utc),
            "question_results": [],
        }

    def test_returns_audit_logs(self):
        uc = AsyncMock()
        uc.get_grade_detail.return_value = self._make_detail()
        uc.get_audit_logs.return_value = [
            {
                "id": uuid.uuid4(),
                "action": "UPDATE",
                "actor_id": uuid.UUID(ACTOR_ID),
                "actor_role": "admin",
                "old_values": {"total_awarded_points": 75.0},
                "new_values": {"total_awarded_points": 90.0},
                "changed_fields": ["total_awarded_points"],
                "ip_address": "127.0.0.1",
                "reason": "Corrected",
                "created_at": datetime.now(timezone.utc),
            }
        ]
        client = TestClient(_create_test_app(uc))
        resp = client.get(
            f"/grading/results/{SESSION_ID}/logs",
            headers=_headers(),
        )
        assert resp.status_code == 200
        body = resp.json()
        assert len(body) == 1
        assert body[0]["action"] == "UPDATE"

    def test_not_found_returns_404(self):
        uc = AsyncMock()
        uc.get_grade_detail.return_value = None
        client = TestClient(_create_test_app(uc))
        resp = client.get(
            f"/grading/results/{SESSION_ID}/logs",
            headers=_headers(),
        )
        assert resp.status_code == 404


class TestValidationExceptionHandler:
    @pytest.mark.anyio
    async def test_validation_exception_handler_logging(self):
        from app.router import create_app
        from fastapi.exceptions import RequestValidationError
        from unittest.mock import patch, MagicMock, AsyncMock

        app = create_app()
        # Find the registered exception handler for RequestValidationError
        handler = app.exception_handlers.get(RequestValidationError)
        assert handler is not None

        mock_request = MagicMock()
        mock_request.method = "POST"
        mock_request.url.path = "/grading/results/some-uuid/override"
        
        mock_exc = MagicMock(spec=RequestValidationError)
        mock_exc.errors.return_value = [{"loc": ["body", "new_score"], "msg": "value is not a valid float"}]

        with patch("app.router.logger.error") as mock_log_error:
            # We mock request_validation_exception_handler since we don't want to actually execute the response construction
            with patch("app.router.request_validation_exception_handler", new_callable=AsyncMock) as mock_default_handler:
                # pyrefly: ignore [not-async]
                await handler(mock_request, mock_exc)
                mock_log_error.assert_called_once_with(
                    "Request validation failed for %s %s: %s",
                    "POST",
                    "/grading/results/some-uuid/override",
                    mock_exc.errors(),
                )
                mock_default_handler.assert_called_once_with(mock_request, mock_exc)
