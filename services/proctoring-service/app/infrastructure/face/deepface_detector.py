"""
DeepFace implementation of the FaceDetector port.

Uses ArcFace model for identity comparison and RetinaFace backend
for face detection. CPU-bound work is offloaded to a ThreadPoolExecutor
to avoid blocking the async event loop.

To swap to AWS Rekognition: replace `DeepFaceDetector()` with
`RekognitionDetector(boto3_client)` in main.py — no usecase changes needed.
"""
import asyncio
import base64
from concurrent.futures import ThreadPoolExecutor

# pyrefly: ignore [missing-import]
import cv2
import httpx
import numpy as np

from app.domain.ports import FaceDetector, DetectResult, CompareResult

_executor = ThreadPoolExecutor(max_workers=2, thread_name_prefix="deepface")


def _b64_to_array(b64: str) -> np.ndarray:
    data = base64.b64decode(b64)
    arr = np.frombuffer(data, np.uint8)
    img = cv2.imdecode(arr, cv2.IMREAD_COLOR)
    if img is None:
        raise ValueError("Failed to decode image from base64")
    return img


def _run_detect(img_array: np.ndarray) -> list:
    # pyrefly: ignore [missing-import]
    from deepface import DeepFace  # lazy import — heavy module
    return DeepFace.extract_faces(
        img_array,
        detector_backend="retinaface",
        enforce_detection=False,
    )


def _run_compare(ref_bytes: bytes, probe_bytes: bytes) -> dict:
    # pyrefly: ignore [missing-import]
    from deepface import DeepFace  # lazy import
    ref_arr = np.frombuffer(ref_bytes, np.uint8)
    probe_arr = np.frombuffer(probe_bytes, np.uint8)
    ref_img = cv2.imdecode(ref_arr, cv2.IMREAD_COLOR)
    probe_img = cv2.imdecode(probe_arr, cv2.IMREAD_COLOR)
    return DeepFace.verify(
        img1_path=ref_img,
        img2_path=probe_img,
        model_name="ArcFace",
        enforce_detection=False,
    )


class DeepFaceDetector(FaceDetector):
    """Self-hosted face detector using DeepFace + ArcFace + RetinaFace."""

    async def detect(self, image_b64: str) -> DetectResult:
        img = _b64_to_array(image_b64)
        loop = asyncio.get_event_loop()
        faces = await loop.run_in_executor(_executor, _run_detect, img)
        return DetectResult(has_face=len(faces) > 0, face_count=len(faces))

    async def compare(self, reference_url: str, probe_b64: str) -> CompareResult:
        # Download reference image from Cloudinary
        async with httpx.AsyncClient(timeout=8.0) as client:
            resp = await client.get(reference_url)
            resp.raise_for_status()
            ref_bytes = resp.content

        probe_bytes = base64.b64decode(probe_b64)
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(_executor, _run_compare, ref_bytes, probe_bytes)
        confidence = round(max(0.0, 1.0 - result["distance"]), 4)
        return CompareResult(is_match=result["verified"], confidence=confidence)
