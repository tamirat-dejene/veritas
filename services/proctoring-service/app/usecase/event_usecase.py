"""
Event usecase — core behavioral event logic.

Responsibilities:
 - Persist incoming behavioral events with computed severity
 - Recompute cheating score on every new event (additive weighted, cap 100)
 - Publish proctoring.event.detected and proctoring.cheating_score.updated
 - List events per session for admin dashboard
 - Return current session score
"""
from datetime import datetime
from uuid import UUID

from app.domain.enums import EVENT_SEVERITY, EVENT_SCORE_WEIGHT
from app.domain.models import IngestEventRequest, ProctoringEvent, SessionScore
from app.repository.event_repository import EventRepository
from app.repository.score_repository import ScoreRepository
from app.infrastructure.kafka.producer import KafkaProducer


class EventUseCase:
    def __init__(
        self,
        event_repo: EventRepository,
        score_repo: ScoreRepository,
        producer: KafkaProducer,
    ):
        self._event_repo = event_repo
        self._score_repo = score_repo
        self._producer = producer

    async def ingest_event(
        self,
        candidate_id: UUID,
        enterprise_id: UUID,
        req: IngestEventRequest,
    ) -> tuple[ProctoringEvent, SessionScore]:
        # 1. Determine severity
        severity_val = EVENT_SEVERITY.get(req.event_type, "low")
        if hasattr(severity_val, "value"):
            severity_val = severity_val.value

        # 2. Persist event
        event = await self._event_repo.create(
            session_id=req.session_id,
            candidate_id=candidate_id,
            enterprise_id=enterprise_id,
            event_type=req.event_type.value,
            severity=severity_val,
            metadata=req.metadata,
            occurred_at=req.occurred_at,
        )

        # 3. Recompute score
        score = await self._recompute_score(req.session_id, candidate_id, enterprise_id)

        # 4. Publish events
        await self._producer.publish("proctoring.event.detected", {
            "event_id": str(event.id),
            "session_id": str(req.session_id),
            "candidate_id": str(candidate_id),
            "enterprise_id": str(enterprise_id),
            "event_type": req.event_type.value,
            "severity": severity_val,
            "occurred_at": req.occurred_at.isoformat(),
        })
        await self._producer.publish("proctoring.cheating_score.updated", {
            "session_id": str(req.session_id),
            "candidate_id": str(candidate_id),
            "enterprise_id": str(enterprise_id),
            "cheating_score": score.cheating_score,
            "event_count": score.event_count,
            "is_final": False,
        })

        return event, score

    async def list_events(self, session_id: UUID) -> list[ProctoringEvent]:
        return await self._event_repo.list_by_session(session_id)

    async def get_score(self, session_id: UUID) -> SessionScore | None:
        return await self._score_repo.get_by_session(session_id)

    async def _recompute_score(
        self, session_id: UUID, candidate_id: UUID, enterprise_id: UUID
    ) -> SessionScore:
        events = await self._event_repo.list_by_session(session_id)
        raw = sum(EVENT_SCORE_WEIGHT.get(e.event_type, 0.0) for e in events)
        score = round(min(100.0, raw), 2)
        return await self._score_repo.upsert(
            session_id=session_id,
            candidate_id=candidate_id,
            enterprise_id=enterprise_id,
            cheating_score=score,
            event_count=len(events),
        )
