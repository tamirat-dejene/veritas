from pydantic import BaseModel
from uuid import UUID
from datetime import datetime
from typing import Any

from app.domain.enums import EventType, Severity


# ---------------------------------------------------------------------------
# Inbound request models
# ---------------------------------------------------------------------------

class IngestEventRequest(BaseModel):
    session_id: UUID
    event_type: EventType
    occurred_at: datetime
    metadata: dict[str, Any] = {}


class FaceVerifyRequest(BaseModel):
    session_id: UUID
    image_b64: str  # base64 JPEG/PNG/WEBP webcam frame


# ---------------------------------------------------------------------------
# Domain entities returned from repository / used internally
# ---------------------------------------------------------------------------

class ProctoringEvent(BaseModel):
    id: UUID
    session_id: UUID
    candidate_id: UUID
    enterprise_id: UUID
    event_type: EventType
    severity: Severity
    metadata: dict[str, Any]
    occurred_at: datetime
    created_at: datetime


class SessionScore(BaseModel):
    session_id: UUID
    candidate_id: UUID
    enterprise_id: UUID
    cheating_score: float       # 0.0 – 100.0
    event_count: int
    last_computed_at: datetime


# ---------------------------------------------------------------------------
# Outbound response models
# ---------------------------------------------------------------------------

class FaceVerifyResponse(BaseModel):
    session_id: UUID
    is_match: bool
    confidence: float
    face_count: int


class EventListResponse(BaseModel):
    session_id: UUID
    events: list[ProctoringEvent]
    total: int


class ScoreResponse(BaseModel):
    session_id: UUID
    cheating_score: float
    event_count: int
    last_computed_at: datetime
