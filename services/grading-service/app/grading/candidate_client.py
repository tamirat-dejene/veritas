"""
CandidateServiceClient — async HTTP client for the candidate-service internal API.

The grading-service calls this after receiving a slim ``exam.session.ready_for_grading``
Kafka event to fetch the full grading payload (questions + answers + evaluation criteria).
"""
from __future__ import annotations

import logging
from typing import Any

import httpx

from app.config import settings
from app.grading.models import GradingPayload

logger = logging.getLogger("grading.candidate_client")


class CandidateServiceError(Exception):
    """Raised when the candidate-service returns an unexpected response."""


class CandidateServiceClient:
    """
    Async HTTP client wrapping the candidate-service internal endpoint.

    Usage::

        async with CandidateServiceClient() as client:
            payload = await client.fetch_grading_payload(session_id, enterprise_id)

    Or keep a long-lived instance (the underlying ``httpx.AsyncClient`` is reused
    across requests, which is more efficient in the Kafka consumer loop).
    """

    def __init__(self) -> None:
        self._client: httpx.AsyncClient | None = None

    async def start(self) -> None:
        """Open the underlying HTTP connection pool."""
        self._client = httpx.AsyncClient(
            base_url=settings.CANDIDATE_SERVICE_URL,
            timeout=settings.CANDIDATE_SERVICE_TIMEOUT_SECONDS,
        )
        logger.info(
            "CandidateServiceClient started — base_url=%s",
            settings.CANDIDATE_SERVICE_URL,
        )

    async def stop(self) -> None:
        """Close the underlying HTTP connection pool."""
        if self._client is not None:
            await self._client.aclose()
            self._client = None

    async def __aenter__(self) -> "CandidateServiceClient":
        await self.start()
        return self

    async def __aexit__(self, *_: Any) -> None:
        await self.stop()

    async def fetch_grading_payload(
        self,
        session_id: str,
        enterprise_id: str,
    ) -> GradingPayload:
        """
        Call ``GET /internal/sessions/{session_id}/grading-payload`` and return
        the parsed ``GradingPayload``.

        Raises:
            CandidateServiceError: on non-2xx responses or network errors.
        """
        if self._client is None:
            raise RuntimeError(
                "CandidateServiceClient is not started. Call start() or use as async context manager."
            )

        url = f"/internal/sessions/{session_id}/grading-payload"
        logger.info(
            "Fetching grading payload  session=%s  enterprise=%s  url=%s%s",
            session_id,
            enterprise_id,
            settings.CANDIDATE_SERVICE_URL,
            url,
        )

        try:
            response = await self._client.get(
                url,
                headers={"X-Enterprise-Id": enterprise_id},
            )
        except httpx.RequestError as exc:
            logger.error(
                "Network error fetching grading payload  session=%s  error=%s",
                session_id,
                exc,
            )
            raise CandidateServiceError(
                f"Network error fetching grading payload for session {session_id}: {exc}"
            ) from exc

        logger.debug(
            "candidate-service responded  session=%s  status=%d",
            session_id,
            response.status_code,
        )

        if response.status_code != 200:
            logger.error(
                "Unexpected status from candidate-service  session=%s  status=%d  body=%s",
                session_id,
                response.status_code,
                response.text[:500],
            )
            raise CandidateServiceError(
                f"candidate-service returned {response.status_code} for session {session_id}: "
                f"{response.text}"
            )

        try:
            parsed = GradingPayload.model_validate(response.json())
            logger.info(
                "Grading payload received  session=%s  items=%d",
                session_id,
                len(parsed.items),
            )
            return parsed
        except Exception as exc:
            logger.error(
                "Failed to parse GradingPayload  session=%s  error=%s  raw_body=%.500s",
                session_id,
                exc,
                response.text,
            )
            raise CandidateServiceError(
                f"Failed to parse GradingPayload for session {session_id}: {exc}"
            ) from exc
