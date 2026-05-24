import logging
from uuid import UUID
from typing import Any, Optional, Dict, List, Tuple

from app.repository.grading_repository import GradingRepository, DataTamperingError

logger = logging.getLogger("grading.usecase")


class GradingUseCase:
    def __init__(self, repository: GradingRepository):
        self._repository = repository

    async def list_graded_students(
        self,
        enterprise_id: UUID,
        exam_id: Optional[UUID] = None,
        limit: int = 10,
        offset: int = 0
    ) -> Tuple[List[Dict[str, Any]], int]:
        """Get paginated graded examinees for an enterprise or exam."""
        return await self._repository.list_graded_students(
            enterprise_id=enterprise_id,
            exam_id=exam_id,
            limit=limit,
            offset=offset
        )

    async def get_grade_detail(self, session_id: UUID) -> Optional[Dict[str, Any]]:
        """Retrieve grading report detail (scores, questions, tamper flags)."""
        return await self._repository.get_by_session(session_id)

    async def update_grade_manually(
        self,
        session_id: UUID,
        new_score: float,
        actor_id: UUID,
        actor_role: str,
        reason: str,
        ip_address: Optional[str] = None
    ) -> Dict[str, Any]:
        """Manually override a score, generating audit records and HMAC updates."""
        logger.info(
            "Manual score override request for session %s by actor %s (role: %s). New score: %s",
            session_id, actor_id, actor_role, new_score
        )
        return await self._repository.update_grade_manually(
            session_id=session_id,
            new_awarded_points=new_score,
            actor_id=actor_id,
            actor_role=actor_role,
            reason=reason,
            ip_address=ip_address
        )

    async def get_audit_logs(self, session_id: UUID) -> List[Dict[str, Any]]:
        """Get the full immutable history/audit logs of edits for a session's grade."""
        return await self._repository.get_audit_logs(session_id)

    async def get_grading_status(self, session_id: UUID) -> Optional[Dict[str, Any]]:
        """Return the current grading status for a session.

        Returns a dict with ``status``, ``graded_by``, ``percentage``, and
        ``enterprise_id`` (for tenant checks), or None if grading has not
        started yet (event not yet received).
        """
        return await self._repository.get_grading_status(session_id)
