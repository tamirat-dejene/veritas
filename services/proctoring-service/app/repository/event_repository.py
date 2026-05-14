"""
Event repository — raw SQL with asyncpg.
Owns: proctoring_events table.
"""
import asyncpg
from uuid import UUID
from datetime import datetime
from typing import Any

from app.domain.models import ProctoringEvent
from app.domain.enums import EventType, Severity


class EventRepository:
    def __init__(self, pool: asyncpg.Pool):
        self._pool = pool

    async def create(
        self,
        session_id: UUID,
        candidate_id: UUID,
        enterprise_id: UUID,
        event_type: str,
        severity: str,
        metadata: dict[str, Any],
        occurred_at: datetime,
    ) -> ProctoringEvent:
        row = await self._pool.fetchrow(
            """
            INSERT INTO proctoring_events
                (session_id, candidate_id, enterprise_id, event_type,
                 severity, metadata, occurred_at)
            VALUES ($1, $2, $3, $4, $5::proctoring_severity, $6, $7)
            RETURNING *
            """,
            session_id, candidate_id, enterprise_id, event_type,
            severity, metadata, occurred_at,
        )
        return self._to_model(row)

    async def list_by_session(self, session_id: UUID) -> list[ProctoringEvent]:
        rows = await self._pool.fetch(
            "SELECT * FROM proctoring_events WHERE session_id = $1 ORDER BY occurred_at ASC",
            session_id,
        )
        return [self._to_model(r) for r in rows]

    async def count_by_session(self, session_id: UUID) -> int:
        row = await self._pool.fetchrow(
            "SELECT COUNT(*) AS cnt FROM proctoring_events WHERE session_id = $1",
            session_id,
        )
        return row["cnt"]

    @staticmethod
    def _to_model(row: asyncpg.Record) -> ProctoringEvent:
        return ProctoringEvent(
            id=row["id"],
            session_id=row["session_id"],
            candidate_id=row["candidate_id"],
            enterprise_id=row["enterprise_id"],
            event_type=EventType(row["event_type"]),
            severity=Severity(row["severity"]),
            metadata=dict(row["metadata"]) if row["metadata"] else {},
            occurred_at=row["occurred_at"],
            created_at=row["created_at"],
        )
