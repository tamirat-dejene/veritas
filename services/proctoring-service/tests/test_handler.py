"""
Unit tests for the FastAPI HTTP endpoints in proctoring-service.
"""
import uuid
from contextlib import asynccontextmanager
from datetime import datetime, timezone
from unittest.mock import AsyncMock

# pyrefly: ignore [missing-import]
import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from app.domain.enums import EventType, Severity
from app.domain.errors import FaceNotRegisteredError, InternalServiceError, NoClearFaceError
from app.domain.models import FaceVerifyResponse, ProctoringEvent, SessionScore
from app.handler import event_handler, face_handler, health_handler
from app.middleware.context import IdentityMiddleware, get_candidate_id, get_enterprise_id
from tests.conftest import CANDIDATE_ID, ENTERPRISE_ID, EVENT_ID, SESSION_ID


def _create_test_app(event_uc=None, face_uc=None) -> FastAPI:
    @asynccontextmanager
    async def _noop_lifespan(app: FastAPI):
        yield

    app = FastAPI(lifespan=_noop_lifespan)
    app.add_middleware(IdentityMiddleware)
    app.include_router(health_handler.router)
    app.include_router(event_handler.router)
    app.include_router(face_handler.router)
    app.state.event_uc = event_uc
    app.state.face_uc = face_uc
    return app


def _headers(
    candidate_id: str = CANDIDATE_ID,
    enterprise_id: str = ENTERPRISE_ID,
) -> dict:
    return {
        "X-Subject-Id": candidate_id,
        "X-Enterprise-Id": enterprise_id,
    }


# ===================================================================
# health_handler Tests
# ===================================================================

def test_health_check():
    client = TestClient(_create_test_app())
    resp = client.get("/health")
    assert resp.status_code == 200
    assert resp.json() == {"status": "OK", "service": "proctoring-service"}


# ===================================================================
# event_handler Tests
# ===================================================================

class TestIngestEvent:

    def test_success(self):
        event_uc = AsyncMock()
        event_mock = ProctoringEvent(
            id=uuid.UUID(EVENT_ID),
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            event_type=EventType.TAB_SWITCH,
            severity=Severity.MEDIUM,
            metadata={"prev_tab": "exam", "next_tab": "google"},
            occurred_at=datetime.now(timezone.utc),
            created_at=datetime.now(timezone.utc),
        )
        score_mock = SessionScore(
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            cheating_score=15.0,
            event_count=3,
            last_computed_at=datetime.now(timezone.utc),
        )
        event_uc.ingest_event.return_value = (event_mock, score_mock)

        client = TestClient(_create_test_app(event_uc=event_uc))
        payload = {
            "session_id": SESSION_ID,
            "event_type": "tab_switch",
            "occurred_at": datetime.now(timezone.utc).isoformat(),
            "metadata": {"prev_tab": "exam", "next_tab": "google"},
        }
        resp = client.post("/proctoring/events", json=payload, headers=_headers())
        assert resp.status_code == 201
        body = resp.json()
        assert body["message"] == "Event recorded"
        assert body["event_id"] == EVENT_ID
        assert body["cheating_score"] == 15.0
        event_uc.ingest_event.assert_called_once()

    def test_missing_identity_headers_returns_401(self):
        event_uc = AsyncMock()
        client = TestClient(_create_test_app(event_uc=event_uc))
        payload = {
            "session_id": SESSION_ID,
            "event_type": "tab_switch",
            "occurred_at": datetime.now(timezone.utc).isoformat(),
            "metadata": {},
        }
        resp = client.post("/proctoring/events", json=payload)
        assert resp.status_code == 401
        event_uc.ingest_event.assert_not_called()

    def test_usecase_exception_returns_500(self):
        event_uc = AsyncMock()
        event_uc.ingest_event.side_effect = RuntimeError("Database error")
        client = TestClient(_create_test_app(event_uc=event_uc))
        payload = {
            "session_id": SESSION_ID,
            "event_type": "tab_switch",
            "occurred_at": datetime.now(timezone.utc).isoformat(),
            "metadata": {},
        }
        resp = client.post("/proctoring/events", json=payload, headers=_headers())
        assert resp.status_code == 500
        assert resp.json() == {"detail": "Failed to ingest event"}


class TestListEvents:

    def test_success(self):
        event_uc = AsyncMock()
        event_mock = ProctoringEvent(
            id=uuid.UUID(EVENT_ID),
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            event_type=EventType.TAB_SWITCH,
            severity=Severity.MEDIUM,
            metadata={},
            occurred_at=datetime.now(timezone.utc),
            created_at=datetime.now(timezone.utc),
        )
        event_uc.list_events.return_value = [event_mock]

        client = TestClient(_create_test_app(event_uc=event_uc))
        resp = client.get(f"/proctoring/sessions/{SESSION_ID}/events", headers=_headers())
        assert resp.status_code == 200
        body = resp.json()
        assert body["session_id"] == SESSION_ID
        assert len(body["events"]) == 1
        assert body["total"] == 1
        event_uc.list_events.assert_called_once_with(uuid.UUID(SESSION_ID))


class TestGetScore:

    def test_success(self):
        event_uc = AsyncMock()
        score_mock = SessionScore(
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            cheating_score=25.5,
            event_count=5,
            last_computed_at=datetime.now(timezone.utc),
        )
        event_uc.get_score.return_value = score_mock

        client = TestClient(_create_test_app(event_uc=event_uc))
        resp = client.get(f"/proctoring/sessions/{SESSION_ID}/score", headers=_headers())
        assert resp.status_code == 200
        body = resp.json()
        assert body["session_id"] == SESSION_ID
        assert body["cheating_score"] == 25.5
        assert body["event_count"] == 5
        event_uc.get_score.assert_called_once_with(uuid.UUID(SESSION_ID))

    def test_not_found_returns_404(self):
        event_uc = AsyncMock()
        event_uc.get_score.return_value = None

        client = TestClient(_create_test_app(event_uc=event_uc))
        resp = client.get(f"/proctoring/sessions/{SESSION_ID}/score", headers=_headers())
        assert resp.status_code == 404
        assert resp.json() == {"detail": "No proctoring data found for this session"}


# ===================================================================
# face_handler Tests
# ===================================================================

class TestVerifyFace:

    def test_success(self):
        face_uc = AsyncMock()
        face_uc.verify_face.return_value = FaceVerifyResponse(
            session_id=uuid.UUID(SESSION_ID),
            is_match=True,
            confidence=0.92,
            face_count=1,
        )

        client = TestClient(_create_test_app(face_uc=face_uc))
        payload = {
            "session_id": SESSION_ID,
            "image_b64": "data:image/jpeg;base64,abcdef",
        }
        resp = client.post("/face/verify", json=payload, headers=_headers())
        assert resp.status_code == 200
        body = resp.json()
        assert body["is_match"] is True
        assert body["confidence"] == 0.92
        assert body["face_count"] == 1
        face_uc.verify_face.assert_called_once()

    def test_face_not_registered_returns_404(self):
        face_uc = AsyncMock()
        face_uc.verify_face.side_effect = FaceNotRegisteredError("Not registered")

        client = TestClient(_create_test_app(face_uc=face_uc))
        payload = {
            "session_id": SESSION_ID,
            "image_b64": "data:image/jpeg;base64,abcdef",
        }
        resp = client.post("/face/verify", json=payload, headers=_headers())
        assert resp.status_code == 404
        assert resp.json() == {"detail": "Not registered"}

    def test_no_clear_face_returns_422(self):
        face_uc = AsyncMock()
        face_uc.verify_face.side_effect = NoClearFaceError("No face detected")

        client = TestClient(_create_test_app(face_uc=face_uc))
        payload = {
            "session_id": SESSION_ID,
            "image_b64": "data:image/jpeg;base64,abcdef",
        }
        resp = client.post("/face/verify", json=payload, headers=_headers())
        assert resp.status_code == 422
        assert resp.json() == {"detail": "No face detected"}

    def test_internal_service_error_returns_502(self):
        face_uc = AsyncMock()
        face_uc.verify_face.side_effect = InternalServiceError("Downstream failed")

        client = TestClient(_create_test_app(face_uc=face_uc))
        payload = {
            "session_id": SESSION_ID,
            "image_b64": "data:image/jpeg;base64,abcdef",
        }
        resp = client.post("/face/verify", json=payload, headers=_headers())
        assert resp.status_code == 502
        assert resp.json() == {"detail": "Downstream failed"}

    def test_unhandled_exception_returns_500(self):
        face_uc = AsyncMock()
        face_uc.verify_face.side_effect = RuntimeError("Crash")

        client = TestClient(_create_test_app(face_uc=face_uc))
        payload = {
            "session_id": SESSION_ID,
            "image_b64": "data:image/jpeg;base64,abcdef",
        }
        resp = client.post("/face/verify", json=payload, headers=_headers())
        assert resp.status_code == 500
        assert resp.json() == {"detail": "Face verification failed"}


# ===================================================================
# middleware.context Tests
# ===================================================================

def test_middleware_invalid_uuids():
    # Pass invalid UUID strings for headers
    headers = {
        "X-Subject-Id": "invalid-uuid-1",
        "X-Enterprise-Id": "invalid-uuid-2",
    }
    client = TestClient(_create_test_app())
    resp = client.get("/health", headers=headers)
    assert resp.status_code == 200  # health doesn't require identity, middleware shouldn't crash


def test_get_identity_missing():
    from fastapi import HTTPException
    
    # Mock Request class
    class FakeRequest:
        class State:
            pass
        def __init__(self):
            self.state = self.State()
            
    req = FakeRequest()
    
    with pytest.raises(HTTPException) as exc_info:
        get_candidate_id(req)
    assert exc_info.value.status_code == 401
    
    with pytest.raises(HTTPException) as exc_info:
        get_enterprise_id(req)
    assert exc_info.value.status_code == 401

