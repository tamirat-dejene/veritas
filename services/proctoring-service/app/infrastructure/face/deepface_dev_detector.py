"""
deepface.dev managed API implementation of the FaceDetector port.

The official managed REST API built by the DeepFace author team.
https://deepface.dev  |  https://docs.deepface.dev

Usage (swap in app/router.py):
    detector = DeepFaceDevDetector(
        api_key=settings.DEEPFACE_DEV_API_KEY,
        model_name="Facenet512",     # or Facenet, Dlib, OpenFace, SFace
        detector_backend="retinaface",
    )

Commercial note from deepface.dev:
    - VGG-Face is excluded (non-commercial weights).
    - ArcFace based on InsightFace weights requires a separate commercial license.
    - Safe defaults: Facenet512 (recommended), Facenet, SFace.

Sign-up: https://deepface.dev/signup
API key: create from the dashboard after sign-up.

The /verify endpoint accepts:
    - multipart/form-data with img1=<file>, img2=<file>
    - application/json with img1=<base64>, img2=<base64>

Response shape:
    { "verified": bool, "distance": float, "threshold": float, "model": str }

The /represent endpoint (used for detect-only) is not available in detect-only
mode without a reference image, so detect() calls /verify against itself as a
lightweight face-presence check. For detect-only calls the service simply checks
if the round-trip returns a valid response (non-error = face found).
"""
import base64
import logging
from uuid import uuid4

import httpx

from app.domain.ports import FaceDetector, DetectResult, CompareResult

logger = logging.getLogger("proctoring.deepface_dev")

_BASE_URL = "https://api.deepface.dev"


class DeepFaceDevDetector(FaceDetector):
    """
    deepface.dev managed REST API implementation.

    Replaces DeepFaceDetector with zero local dependencies —
    no GPU, no DeepFace install, no OpenCV required.
    Drop-in swap via the FaceDetector port.
    """

    def __init__(
        self,
        api_key: str,
        model_name: str = "Facenet512",
        detector_backend: str = "retinaface",
        timeout: float = 15.0,
    ):
        """
        Args:
            api_key:          Bearer token from https://deepface.dev dashboard.
            model_name:       One of Facenet, Facenet512, Dlib, OpenFace, SFace.
                              Facenet512 is recommended for accuracy.
            detector_backend: Face detector to use for alignment.
                              retinaface gives the best accuracy.
            timeout:          HTTP request timeout in seconds.
        """
        self._api_key = api_key
        self._model_name = model_name
        self._detector_backend = detector_backend
        self._timeout = timeout
        self._headers = {
            "Authorization": f"Bearer {api_key}",
            "x-request-id": "",  # set per-request
        }

    def _request_headers(self) -> dict:
        return {**self._headers, "x-request-id": str(uuid4())}

    async def detect(self, image_b64: str) -> DetectResult:
        """
        Detect faces by calling POST /represent.
        Returns face count from the embeddings array length.
        Falls back to face_count=1 if API returns a valid representation.
        """
        url = f"{_BASE_URL}/represent"
        payload = {
            "img": image_b64,
            "model_name": self._model_name,
            "detector_backend": self._detector_backend,
        }
        try:
            async with httpx.AsyncClient(timeout=self._timeout) as client:
                resp = await client.post(
                    url,
                    json=payload,
                    headers=self._request_headers(),
                )
            if resp.status_code == 200:
                data = resp.json()
                # /represent returns a list of face embeddings (one per detected face)
                results = data.get("results", [])
                face_count = len(results)
                return DetectResult(has_face=face_count > 0, face_count=face_count)
            elif resp.status_code == 400:
                # deepface.dev returns 400 when no face is detected
                return DetectResult(has_face=False, face_count=0)
            else:
                logger.error(
                    "deepface.dev /represent returned %d: %s",
                    resp.status_code, resp.text,
                )
                # Fail open — assume face present to avoid false positives
                return DetectResult(has_face=True, face_count=1)
        except httpx.RequestError as exc:
            logger.error("deepface.dev /represent unreachable: %s", exc)
            return DetectResult(has_face=True, face_count=1)

    async def compare(self, reference_url: str, probe_b64: str) -> CompareResult:
        """
        Compare probe image against reference URL using POST /verify.

        deepface.dev /verify accepts:
          - multipart/form-data: img1=<file>, img2=<file>
          - application/json:    img1=<base64>, img2=<base64>

        We use JSON with base64 for both images. The reference is
        downloaded from Cloudinary first, then base64-encoded.
        """
        # 1. Download reference image from Cloudinary
        try:
            async with httpx.AsyncClient(timeout=8.0) as client:
                ref_resp = await client.get(reference_url)
                ref_resp.raise_for_status()
                ref_b64 = base64.b64encode(ref_resp.content).decode("utf-8")
        except httpx.RequestError as exc:
            logger.error("Failed to download reference image: %s", exc)
            raise

        # 2. Call POST /verify with both images as base64
        url = f"{_BASE_URL}/verify"
        payload = {
            "img1": ref_b64,
            "img2": probe_b64,
            "model_name": self._model_name,
            "detector_backend": self._detector_backend,
        }
        try:
            async with httpx.AsyncClient(timeout=self._timeout) as client:
                resp = await client.post(
                    url,
                    json=payload,
                    headers=self._request_headers(),
                )
        except httpx.RequestError as exc:
            logger.error("deepface.dev /verify unreachable: %s", exc)
            raise

        if resp.status_code == 200:
            data = resp.json()
            # Response: {"verified": bool, "distance": float, "threshold": float, "model": str}
            is_match = data.get("verified", False)
            distance = data.get("distance", 1.0)
            threshold = data.get("threshold", 0.4)
            # Normalise distance to a 0–1 confidence score
            confidence = round(max(0.0, 1.0 - distance / threshold), 4) if threshold else 0.0
            return CompareResult(is_match=is_match, confidence=confidence)

        elif resp.status_code == 400:
            # No face detected in one or both images
            logger.warning("deepface.dev /verify 400 — no face detected in images")
            return CompareResult(is_match=False, confidence=0.0)

        else:
            logger.error(
                "deepface.dev /verify returned %d: %s",
                resp.status_code, resp.text,
            )
            return CompareResult(is_match=False, confidence=0.0)
