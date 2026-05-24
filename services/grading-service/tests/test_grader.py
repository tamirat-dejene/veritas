"""
Unit tests for app.grading.grader — MCQ scoring, SA batch prep, and full pipeline.
"""

# pyrefly: ignore [missing-import]  
import pytest
from unittest.mock import AsyncMock, patch
from typing import Any, Dict

from app.grading.grader import (
    _grade_mcq,
    _prepare_sa_batch_item,
    grade_exam,
    QuestionResult,
    ExamGradeReport,
)
from app.grading.models import GradingItem, EvaluationCriteria
from tests.conftest import (
    _base_event,
    _mcq_item,
    _sa_item,
    QUESTION_ID_1,
    QUESTION_ID_2,
    QUESTION_ID_3,
    SQ_ID_1,
    SQ_ID_2,
    SQ_ID_3,
    SESSION_ID,
    CANDIDATE_ID,
    EXAM_ID,
    ENTERPRISE_ID,
    ENROLLMENT_ID,
)


# ===================================================================
# MCQ grading
# ===================================================================


class TestGradeMCQ:
    """Tests for deterministic multiple-choice grading."""

    def _item(self, **kw) -> GradingItem:
        defaults = dict(
            question_id=QUESTION_ID_1,
            session_question_id=SQ_ID_1,
            question_type="multiple_choice",
            content="Pick",
            title="MCQ",
            topic="gen",
            points=10.0,
            negative_points=0.0,
            correct_option_ids=["a", "b"],
            has_answer=True,
            candidate_answer={"selectedOptionIds": ["a", "b"]},
        )
        defaults.update(kw)
        return GradingItem(**defaults)

    def test_correct_answer_full_points(self):
        result = _grade_mcq(self._item())
        assert result.status == "correct"
        assert result.awarded_points == 10.0

    def test_incorrect_answer_zero_points(self):
        item = self._item(candidate_answer={"selectedOptionIds": ["c"]})
        result = _grade_mcq(item)
        assert result.status == "incorrect"
        assert result.awarded_points == 0.0

    def test_incorrect_with_negative_points_floors_at_zero(self):
        item = self._item(
            negative_points=3.0,
            candidate_answer={"selectedOptionIds": ["c"]},
        )
        result = _grade_mcq(item)
        assert result.status == "incorrect"
        assert result.awarded_points == 0.0  # max(0, 0 - 3) = 0

    def test_skipped_unanswered(self):
        item = self._item(has_answer=False, candidate_answer=None)
        result = _grade_mcq(item)
        assert result.status == "skipped"
        assert result.awarded_points == 0.0

    def test_partial_selection_is_incorrect(self):
        item = self._item(candidate_answer={"selectedOptionIds": ["a"]})
        result = _grade_mcq(item)
        assert result.status == "incorrect"

    def test_superset_selection_is_incorrect(self):
        item = self._item(candidate_answer={"selectedOptionIds": ["a", "b", "c"]})
        result = _grade_mcq(item)
        assert result.status == "incorrect"

    def test_empty_selection_is_incorrect(self):
        item = self._item(candidate_answer={"selectedOptionIds": []})
        result = _grade_mcq(item)
        # correct_option_ids = ["a","b"], selected = [], so mismatch
        assert result.status == "incorrect"

    def test_result_fields_populated(self):
        result = _grade_mcq(self._item())
        assert result.question_id == QUESTION_ID_1
        assert result.session_question_id == SQ_ID_1
        assert result.question_type == "multiple_choice"
        assert result.max_points == 10.0


# ===================================================================
# Short-answer batch preparation
# ===================================================================


class TestPrepareSABatch:
    """Tests for _prepare_sa_batch_item helper."""

    def _item(self, **kw) -> GradingItem:
        defaults = dict(
            question_id=QUESTION_ID_2,
            session_question_id=SQ_ID_2,
            question_type="short_answer",
            content="Describe",
            title="SA",
            topic="bio",
            points=20.0,
            expected_answer="The mitochondria",
            evaluation_criteria=EvaluationCriteria(keywords=["mitochondria"]),
            has_answer=True,
            candidate_answer={"text": "Powerhouse of cell"},
        )
        defaults.update(kw)
        return GradingItem(**defaults)

    def test_builds_batch_dict(self):
        result = _prepare_sa_batch_item(self._item())
        assert result is not None
        assert result["question_id"] == QUESTION_ID_2
        assert result["student_text"] == "Powerhouse of cell"
        assert result["expected_answer"] == "The mitochondria"
        assert result["keywords"] == ["mitochondria"]

    def test_returns_none_when_no_answer(self):
        item = self._item(has_answer=False, candidate_answer=None)
        assert _prepare_sa_batch_item(item) is None

    def test_returns_none_when_empty_text(self):
        item = self._item(candidate_answer={"text": ""})
        assert _prepare_sa_batch_item(item) is None

    def test_returns_none_when_text_is_none(self):
        item = self._item(candidate_answer={"text": None})
        assert _prepare_sa_batch_item(item) is None

    def test_no_evaluation_criteria_gives_empty_keywords(self):
        item = self._item(evaluation_criteria=None)
        result = _prepare_sa_batch_item(item)
        assert result is not None
        assert result["keywords"] == []


# ===================================================================
# Full pipeline
# ===================================================================


class TestGradeExam:
    """Integration-style tests for the grade_exam pipeline (AI call mocked)."""

    @pytest.mark.asyncio
    async def test_mcq_only_exam(self):
        from app.grading.models import GradingPayload
        payload_dict = _base_event(items=[
            _mcq_item(QUESTION_ID_1, SQ_ID_1, selected_ids=["opt-a", "opt-b"]),
            _mcq_item(QUESTION_ID_2, SQ_ID_2, points=5.0, selected_ids=["wrong"], correct_ids=["opt-a"]),
        ])
        payload = GradingPayload.model_validate(payload_dict)
        report = await grade_exam("test-event-id", payload)
        assert isinstance(report, ExamGradeReport)
        assert report.total_max_points == 15.0
        assert report.total_awarded_points == 10.0
        assert len(report.question_results) == 2

    @pytest.mark.asyncio
    async def test_sa_only_exam_with_ai(self):
        from app.grading.models import GradingPayload
        payload_dict = _base_event(items=[
            _sa_item(QUESTION_ID_1, SQ_ID_1, points=20.0),
        ])
        payload = GradingPayload.model_validate(payload_dict)
        ai_scores = {QUESTION_ID_1: 0.85}
        with patch("app.grading.grader.evaluate_short_answers", new_callable=AsyncMock, return_value=ai_scores):
            report = await grade_exam("test-event-id", payload)
        assert report.total_max_points == 20.0
        assert report.total_awarded_points == 17.0  # 0.85 * 20 = 17.0
        assert report.question_results[0].status == "ai_graded"

    @pytest.mark.asyncio
    async def test_mixed_exam(self):
        from app.grading.models import GradingPayload
        payload_dict = _base_event(items=[
            _mcq_item(QUESTION_ID_1, SQ_ID_1, points=10.0, selected_ids=["opt-a", "opt-b"]),
            _sa_item(QUESTION_ID_2, SQ_ID_2, points=20.0),
        ])
        payload = GradingPayload.model_validate(payload_dict)
        ai_scores = {QUESTION_ID_2: 0.5}
        with patch("app.grading.grader.evaluate_short_answers", new_callable=AsyncMock, return_value=ai_scores):
            report = await grade_exam("test-event-id", payload)
        assert report.total_max_points == 30.0
        assert report.total_awarded_points == 20.0  # 10 + 0.5*20
        assert report.percentage == pytest.approx(66.67, abs=0.01)

    @pytest.mark.asyncio
    async def test_skipped_sa_not_sent_to_ai(self):
        from app.grading.models import GradingPayload
        payload_dict = _base_event(items=[
            _sa_item(QUESTION_ID_1, SQ_ID_1, has_answer=False),
        ])
        payload = GradingPayload.model_validate(payload_dict)
        mock_ai = AsyncMock(return_value={})
        with patch("app.grading.grader.evaluate_short_answers", mock_ai):
            report = await grade_exam("test-event-id", payload)
        mock_ai.assert_not_called()
        assert report.question_results[0].status == "skipped"

    @pytest.mark.asyncio
    async def test_ai_failure_graceful_degradation(self):
        """When AI returns empty scores, SA items get 0 points with 'skipped' status."""
        from app.grading.models import GradingPayload
        payload_dict = _base_event(items=[
            _sa_item(QUESTION_ID_1, SQ_ID_1, points=20.0),
        ])
        payload = GradingPayload.model_validate(payload_dict)
        with patch("app.grading.grader.evaluate_short_answers", new_callable=AsyncMock, return_value={}):
            report = await grade_exam("test-event-id", payload)
        assert report.total_awarded_points == 0.0
        assert report.question_results[0].status == "skipped"

    @pytest.mark.asyncio
    async def test_empty_exam(self):
        from app.grading.models import GradingPayload
        payload_dict = _base_event(items=[])
        payload = GradingPayload.model_validate(payload_dict)
        report = await grade_exam("test-event-id", payload)
        assert report.total_max_points == 0.0
        assert report.percentage == 0.0
        assert len(report.question_results) == 0

    @pytest.mark.asyncio
    async def test_unknown_question_type_skipped(self):
        from app.grading.models import GradingPayload
        payload_dict = _base_event(items=[
            {
                "question_id": QUESTION_ID_1,
                "session_question_id": SQ_ID_1,
                "question_type": "essay",  # unknown type
                "content": "Write an essay",
                "title": "Essay Q",
                "topic": "gen",
                "points": 30.0,
                "has_answer": True,
                "candidate_answer": {"text": "something"},
            }
        ])
        payload = GradingPayload.model_validate(payload_dict)
        report = await grade_exam("test-event-id", payload)
        assert len(report.question_results) == 0
        assert report.total_max_points == 0.0
