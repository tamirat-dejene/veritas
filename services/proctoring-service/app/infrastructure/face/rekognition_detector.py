"""
AWS Rekognition stub — swap-ready implementation of the FaceDetector port.

To activate: replace `DeepFaceDetector()` with `RekognitionDetector(client)`
in main.py. No usecase or handler changes needed.

Usage:
    import boto3
    rekognition_client = boto3.client("rekognition", region_name="us-east-1")
    detector = RekognitionDetector(rekognition_client)
"""
import base64
from app.domain.ports import FaceDetector, DetectResult, CompareResult


class RekognitionDetector(FaceDetector):
    """AWS Rekognition implementation — production-grade external alternative."""

    def __init__(self, client):
        """
        Args:
            client: boto3 Rekognition client instance.
        """
        self._client = client

    async def detect(self, image_b64: str) -> DetectResult:
        import asyncio
        loop = asyncio.get_event_loop()
        image_bytes = base64.b64decode(image_b64)
        response = await loop.run_in_executor(
            None,
            lambda: self._client.detect_faces(
                Image={"Bytes": image_bytes},
                Attributes=["DEFAULT"],
            ),
        )
        face_count = len(response.get("FaceDetails", []))
        return DetectResult(has_face=face_count > 0, face_count=face_count)

    async def compare(self, reference_url: str, probe_b64: str) -> CompareResult:
        import asyncio, httpx
        loop = asyncio.get_event_loop()

        async with httpx.AsyncClient(timeout=8.0) as c:
            ref_bytes = (await c.get(reference_url)).content

        probe_bytes = base64.b64decode(probe_b64)

        response = await loop.run_in_executor(
            None,
            lambda: self._client.compare_faces(
                SourceImage={"Bytes": ref_bytes},
                TargetImage={"Bytes": probe_bytes},
                SimilarityThreshold=80.0,
            ),
        )
        matches = response.get("FaceMatches", [])
        if matches:
            similarity = matches[0]["Similarity"] / 100.0
            return CompareResult(is_match=True, confidence=round(similarity, 4))
        return CompareResult(is_match=False, confidence=0.0)
