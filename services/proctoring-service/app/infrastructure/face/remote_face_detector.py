"""
Remote API implementation of the FaceDetector port.

Integrates with a custom DeepFace API deployment.
"""
import base64
import logging

import httpx

from app.domain.ports import FaceDetector, DetectResult, CompareResult

logger = logging.getLogger("proctoring.remote_face")


class RemoteFaceDetector(FaceDetector):
    """
    Custom DeepFace REST API implementation.
    """

    def __init__(
        self,
        base_url: str,
        model_name: str = "Facenet512",
        detector_backend: str = "retinaface",
        timeout: float = 15.0,
    ):
        self._base_url = base_url.rstrip("/")
        self._model_name = model_name
        self._detector_backend = detector_backend
        self._timeout = timeout
        self._headers = {
            "Content-Type": "application/json",
        }

    def _strip_b64_prefix(self, b64_str: str) -> str:
        if "base64," in b64_str:
            return b64_str.split("base64,")[1]
        return b64_str

    async def detect(self, image_b64: str) -> DetectResult:
        url = f"{self._base_url}/represent"
        payload = {
            "img": self._strip_b64_prefix(image_b64),
            "model_name": self._model_name,
            "detector_backend": self._detector_backend,
        }
        try:
            async with httpx.AsyncClient(timeout=self._timeout) as client:
                resp = await client.post(
                    url,
                    json=payload,
                    headers=self._headers,
                )
            if resp.status_code == 200:
                data = resp.json()
                results = data.get("results", [])
                face_count = len(results)
                return DetectResult(has_face=face_count > 0, face_count=face_count)
            elif resp.status_code == 400:
                return DetectResult(has_face=False, face_count=0)
            else:
                logger.error("Face API /represent returned %d: %s", resp.status_code, resp.text)
                return DetectResult(has_face=True, face_count=1)
        except httpx.RequestError as exc:
            logger.error("Face API /represent unreachable: %s", exc)
            return DetectResult(has_face=True, face_count=1)

    async def compare(self, reference_embedding: list[float], probe_b64: str) -> CompareResult:
        url = f"{self._base_url}/compare"
        payload = {
            "embedding": reference_embedding,
            "img": self._strip_b64_prefix(probe_b64),
            "model_name": self._model_name,
            "detector_backend": "opencv",
        }
        try:
            async with httpx.AsyncClient(timeout=self._timeout) as client:
                resp = await client.post(
                    url,
                    json=payload,
                    headers=self._headers,
                )
        except httpx.RequestError as exc:
            logger.error("Face API /compare unreachable: %s", exc)
            raise

        if resp.status_code == 200:
            data = resp.json()
            is_match = data.get("verified", False)
            distance = data.get("distance", 1.0)
            threshold = data.get("threshold", 0.4)
            confidence = round(max(0.0, 1.0 - distance / threshold), 4) if threshold else 0.0
            return CompareResult(is_match=is_match, confidence=confidence)

        elif resp.status_code == 400:
            logger.warning("Face API /compare 400 — no face detected in images")
            return CompareResult(is_match=False, confidence=0.0)

        else:
            logger.error("Face API /compare returned %d: %s", resp.status_code, resp.text)
            return CompareResult(is_match=False, confidence=0.0)
