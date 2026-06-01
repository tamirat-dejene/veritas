"""
EnterpriseServiceClient — async HTTP client for the enterprise-service internal API.
"""
from __future__ import annotations

import logging
from typing import Any

import httpx

from app.config import settings

logger = logging.getLogger("grading.enterprise_client")


class EnterpriseServiceError(Exception):
    """Raised when the enterprise-service returns an unexpected response."""


class EnterpriseServiceClient:
    """
    Async HTTP client wrapping the enterprise-service internal endpoint.
    """

    def __init__(self) -> None:
        self._client: httpx.AsyncClient | None = None

    async def start(self) -> None:
        """Open the underlying HTTP connection pool."""
        self._client = httpx.AsyncClient(
            base_url=settings.ENTERPRISE_SERVICE_URL,
            timeout=30.0,
        )
        logger.info(
            "EnterpriseServiceClient started — base_url=%s",
            settings.ENTERPRISE_SERVICE_URL,
        )

    async def stop(self) -> None:
        """Close the underlying HTTP connection pool."""
        if self._client is not None:
            await self._client.aclose()
            self._client = None

    async def __aenter__(self) -> "EnterpriseServiceClient":
        await self.start()
        return self

    async def __aexit__(self, *_: Any) -> None:
        await self.stop()

    async def fetch_user(self, user_id: str) -> dict | None:
        """
        Call ``GET /internal/users/{user_id}`` and return the user payload.
        """
        if self._client is None:
            raise RuntimeError("EnterpriseServiceClient is not started.")

        url = f"/internal/users/{user_id}"
        try:
            response = await self._client.get(url)
            if response.status_code == 200:
                return response.json()
            elif response.status_code == 404:
                logger.warning("User %s not found in enterprise-service", user_id)
                return None
            else:
                logger.error(
                    "Unexpected status from enterprise-service user lookup  user=%s  status=%d",
                    user_id,
                    response.status_code,
                )
                return None
        except Exception as exc:
            logger.error(
                "Error fetching user details from enterprise-service  user=%s  error=%s",
                user_id,
                exc,
            )
            return None
