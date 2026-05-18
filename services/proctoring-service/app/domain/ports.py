"""
FaceDetector port — abstract interface that both DeepFace and
external API implementations must satisfy.

Swap the concrete implementation in main.py without touching
any usecase code.
"""
from abc import ABC, abstractmethod
from dataclasses import dataclass


@dataclass
class DetectResult:
    has_face: bool
    face_count: int   # >1 means multiple faces visible


@dataclass
class CompareResult:
    is_match: bool
    confidence: float  # 0.0 – 1.0 (higher = more similar)


class FaceDetector(ABC):
    """
    Port interface for face detection and identity comparison.

    Implementations:
        - DeepFaceDetector  (self-hosted, default)
        - RekognitionDetector  (AWS, swap-ready stub)
    """

    @abstractmethod
    async def detect(self, image_b64: str) -> DetectResult:
        """
        Detect faces in a base64-encoded image.

        Args:
            image_b64: Base64-encoded JPEG/PNG/WEBP image from webcam.

        Returns:
            DetectResult with has_face and face_count.
        """
        ...

    @abstractmethod
    async def compare(self, reference_embedding: list[float], probe_b64: str) -> CompareResult:
        """
        Compare a probe image against a pre-calculated reference embedding.

        Args:
            reference_embedding: Float array of the face embedding
                                 (from exam_sessions.face_registered_embedding).
            probe_b64: Base64-encoded current webcam frame.

        Returns:
            CompareResult with is_match and confidence score.
        """
        ...
