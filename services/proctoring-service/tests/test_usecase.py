"""
Unit tests for the business logic use cases in proctoring-service.
"""
import json
import uuid
from datetime import datetime, timedelta, timezone
from unittest.mock import AsyncMock, MagicMock

# pyrefly: ignore [missing-import]
import pytest

from app.domain.enums import EventType, Severity
from app.domain.errors import FaceNotRegisteredError
from app.domain.models import IngestEventRequest, ProctoringEvent, SessionScore
from app.domain.ports import CompareResult, DetectResult
from app.usecase.event_usecase import EventUseCase
from app.usecase.face_usecase import FaceUseCase
from tests.conftest import CANDIDATE_ID, ENTERPRISE_ID, EVENT_ID, SESSION_ID


# ===================================================================
# EventUseCase Tests
# ===================================================================

class TestEventUseCase:

    @pytest.mark.asyncio
    async def test_ingest_event_computes_score_and_publishes(
        self, mock_pool, mock_kafka_producer
    ):
        # Setup mocks
        event_repo = AsyncMock()
        score_repo = AsyncMock()
        
        # Mock created event returned from repo
        created_event = ProctoringEvent(
            id=uuid.UUID(EVENT_ID),
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            event_type=EventType.TAB_SWITCH,
            severity=Severity.MEDIUM,
            metadata={"test": "meta"},
            occurred_at=datetime.now(timezone.utc),
            created_at=datetime.now(timezone.utc),
        )
        event_repo.create.return_value = created_event

        # List of past events for recomputation (we return 2 events: TAB_SWITCH and MOUSE_INACTIVE)
        past_events = [
            ProctoringEvent(
                id=uuid.UUID(EVENT_ID),
                session_id=uuid.UUID(SESSION_ID),
                candidate_id=uuid.UUID(CANDIDATE_ID),
                enterprise_id=uuid.UUID(ENTERPRISE_ID),
                event_type=EventType.TAB_SWITCH,
                severity=Severity.MEDIUM,
                metadata={},
                occurred_at=datetime.now(timezone.utc),
                created_at=datetime.now(timezone.utc),
            ),
            ProctoringEvent(
                id=uuid.uuid4(),
                session_id=uuid.UUID(SESSION_ID),
                candidate_id=uuid.UUID(CANDIDATE_ID),
                enterprise_id=uuid.UUID(ENTERPRISE_ID),
                event_type=EventType.MOUSE_INACTIVE,
                severity=Severity.LOW,
                metadata={},
                occurred_at=datetime.now(timezone.utc),
                created_at=datetime.now(timezone.utc),
            ),
        ]
        event_repo.list_by_session.return_value = past_events

        # Mock upsert score return value
        # Score calculation: TAB_SWITCH (5.0) + MOUSE_INACTIVE (3.0) = 8.0
        saved_score = SessionScore(
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            cheating_score=8.0,
            event_count=2,
            last_computed_at=datetime.now(timezone.utc),
        )
        score_repo.upsert.return_value = saved_score

        uc = EventUseCase(event_repo, score_repo, mock_kafka_producer)
        req = IngestEventRequest(
            session_id=uuid.UUID(SESSION_ID),
            event_type=EventType.TAB_SWITCH,
            occurred_at=datetime.now(timezone.utc),
            metadata={"test": "meta"},
        )

        event, score = await uc.ingest_event(
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            req=req,
        )

        # Assertions
        assert event == created_event
        assert score == saved_score

        # Verify event creation arguments
        event_repo.create.assert_called_once_with(
            session_id=req.session_id,
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            event_type=req.event_type.value,
            severity=Severity.MEDIUM.value,
            metadata=req.metadata,
            occurred_at=req.occurred_at,
        )

        # Verify score upsert arguments
        score_repo.upsert.assert_called_once_with(
            session_id=req.session_id,
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            cheating_score=8.0,
            event_count=2,
        )

        # Verify Kafka messages
        assert mock_kafka_producer.publish.call_count == 2
        mock_kafka_producer.publish.assert_any_call(
            "proctoring.event.detected",
            {
                "event_id": str(created_event.id),
                "session_id": str(req.session_id),
                "candidate_id": str(CANDIDATE_ID),
                "enterprise_id": str(ENTERPRISE_ID),
                "event_type": req.event_type.value,
                "severity": Severity.MEDIUM.value,
                "occurred_at": req.occurred_at.isoformat(),
            },
        )
        mock_kafka_producer.publish.assert_any_call(
            "proctoring.cheating_score.updated",
            {
                "session_id": str(req.session_id),
                "candidate_id": str(CANDIDATE_ID),
                "enterprise_id": str(ENTERPRISE_ID),
                "cheating_score": 8.0,
                "event_count": 2,
                "is_final": False,
            },
        )

    @pytest.mark.asyncio
    async def test_ingest_event_caps_score_at_100(
        self, mock_kafka_producer
    ):
        event_repo = AsyncMock()
        score_repo = AsyncMock()
        
        # Mock created event returned from repo
        created_event = ProctoringEvent(
            id=uuid.UUID(EVENT_ID),
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            event_type=EventType.IDENTITY_MISMATCH,
            severity=Severity.CRITICAL,
            metadata={},
            occurred_at=datetime.now(timezone.utc),
            created_at=datetime.now(timezone.utc),
        )
        event_repo.create.return_value = created_event

        # Generate a large number of past events to exceed 100 points
        # Identity Mismatch has weight 20.0, so 6 events = 120.0 -> cap at 100.0
        # Space them out by 20 seconds to bypass the deduplication cooldown.
        base_time = datetime.now(timezone.utc)
        large_event_list = []
        for i in range(6):
            e = ProctoringEvent(
                id=uuid.uuid4(),
                session_id=uuid.UUID(SESSION_ID),
                candidate_id=uuid.UUID(CANDIDATE_ID),
                enterprise_id=uuid.UUID(ENTERPRISE_ID),
                event_type=EventType.IDENTITY_MISMATCH,
                severity=Severity.CRITICAL,
                metadata={},
                occurred_at=base_time + timedelta(seconds=i * 20),
                created_at=base_time + timedelta(seconds=i * 20),
            )
            large_event_list.append(e)
            
        event_repo.list_by_session.return_value = large_event_list

        uc = EventUseCase(event_repo, score_repo, mock_kafka_producer)
        req = IngestEventRequest(
            session_id=uuid.UUID(SESSION_ID),
            event_type=EventType.IDENTITY_MISMATCH,
            occurred_at=datetime.now(timezone.utc),
            metadata={},
        )

        await uc.ingest_event(
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            req=req,
        )

        # Verify score upsert with 100.0 cap
        score_repo.upsert.assert_called_once_with(
            session_id=req.session_id,
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            cheating_score=100.0,
            event_count=6,
        )

    @pytest.mark.asyncio
    async def test_deduplication_cooldown(self, mock_kafka_producer):
        event_repo = AsyncMock()
        score_repo = AsyncMock()
        base_time = datetime.now(timezone.utc)
        
        # Two events of the same type within 5 seconds (cooldown is 15)
        events = [
            ProctoringEvent(
                id=uuid.uuid4(), session_id=uuid.UUID(SESSION_ID), candidate_id=uuid.UUID(CANDIDATE_ID),
                enterprise_id=uuid.UUID(ENTERPRISE_ID), event_type=EventType.TAB_SWITCH, severity=Severity.MEDIUM,
                metadata={}, occurred_at=base_time, created_at=base_time,
            ),
            ProctoringEvent(
                id=uuid.uuid4(), session_id=uuid.UUID(SESSION_ID), candidate_id=uuid.UUID(CANDIDATE_ID),
                enterprise_id=uuid.UUID(ENTERPRISE_ID), event_type=EventType.TAB_SWITCH, severity=Severity.MEDIUM,
                metadata={}, occurred_at=base_time + timedelta(seconds=5), created_at=base_time + timedelta(seconds=5),
            )
        ]
        event_repo.list_by_session.return_value = events
        uc = EventUseCase(event_repo, score_repo, mock_kafka_producer)
        
        req = IngestEventRequest(session_id=uuid.UUID(SESSION_ID), event_type=EventType.TAB_SWITCH, occurred_at=base_time, metadata={})
        await uc.ingest_event(uuid.UUID(CANDIDATE_ID), uuid.UUID(ENTERPRISE_ID), req)
        
        # Second event should be ignored for score calculation. Only 1 Tab Switch (5.0) should count.
        score_repo.upsert.assert_called_once()
        assert score_repo.upsert.call_args[1]["cheating_score"] == 5.0

    @pytest.mark.asyncio
    async def test_per_type_capping(self, mock_kafka_producer):
        event_repo = AsyncMock()
        score_repo = AsyncMock()
        base_time = datetime.now(timezone.utc)
        
        # 6 Tab Switches spaced by 20s. Each is 5.0. Total raw = 30.0.
        # But Tab Switch is capped at 20.0.
        events = []
        for i in range(6):
            events.append(ProctoringEvent(
                id=uuid.uuid4(), session_id=uuid.UUID(SESSION_ID), candidate_id=uuid.UUID(CANDIDATE_ID),
                enterprise_id=uuid.UUID(ENTERPRISE_ID), event_type=EventType.TAB_SWITCH, severity=Severity.MEDIUM,
                metadata={}, occurred_at=base_time + timedelta(seconds=i*20), created_at=base_time,
            ))
        event_repo.list_by_session.return_value = events
        uc = EventUseCase(event_repo, score_repo, mock_kafka_producer)
        
        req = IngestEventRequest(session_id=uuid.UUID(SESSION_ID), event_type=EventType.TAB_SWITCH, occurred_at=base_time, metadata={})
        await uc.ingest_event(uuid.UUID(CANDIDATE_ID), uuid.UUID(ENTERPRISE_ID), req)
        
        score_repo.upsert.assert_called_once()
        assert score_repo.upsert.call_args[1]["cheating_score"] == 20.0

    @pytest.mark.asyncio
    async def test_score_recovery(self, mock_kafka_producer):
        event_repo = AsyncMock()
        score_repo = AsyncMock()
        base_time = datetime.now(timezone.utc)
        
        # 1 Tab Switch (5.0) + 2 Periodic Face OK (-2.0 each) = 1.0
        events = [
            ProctoringEvent(
                id=uuid.uuid4(), session_id=uuid.UUID(SESSION_ID), candidate_id=uuid.UUID(CANDIDATE_ID),
                enterprise_id=uuid.UUID(ENTERPRISE_ID), event_type=EventType.TAB_SWITCH, severity=Severity.MEDIUM,
                metadata={}, occurred_at=base_time, created_at=base_time,
            ),
            ProctoringEvent(
                id=uuid.uuid4(), session_id=uuid.UUID(SESSION_ID), candidate_id=uuid.UUID(CANDIDATE_ID),
                enterprise_id=uuid.UUID(ENTERPRISE_ID), event_type=EventType.PERIODIC_FACE_OK, severity=Severity.LOW,
                metadata={}, occurred_at=base_time + timedelta(seconds=10), created_at=base_time,
            ),
            ProctoringEvent(
                id=uuid.uuid4(), session_id=uuid.UUID(SESSION_ID), candidate_id=uuid.UUID(CANDIDATE_ID),
                enterprise_id=uuid.UUID(ENTERPRISE_ID), event_type=EventType.PERIODIC_FACE_OK, severity=Severity.LOW,
                metadata={}, occurred_at=base_time + timedelta(seconds=20), created_at=base_time,
            )
        ]
        event_repo.list_by_session.return_value = events
        uc = EventUseCase(event_repo, score_repo, mock_kafka_producer)
        
        req = IngestEventRequest(session_id=uuid.UUID(SESSION_ID), event_type=EventType.TAB_SWITCH, occurred_at=base_time, metadata={})
        await uc.ingest_event(uuid.UUID(CANDIDATE_ID), uuid.UUID(ENTERPRISE_ID), req)
        
        score_repo.upsert.assert_called_once()
        assert score_repo.upsert.call_args[1]["cheating_score"] == 1.0


# ===================================================================
# FaceUseCase Tests
# ===================================================================

class TestFaceUseCase:

    @pytest.mark.asyncio
    async def test_verify_face_cache_hit(
        self, mock_redis, mock_face_detector, mock_candidate_client, mock_kafka_producer
    ):
        event_uc = AsyncMock()
        
        # Redis cache hit setup
        mock_redis.get.return_value = json.dumps({
            "url": "http://face-reference-url",
            "embedding": [0.1, 0.2, 0.3],
        })

        # Mock detector methods
        mock_face_detector.detect.return_value = DetectResult(has_face=True, face_count=1)
        mock_face_detector.compare.return_value = CompareResult(is_match=True, confidence=0.95)

        uc = FaceUseCase(
            detector=mock_face_detector,
            candidate_client=mock_candidate_client,
            event_uc=event_uc,
            producer=mock_kafka_producer,
            redis_client=mock_redis,
        )

        req = MagicMock(session_id=uuid.UUID(SESSION_ID), image_b64="data:image/jpeg;base64,probe_data")
        resp = await uc.verify_face(
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            req=req,
        )

        assert resp.is_match is True
        assert resp.confidence == 0.95
        assert resp.face_count == 1

        # Cache Hit -> should not call candidate client
        mock_candidate_client.get_face_reference_data.assert_not_called()
        mock_redis.get.assert_called_once_with(f"face_ref:{SESSION_ID}")
        
        # Verify Face OK event is ingested
        event_uc.ingest_event.assert_called_once()
        call_req = event_uc.ingest_event.call_args[0][2]
        assert call_req.event_type == EventType.PERIODIC_FACE_OK

    @pytest.mark.asyncio
    async def test_verify_face_cache_miss_and_fetch(
        self, mock_redis, mock_face_detector, mock_candidate_client, mock_kafka_producer
    ):
        event_uc = AsyncMock()
        
        # Redis cache miss setup
        mock_redis.get.return_value = None

        # Mock candidate client response
        mock_candidate_client.get_face_reference_data.return_value = {
            "url": "http://face-reference-url",
            "embedding": [0.1, 0.2, 0.3],
        }

        # Mock detector methods
        mock_face_detector.detect.return_value = DetectResult(has_face=True, face_count=1)
        mock_face_detector.compare.return_value = CompareResult(is_match=True, confidence=0.95)

        uc = FaceUseCase(
            detector=mock_face_detector,
            candidate_client=mock_candidate_client,
            event_uc=event_uc,
            producer=mock_kafka_producer,
            redis_client=mock_redis,
        )

        req = MagicMock(session_id=uuid.UUID(SESSION_ID), image_b64="data:image/jpeg;base64,probe_data")
        await uc.verify_face(
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            req=req,
        )

        # Cache Miss -> should fetch from client and store in redis
        mock_candidate_client.get_face_reference_data.assert_called_once_with(uuid.UUID(SESSION_ID))
        mock_redis.set.assert_called_once_with(
            f"face_ref:{SESSION_ID}",
            json.dumps({
                "url": "http://face-reference-url",
                "embedding": [0.1, 0.2, 0.3],
            }),
            ex=7200,
        )

    @pytest.mark.asyncio
    async def test_verify_face_not_registered_raises(
        self, mock_redis, mock_candidate_client, mock_face_detector, mock_kafka_producer
    ):
        event_uc = AsyncMock()
        mock_redis.get.return_value = None
        mock_candidate_client.get_face_reference_data.return_value = None  # no registration

        uc = FaceUseCase(
            detector=mock_face_detector,
            candidate_client=mock_candidate_client,
            event_uc=event_uc,
            producer=mock_kafka_producer,
            redis_client=mock_redis,
        )

        req = MagicMock(session_id=uuid.UUID(SESSION_ID), image_b64="data:image/jpeg;base64,probe_data")
        with pytest.raises(FaceNotRegisteredError):
            await uc.verify_face(
                session_id=uuid.UUID(SESSION_ID),
                candidate_id=uuid.UUID(CANDIDATE_ID),
                enterprise_id=uuid.UUID(ENTERPRISE_ID),
                req=req,
            )

    @pytest.mark.asyncio
    async def test_verify_face_multiple_faces_anomaly(
        self, mock_redis, mock_face_detector, mock_candidate_client, mock_kafka_producer
    ):
        event_uc = AsyncMock()
        mock_redis.get.return_value = json.dumps({
            "url": "http://face-reference-url",
            "embedding": [0.1, 0.2, 0.3],
        })

        # Mock multiple faces
        mock_face_detector.detect.return_value = DetectResult(has_face=True, face_count=2)
        mock_face_detector.compare.return_value = CompareResult(is_match=True, confidence=0.8)

        uc = FaceUseCase(
            detector=mock_face_detector,
            candidate_client=mock_candidate_client,
            event_uc=event_uc,
            producer=mock_kafka_producer,
            redis_client=mock_redis,
        )

        req = MagicMock(session_id=uuid.UUID(SESSION_ID), image_b64="data:image/jpeg;base64,probe_data")
        await uc.verify_face(
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            req=req,
        )

        # MULTIPLE_FACES event and PERIODIC_FACE_OK should be ingested
        assert event_uc.ingest_event.call_count == 2
        calls = event_uc.ingest_event.call_args_list
        event_types = [call[0][2].event_type for call in calls]
        assert EventType.MULTIPLE_FACES in event_types
        assert EventType.PERIODIC_FACE_OK in event_types

    @pytest.mark.asyncio
    async def test_verify_face_no_face_detected_early_return(
        self, mock_redis, mock_face_detector, mock_candidate_client, mock_kafka_producer
    ):
        event_uc = AsyncMock()
        mock_redis.get.return_value = json.dumps({
            "url": "http://face-reference-url",
            "embedding": [0.1, 0.2, 0.3],
        })

        # Mock no face detected
        mock_face_detector.detect.return_value = DetectResult(has_face=False, face_count=0)

        uc = FaceUseCase(
            detector=mock_face_detector,
            candidate_client=mock_candidate_client,
            event_uc=event_uc,
            producer=mock_kafka_producer,
            redis_client=mock_redis,
        )

        req = MagicMock(session_id=uuid.UUID(SESSION_ID), image_b64="data:image/jpeg;base64,probe_data")
        resp = await uc.verify_face(
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            req=req,
        )

        # Early return check
        assert resp.is_match is False
        assert resp.face_count == 0
        assert resp.confidence == 0.0

        # FACE_NOT_DETECTED event should be ingested
        event_uc.ingest_event.assert_called_once()
        call_req = event_uc.ingest_event.call_args[0][2]
        assert call_req.event_type == EventType.FACE_NOT_DETECTED

        # detector.compare should NOT be called
        mock_face_detector.compare.assert_not_called()

    @pytest.mark.asyncio
    async def test_verify_face_mismatch(
        self, mock_redis, mock_face_detector, mock_candidate_client, mock_kafka_producer
    ):
        event_uc = AsyncMock()
        mock_redis.get.return_value = json.dumps({
            "url": "http://face-reference-url",
            "embedding": [0.1, 0.2, 0.3],
        })

        # Mock mismatch
        mock_face_detector.detect.return_value = DetectResult(has_face=True, face_count=1)
        mock_face_detector.compare.return_value = CompareResult(is_match=False, confidence=0.2)

        uc = FaceUseCase(
            detector=mock_face_detector,
            candidate_client=mock_candidate_client,
            event_uc=event_uc,
            producer=mock_kafka_producer,
            redis_client=mock_redis,
        )

        req = MagicMock(session_id=uuid.UUID(SESSION_ID), image_b64="data:image/jpeg;base64,probe_data")
        resp = await uc.verify_face(
            session_id=uuid.UUID(SESSION_ID),
            candidate_id=uuid.UUID(CANDIDATE_ID),
            enterprise_id=uuid.UUID(ENTERPRISE_ID),
            req=req,
        )

        assert resp.is_match is False
        assert resp.confidence == 0.2

        # IDENTITY_MISMATCH event should be ingested
        event_uc.ingest_event.assert_called_once()
        call_req = event_uc.ingest_event.call_args[0][2]
        assert call_req.event_type == EventType.IDENTITY_MISMATCH
        assert call_req.metadata["confidence"] == 0.2
