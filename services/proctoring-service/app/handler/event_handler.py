"""
Proctoring event handlers.

POST /proctoring/events         — candidate ingests a behavioral event
GET  /proctoring/sessions/{id}/events  — admin lists all events for a session
GET  /proctoring/sessions/{id}/score   — admin gets the current cheating score
"""
from uuid import UUID

from fastapi import APIRouter, Request, HTTPException

from app.domain.models import (
    IngestEventRequest,
    EventListResponse,
    ScoreResponse,
)
from app.middleware.context import get_candidate_id, get_enterprise_id

router = APIRouter()

@router.post("/proctoring/events", status_code=201)
async def ingest_event(body: IngestEventRequest, request: Request):
    """
    Ingest a single behavioral event from the candidate's browser.

    Computes severity, persists the event, recomputes cheating score,
    and publishes proctoring.event.detected + proctoring.cheating_score.updated.
    """
    candidate_id = get_candidate_id(request)
    enterprise_id = get_enterprise_id(request)
    event_uc = request.app.state.event_uc

    try:
        event, score = await event_uc.ingest_event(candidate_id, enterprise_id, body)
    except Exception as exc:
        raise HTTPException(status_code=500, detail="Failed to ingest event") from exc

    return {
        "message": "Event recorded",
        "event_id": str(event.id),
        "cheating_score": score.cheating_score,
    }


@router.get(
    "/proctoring/sessions/{session_id}/events",
    response_model=EventListResponse,
)
async def list_events(session_id: UUID, request: Request):
    """
    List all proctoring events for a session (admin view).
    Returns events in chronological order.
    """
    event_uc = request.app.state.event_uc
    events = await event_uc.list_events(session_id)
    return EventListResponse(
        session_id=session_id,
        events=events,
        total=len(events),
    )


@router.get(
    "/proctoring/sessions/{session_id}/score",
    response_model=ScoreResponse,
)
async def get_score(session_id: UUID, request: Request):
    """
    Return the current cheating probability score for a session (admin view).
    Returns 404 if no events have been recorded yet for this session.
    """
    event_uc = request.app.state.event_uc
    score = await event_uc.get_score(session_id)
    if not score:
        raise HTTPException(
            status_code=404,
            detail="No proctoring data found for this session",
        )
    return ScoreResponse(
        session_id=score.session_id,
        cheating_score=score.cheating_score,
        event_count=score.event_count,
        last_computed_at=score.last_computed_at,
    )
