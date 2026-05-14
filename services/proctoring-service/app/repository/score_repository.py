"""
Score repository — raw SQL with asyncpg.
Owns: proctoring_session_scores table.
Uses INSERT ... ON CONFLICT (UPSERT) pattern.
"""
import asyncpg
from uuid import UUID

from app.domain.models import SessionScore


class ScoreRepository:
    def __init__(self, pool: asyncpg.Pool):
        self._pool = pool

    async def upsert(
        self,
        session_id: UUID,
        candidate_id: UUID,
        enterprise_id: UUID,
        cheating_score: float,
        event_count: int,
    ) -> SessionScore:
        row = await self._pool.fetchrow(
            """
            INSERT INTO proctoring_session_scores
                (session_id, candidate_id, enterprise_id, cheating_score, event_count,
                 last_computed_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
            ON CONFLICT (session_id) DO UPDATE SET
                cheating_score   = EXCLUDED.cheating_score,
                event_count      = EXCLUDED.event_count,
                last_computed_at = NOW(),
                updated_at       = NOW()
            RETURNING *
            """,
            session_id, candidate_id, enterprise_id, cheating_score, event_count,
        )
        return self._to_model(row)

    async def get_by_session(self, session_id: UUID) -> SessionScore | None:
        row = await self._pool.fetchrow(
            "SELECT * FROM proctoring_session_scores WHERE session_id = $1",
            session_id,
        )
        return self._to_model(row) if row else None

    @staticmethod
    def _to_model(row: asyncpg.Record) -> SessionScore:
        return SessionScore(
            session_id=row["session_id"],
            candidate_id=row["candidate_id"],
            enterprise_id=row["enterprise_id"],
            cheating_score=float(row["cheating_score"]),
            event_count=row["event_count"],
            last_computed_at=row["last_computed_at"],
        )
