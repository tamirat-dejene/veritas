"""
Unit tests for app.grading.ai_client — HTTP interactions with HF grading space.
"""
import pytest
import httpx
import respx
from unittest.mock import patch

from app.grading.ai_client import evaluate_short_answers, build_batch_payload
from app.config import settings


# ===================================================================
# build_batch_payload
# ===================================================================


class TestBuildBatchPayload:
    def test_wraps_items_in_dict(self):
        items = [{"question_id": "q1", "student_text": "answer"}]
        result = build_batch_payload(items)
        assert result == {"items": items}

    def test_empty_items(self):
        result = build_batch_payload([])
        assert result == {"items": []}


# ===================================================================
# evaluate_short_answers
# ===================================================================


class TestEvaluateShortAnswers:
    """Test the HTTP call to HF space with mocked responses."""

    BATCH = [
        {
            "question_id": "q1",
            "student_text": "cell energy",
            "expected_answer": "mitochondria",
            "keywords": ["mitochondria"],
        },
        {
            "question_id": "q2",
            "student_text": "dna",
            "expected_answer": "deoxyribonucleic acid",
            "keywords": ["dna"],
        },
    ]

    @pytest.mark.asyncio
    @respx.mock
    async def test_successful_response(self):
        respx.post(settings.HF_EVALUATE_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "graded_items": [
                        {"question_id": "q1", "score_percentage": 0.9},
                        {"question_id": "q2", "score_percentage": 0.7},
                    ]
                },
            )
        )
        scores = await evaluate_short_answers(self.BATCH)
        assert scores == {"q1": 0.9, "q2": 0.7}

    @pytest.mark.asyncio
    async def test_empty_batch_skips_call(self):
        scores = await evaluate_short_answers([])
        assert scores == {}

    @pytest.mark.asyncio
    @respx.mock
    async def test_timeout_returns_empty(self):
        respx.post(settings.HF_EVALUATE_URL).mock(
            side_effect=httpx.ReadTimeout("timed out")
        )
        scores = await evaluate_short_answers(self.BATCH)
        assert scores == {}

    @pytest.mark.asyncio
    @respx.mock
    async def test_http_error_returns_empty(self):
        respx.post(settings.HF_EVALUATE_URL).mock(
            return_value=httpx.Response(500, text="Internal Server Error")
        )
        scores = await evaluate_short_answers(self.BATCH)
        assert scores == {}

    @pytest.mark.asyncio
    @respx.mock
    async def test_network_error_returns_empty(self):
        respx.post(settings.HF_EVALUATE_URL).mock(
            side_effect=httpx.ConnectError("connection refused")
        )
        scores = await evaluate_short_answers(self.BATCH)
        assert scores == {}

    @pytest.mark.asyncio
    @respx.mock
    async def test_malformed_response_returns_empty(self):
        respx.post(settings.HF_EVALUATE_URL).mock(
            return_value=httpx.Response(200, json={"unexpected": "schema"})
        )
        scores = await evaluate_short_answers(self.BATCH)
        # "graded_items" key missing → empty list → empty scores
        assert scores == {}

    @pytest.mark.asyncio
    @respx.mock
    async def test_partial_graded_items(self):
        respx.post(settings.HF_EVALUATE_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "graded_items": [
                        {"question_id": "q1", "score_percentage": 0.5},
                        # q2 missing from response
                    ]
                },
            )
        )
        scores = await evaluate_short_answers(self.BATCH)
        assert scores == {"q1": 0.5}
        assert "q2" not in scores

    @pytest.mark.asyncio
    @respx.mock
    async def test_passes_authorization_header_when_token_is_configured(self):
        with patch.object(settings, "HF_TOKEN", "mock-hf-token-abc"):
            route = respx.post(settings.HF_EVALUATE_URL).mock(
                return_value=httpx.Response(
                    200,
                    json={
                        "graded_items": [
                            {"question_id": "q1", "score_percentage": 0.9},
                        ]
                    },
                )
            )
            scores = await evaluate_short_answers(self.BATCH)
            assert scores == {"q1": 0.9}
            
            # Verify the Authorization header was sent
            request = route.calls.last.request
            assert "Authorization" in request.headers
            assert request.headers["Authorization"] == "Bearer mock-hf-token-abc"
