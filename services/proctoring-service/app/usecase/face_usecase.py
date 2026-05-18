"""
Face verification usecase — Premium+ periodic identity check.

Flow:
  1. Fetch face_registered_url from candidate-service (session reference)
  2. Detect faces in incoming webcam frame
  3. Log face_not_detected / multiple_faces anomalies as events
  4. Compare face vs reference using FaceDetector port
  5. Log identity_mismatch event if not matched
  6. Publish proctoring.identity.verified to Kafka
  7. Return FaceVerifyResponse to caller
"""
from datetime import datetime, timezone
from uuid import UUID
import json

from app.domain.models import FaceVerifyRequest, FaceVerifyResponse, IngestEventRequest
from app.domain.enums import EventType
from app.domain.ports import FaceDetector
from app.domain.errors import FaceNotRegisteredError
from app.infrastructure.client.candidate_client import CandidateServiceClient
from app.infrastructure.kafka.producer import KafkaProducer
from app.usecase.event_usecase import EventUseCase


class FaceUseCase:
    def __init__(
        self,
        detector: FaceDetector,
        candidate_client: CandidateServiceClient,
        event_uc: EventUseCase,
        producer: KafkaProducer,
        redis_client,
    ):
        self._detector = detector
        self._candidate_client = candidate_client
        self._event_uc = event_uc
        self._producer = producer
        self._redis = redis_client

    async def verify_face(
        self,
        session_id: UUID,
        candidate_id: UUID,
        enterprise_id: UUID,
        req: FaceVerifyRequest,
    ) -> FaceVerifyResponse:
        now = datetime.now(tz=timezone.utc)

        # 1. Fetch reference from candidate-service (or cache)
        cache_key = f"face_ref:{session_id}"
        cached_data = await self._redis.get(cache_key)
        
        if cached_data:
            ref_data = json.loads(cached_data)
        else:
            ref_data = await self._candidate_client.get_face_reference_data(session_id)
            if ref_data and ref_data.get("embedding"):
                await self._redis.set(cache_key, json.dumps(ref_data), ex=7200)

        if not ref_data or not ref_data.get("embedding"):
            raise FaceNotRegisteredError(
                f"No face embedding registered for session {session_id}"
            )
        
        ref_embedding = ref_data["embedding"]

        # 2. Detect faces in probe frame
        detect = await self._detector.detect(req.image_b64)

        # 3. Log detection anomalies
        if detect.face_count > 1:
            await self._event_uc.ingest_event(
                candidate_id, enterprise_id,
                IngestEventRequest(
                    session_id=session_id,
                    event_type=EventType.MULTIPLE_FACES,
                    occurred_at=now,
                    metadata={"face_count": detect.face_count},
                ),
            )

        if not detect.has_face:
            await self._event_uc.ingest_event(
                candidate_id, enterprise_id,
                IngestEventRequest(
                    session_id=session_id,
                    event_type=EventType.FACE_NOT_DETECTED,
                    occurred_at=now,
                    metadata={},
                ),
            )
            return FaceVerifyResponse(
                session_id=session_id,
                is_match=False,
                confidence=0.0,
                face_count=0,
            )

        # 4. Compare vs reference
        cmp = await self._detector.compare(ref_embedding, req.image_b64)

        # 5. Log mismatch
        if not cmp.is_match:
            await self._event_uc.ingest_event(
                candidate_id, enterprise_id,
                IngestEventRequest(
                    session_id=session_id,
                    event_type=EventType.IDENTITY_MISMATCH,
                    occurred_at=now,
                    metadata={"confidence": cmp.confidence},
                ),
            )
        else:
            # Log successful verification (score weight = 0, no cheating impact)
            await self._event_uc.ingest_event(
                candidate_id, enterprise_id,
                IngestEventRequest(
                    session_id=session_id,
                    event_type=EventType.PERIODIC_FACE_OK,
                    occurred_at=now,
                    metadata={"confidence": cmp.confidence},
                ),
            )

        # 6. Publish verification event
        await self._producer.publish("proctoring.identity.verified", {
            "session_id": str(session_id),
            "candidate_id": str(candidate_id),
            "enterprise_id": str(enterprise_id),
            "is_match": cmp.is_match,
            "confidence": cmp.confidence,
            "face_count": detect.face_count,
            "verified_at": now.isoformat(),
        })

        return FaceVerifyResponse(
            session_id=session_id,
            is_match=cmp.is_match,
            confidence=cmp.confidence,
            face_count=detect.face_count,
        )
