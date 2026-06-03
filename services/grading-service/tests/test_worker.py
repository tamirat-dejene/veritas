"""
Unit tests for app.grading.worker — Kafka consumer message handling.
"""
import json
# pyrefly: ignore [missing-import]  
import pytest
from app.domain.models import GradingStatus, QuestionGradingStatus, QuestionType
from unittest.mock import AsyncMock, MagicMock, patch

from app.grading.worker import _handle_message, process_incoming_event
from app.grading.grader import ExamGradeReport, QuestionResult
from app.grading.models import GradingPayload
from tests.conftest import (
    _base_event,
    _mcq_item,
    SESSION_ID,
    CANDIDATE_ID,
    EXAM_ID,
    ENTERPRISE_ID,
    ENROLLMENT_ID,
    EVENT_ID,
    QUESTION_ID_1,
    SQ_ID_1,
)


# ===================================================================
# _handle_message
# ===================================================================


def _kafka_msg(payload: dict) -> MagicMock:
    """Create a mock Kafka message with a JSON-encoded value."""
    msg = MagicMock()
    msg.value = json.dumps(payload).encode("utf-8")
    msg.topic = "exam.session.ready_for_grading"
    msg.partition = 0
    msg.offset = 123
    return msg


class TestHandleMessage:

    @pytest.mark.asyncio
    async def test_valid_event_is_processed(self):
        payload = _base_event()
        msg = _kafka_msg(payload)
        mock_pool = AsyncMock()
        mock_client = AsyncMock()

        with patch(
            "app.grading.worker.process_incoming_event",
            new_callable=AsyncMock,
        ) as mock_process:
            await _handle_message(msg, mock_pool, mock_client)
            mock_process.assert_awaited_once()
            # The first arg should be the payload dict
            call_payload = mock_process.call_args[0][0]
            assert call_payload["event_id"] == EVENT_ID

    @pytest.mark.asyncio
    async def test_wrong_event_type_is_ignored(self):
        payload = _base_event(event_type="exam.published")
        msg = _kafka_msg(payload)
        mock_pool = AsyncMock()
        mock_client = AsyncMock()

        with patch(
            "app.grading.worker.process_incoming_event",
            new_callable=AsyncMock,
        ) as mock_process:
            await _handle_message(msg, mock_pool, mock_client)
            mock_process.assert_not_called()

    @pytest.mark.asyncio
    async def test_malformed_json_does_not_raise(self):
        msg = MagicMock()
        msg.value = b"not valid json"
        msg.topic = "exam.session.ready_for_grading"
        msg.partition = 0
        msg.offset = 123
        mock_pool = AsyncMock()
        mock_client = AsyncMock()

        # Should log error but not raise
        await _handle_message(msg, mock_pool, mock_client)

    @pytest.mark.asyncio
    async def test_processing_error_does_not_propagate(self):
        payload = _base_event()
        msg = _kafka_msg(payload)
        mock_pool = AsyncMock()
        mock_client = AsyncMock()

        with patch(
            "app.grading.worker.process_incoming_event",
            new_callable=AsyncMock,
            side_effect=RuntimeError("DB down"),
        ):
            # Should log exception but not raise
            await _handle_message(msg, mock_pool, mock_client)


# ===================================================================
# process_incoming_event
# ===================================================================


class TestProcessIncomingEvent:

    @pytest.mark.asyncio
    async def test_grades_and_saves(self):
        slim_trigger = _base_event()
        payload_dict = _base_event(items=[
            _mcq_item(QUESTION_ID_1, SQ_ID_1, selected_ids=["opt-a", "opt-b"])
        ])
        grading_payload = GradingPayload.model_validate(payload_dict)

        mock_pool = AsyncMock()
        mock_client = AsyncMock()
        mock_client.fetch_grading_payload.return_value = grading_payload

        with patch(
            "app.grading.worker.GradingRepository",
        ) as MockRepoClass:
            mock_repo = AsyncMock()
            MockRepoClass.return_value = mock_repo

            report = await process_incoming_event(slim_trigger, mock_pool, mock_client)

            assert isinstance(report, ExamGradeReport)
            assert report.total_awarded_points == 10.0
            mock_client.fetch_grading_payload.assert_awaited_once_with(
                session_id=SESSION_ID,
                enterprise_id=ENTERPRISE_ID,
            )
            mock_repo.save_grading_report.assert_awaited_once()

    @pytest.mark.asyncio
    async def test_grades_saves_and_publishes(self):
        slim_trigger = _base_event()
        payload_dict = _base_event(
            items=[
                _mcq_item(QUESTION_ID_1, SQ_ID_1, selected_ids=["opt-a", "opt-b"])
            ],
            candidate_name="John Doe",
            candidate_email="john@example.com",
            exam_title="Physics 101",
        )
        grading_payload = GradingPayload.model_validate(payload_dict)

        mock_pool = AsyncMock()
        mock_client = AsyncMock()
        mock_client.fetch_grading_payload.return_value = grading_payload
        mock_producer = AsyncMock()

        with patch(
            "app.grading.worker.GradingRepository",
        ) as MockRepoClass:
            mock_repo = AsyncMock()
            MockRepoClass.return_value = mock_repo

            report = await process_incoming_event(
                slim_trigger,
                mock_pool,
                mock_client,
                producer=mock_producer,
            )

            assert isinstance(report, ExamGradeReport)
            mock_producer.publish.assert_awaited_once()
            topic, event_payload = mock_producer.publish.call_args[0]
            assert topic == "grading.session.completed"
            assert event_payload["candidate_name"] == "John Doe"
            assert event_payload["candidate_email"] == "john@example.com"
            assert event_payload["exam_title"] == "Physics 101"
            assert event_payload["percentage"] == 100.0

