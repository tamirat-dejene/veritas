"""
HTTP client for the candidate-service internal API.

Used by the face verification usecase to fetch face_registered_url
from exam_sessions without storing a duplicate in the proctoring DB.
"""
import httpx
from uuid import UUID

from app.domain.errors import SessionNotFoundError, InternalServiceError


class CandidateServiceClient:
    def __init__(self, base_url: str):
        self._base = base_url.rstrip("/")

    async def get_face_reference_url(self, session_id: UUID) -> str | None:
        """
        Fetch face_registered_url for a session from candidate-service.

        Returns:
            The Cloudinary URL of the face reference image, or None if not set.

        Raises:
            SessionNotFoundError: If the session does not exist.
            InternalServiceError: On network / unexpected HTTP errors.
        """

        
        url = f"{self._base}/sessions/{session_id}"
        try:
            async with httpx.AsyncClient(timeout=5.0) as client:
                resp = await client.get(url)
        except httpx.RequestError as exc:
            raise InternalServiceError(f"Candidate service unreachable: {exc}") from exc

        if resp.status_code == 404:
            raise SessionNotFoundError(f"Session {session_id} not found in candidate-service")
        if resp.status_code != 200:
            raise InternalServiceError(
                f"Unexpected response from candidate-service: {resp.status_code}"
            )

        data = resp.json().get("data", {})
        return data.get("faceRegisteredUrl")  # None if not registered
