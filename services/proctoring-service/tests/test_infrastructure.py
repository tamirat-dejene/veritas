"""
Unit tests for the infrastructure clients and consumers in proctoring-service.
"""
import json
import uuid
from unittest.mock import AsyncMock, MagicMock

import httpx
# pyrefly: ignore [missing-import]
import pytest
# pyrefly: ignore [missing-import]
import respx

from app.domain.errors import InternalServiceError, SessionNotFoundError
from app.infrastructure.client.candidate_client import CandidateServiceClient
from app.infrastructure.face.remote_face_detector import RemoteFaceDetector
from app.infrastructure.kafka.consumer import _handle_message
from tests.conftest import CANDIDATE_ID, ENTERPRISE_ID, SESSION_ID


# ===================================================================
# CandidateServiceClient Tests
# ===================================================================

class TestCandidateServiceClient:

    @respx.mock
    @pytest.mark.asyncio
    async def test_get_face_reference_data_success(self):
        client = CandidateServiceClient("http://candidate-service")
        
        # Mock successful candidate service response
        session_url = f"http://candidate-service/sessions/{SESSION_ID}"
        respx.get(session_url).mock(
            return_value=httpx.Response(
                200,
                json={
                    "data": {
                        "faceRegisteredUrl": "http://registered-url",
                        "faceRegisteredEmbedding": [0.1, 0.2, 0.3],
                    }
                },
            )
        )

        res = await client.get_face_reference_data(uuid.UUID(SESSION_ID))
        assert res is not None
        assert res["url"] == "http://registered-url"
        assert res["embedding"] == [0.1, 0.2, 0.3]

    @respx.mock
    @pytest.mark.asyncio
    async def test_get_face_reference_data_none(self):
        client = CandidateServiceClient("http://candidate-service")
        session_url = f"http://candidate-service/sessions/{SESSION_ID}"
        respx.get(session_url).mock(
            return_value=httpx.Response(
                200,
                json={"data": {}},
            )
        )

        res = await client.get_face_reference_data(uuid.UUID(SESSION_ID))
        assert res is None

    @respx.mock
    @pytest.mark.asyncio
    async def test_get_face_reference_data_not_found(self):
        client = CandidateServiceClient("http://candidate-service")
        session_url = f"http://candidate-service/sessions/{SESSION_ID}"
        respx.get(session_url).mock(return_value=httpx.Response(404))

        with pytest.raises(SessionNotFoundError):
            await client.get_face_reference_data(uuid.UUID(SESSION_ID))

    @respx.mock
    @pytest.mark.asyncio
    async def test_get_face_reference_data_internal_error(self):
        client = CandidateServiceClient("http://candidate-service")
        session_url = f"http://candidate-service/sessions/{SESSION_ID}"
        
        # Mock 500 error
        respx.get(session_url).mock(return_value=httpx.Response(500))
        with pytest.raises(InternalServiceError):
            await client.get_face_reference_data(uuid.UUID(SESSION_ID))

        # Mock connection timeout / unreachable error
        respx.get(session_url).mock(side_effect=httpx.ConnectTimeout("Timeout"))
        with pytest.raises(InternalServiceError):
            await client.get_face_reference_data(uuid.UUID(SESSION_ID))


# ===================================================================
# RemoteFaceDetector Tests
# ===================================================================

class TestRemoteFaceDetector:

    @respx.mock
    @pytest.mark.asyncio
    async def test_detect_multiple_faces(self):
        detector = RemoteFaceDetector("http://face-api")
        respx.post("http://face-api/represent").mock(
            return_value=httpx.Response(
                200,
                json={
                    "results": [
                        {"embedding": [0.1]},
                        {"embedding": [0.2]},
                    ]
                },
            )
        )

        res = await detector.detect("data:image/jpeg;base64,abcdef")
        assert res.has_face is True
        assert res.face_count == 2

    @respx.mock
    @pytest.mark.asyncio
    async def test_detect_no_face_400(self):
        detector = RemoteFaceDetector("http://face-api")
        respx.post("http://face-api/represent").mock(return_value=httpx.Response(400))

        res = await detector.detect("abcdef")
        assert res.has_face is False
        assert res.face_count == 0

    @respx.mock
    @pytest.mark.asyncio
    async def test_detect_failsafe_on_server_error(self):
        detector = RemoteFaceDetector("http://face-api")
        respx.post("http://face-api/represent").mock(return_value=httpx.Response(500))

        res = await detector.detect("abcdef")
        assert res.has_face is True
        assert res.face_count == 1

    @respx.mock
    @pytest.mark.asyncio
    async def test_compare_match(self):
        detector = RemoteFaceDetector("http://face-api")
        respx.post("http://face-api/compare").mock(
            return_value=httpx.Response(
                200,
                json={
                    "verified": True,
                    "distance": 0.1,
                    "threshold": 0.4,
                },
            )
        )

        res = await detector.compare([0.1, 0.2], "abcdef")
        assert res.is_match is True
        assert res.confidence == 0.75  # 1 - 0.1/0.4 = 0.75

    @respx.mock
    @pytest.mark.asyncio
    async def test_compare_no_face_400(self):
        detector = RemoteFaceDetector("http://face-api")
        respx.post("http://face-api/compare").mock(return_value=httpx.Response(400))

        res = await detector.compare([0.1, 0.2], "abcdef")
        assert res.is_match is False
        assert res.confidence == 0.0

    @respx.mock
    @pytest.mark.asyncio
    async def test_compare_error(self):
        detector = RemoteFaceDetector("http://face-api")
        respx.post("http://face-api/compare").mock(return_value=httpx.Response(500))

        res = await detector.compare([0.1, 0.2], "abcdef")
        assert res.is_match is False
        assert res.confidence == 0.0

    @respx.mock
    @pytest.mark.asyncio
    async def test_compare_unreachable_raises(self):
        detector = RemoteFaceDetector("http://face-api")
        respx.post("http://face-api/compare").mock(side_effect=httpx.ConnectError("Unreachable"))

        with pytest.raises(httpx.RequestError):
            await detector.compare([0.1, 0.2], "abcdef")


# ===================================================================
# Kafka Consumer Message Handler Tests
# ===================================================================

class TestKafkaConsumerHandler:

    @pytest.mark.asyncio
    async def test_handle_message_success(self, mock_kafka_producer):
        event_uc = AsyncMock()
        score_mock = MagicMock(cheating_score=34.2, event_count=4)
        event_uc.get_score.return_value = score_mock

        # Setup mock Kafka message
        msg = MagicMock()
        msg.value = json.dumps({
            "session_id": SESSION_ID,
            "enterprise_id": ENTERPRISE_ID,
        }).encode("utf-8")

        await _handle_message(msg, event_uc, mock_kafka_producer)

        event_uc.get_score.assert_called_once_with(uuid.UUID(SESSION_ID))
        mock_kafka_producer.publish.assert_called_once_with(
            "proctoring.cheating_score.updated",
            {
                "session_id": SESSION_ID,
                "enterprise_id": ENTERPRISE_ID,
                "cheating_score": 34.2,
                "event_count": 4,
                "is_final": True,
            },
        )

    @pytest.mark.asyncio
    async def test_handle_message_no_score(self, mock_kafka_producer):
        event_uc = AsyncMock()
        event_uc.get_score.return_value = None

        msg = MagicMock()
        msg.value = json.dumps({
            "session_id": SESSION_ID,
            "enterprise_id": ENTERPRISE_ID,
        }).encode("utf-8")

        await _handle_message(msg, event_uc, mock_kafka_producer)

        event_uc.get_score.assert_called_once_with(uuid.UUID(SESSION_ID))
        mock_kafka_producer.publish.assert_not_called()

    @pytest.mark.asyncio
    async def test_handle_message_json_decode_error_ignored(self, mock_kafka_producer):
        event_uc = AsyncMock()
        msg = MagicMock()
        msg.value = b"invalid json"

        # Should not raise exception
        await _handle_message(msg, event_uc, mock_kafka_producer)

        event_uc.get_score.assert_not_called()
        mock_kafka_producer.publish.assert_not_called()
