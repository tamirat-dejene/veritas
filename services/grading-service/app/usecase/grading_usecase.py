import logging
from uuid import UUID
from typing import Any, Optional, Dict, List, Tuple

from app.repository.grading_repository import GradingRepository, DataTamperingError
from app.grading.candidate_client import CandidateServiceClient
from app.grading.enterprise_client import EnterpriseServiceClient

logger = logging.getLogger("grading.usecase")


class GradingUseCase:
    def __init__(
        self,
        repository: GradingRepository,
        candidate_client: CandidateServiceClient,
        enterprise_client: EnterpriseServiceClient,
    ):
        self._repository = repository
        self._candidate_client = candidate_client
        self._enterprise_client = enterprise_client

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
        """Retrieve grading report detail and enrich with candidate and grader profile info."""
        detail = await self._repository.get_by_session(session_id)
        if not detail:
            return None

        # 1. Fetch & Enrich Candidate Profile
        candidate_id = str(detail.get("candidate_id"))
        enterprise_id = str(detail.get("enterprise_id"))
        if candidate_id and enterprise_id:
            candidate_data = await self._candidate_client.fetch_candidate(candidate_id, enterprise_id)
            if candidate_data:
                detail["candidate_info"] = {
                    "id": candidate_data["id"],
                    "first_name": candidate_data["firstName"],
                    "last_name": candidate_data["lastName"],
                    "email": candidate_data.get("email"),
                }

        # 2. Fetch & Enrich Grader Profile
        grader = detail.get("graded_by")
        if grader and grader.get("type") == "human":
            user_data = await self._enterprise_client.fetch_user(grader["id"])
            if user_data:
                grader["user_details"] = {
                    "id": user_data["id"],
                    "first_name": user_data.get("firstName"),
                    "last_name": user_data.get("lastName"),
                    "email": user_data["email"],
                    "role": user_data["role"],
                }

        return detail

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

    async def update_question_grade_manually(
        self,
        session_id: UUID,
        session_question_id: UUID,
        new_question_score: float,
        actor_id: UUID,
        actor_role: str,
        reason: str,
        ip_address: Optional[str] = None
    ) -> Dict[str, Any]:
        """Manually override a specific question's score, generating audit records and HMAC updates."""
        logger.info(
            "Manual question score override request for session %s question %s by actor %s (role: %s). New score: %s",
            session_id, session_question_id, actor_id, actor_role, new_question_score
        )
        return await self._repository.update_question_grade_manually(
            session_id=session_id,
            session_question_id=session_question_id,
            new_question_score=new_question_score,
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
