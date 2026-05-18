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

    async def get_face_reference_data(self, session_id: UUID) -> dict | None:
        """
        Fetch face_registered_url and face_registered_embedding for a session.

        Returns:
            A dictionary with 'url' and 'embedding', or None if neither are set.

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
        url_val = data.get("faceRegisteredUrl")
        embedding_val = data.get("faceRegisteredEmbedding")
        
        if not url_val and not embedding_val:
            return None
            
        return {
            "url": url_val,
            "embedding": embedding_val
        }
