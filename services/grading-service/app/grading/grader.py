"""
Core grading engine — deterministic MCQ scoring + AI-delegated short-answer scoring.

This module is the heart of the grading pipeline.  It:
  1. Parses and validates the incoming event via Pydantic models.
  2. Grades multiple-choice questions deterministically (no network).
  3. Collects short-answer items and delegates them in a **single batch**
     to the AI client.
  4. Aggregates per-question results into a final exam grade report.

Design decisions:
  • Unanswered items (has_answer=False or candidate_answer is None) are
    awarded 0 points immediately — no KeyError, no AI call.
  • MCQ scores floor at 0 unless the platform explicitly allows negatives.
  • Short-answer score = score_percentage × max_points.
"""
from __future__ import annotations

import logging
from dataclasses import dataclass, field
from typing import Any

from app.domain.models import QuestionType, QuestionGradingStatus
from app.grading.models import GradingPayload, GradingItem
from .ai_client import evaluate_short_answers

logger = logging.getLogger("grading.grader")


# ---------------------------------------------------------------------------
# Result containers
# ---------------------------------------------------------------------------

@dataclass
class QuestionResult:
    """Grade result for a single question."""
    question_id: str
    session_question_id: str
    question_type: QuestionType
    title: str
    content: str
    candidate_answer: Any
    max_points: float
    awarded_points: float
    status: QuestionGradingStatus  # "correct" | "incorrect" | "partial" | "skipped" | "ai_graded"


@dataclass
class ExamGradeReport:
    """Aggregated grading output for the entire exam session."""
    event_id: str
    session_id: str
    candidate_id: str
    exam_id: str
    enterprise_id: str
    enrollment_id: str
    total_max_points: float = 0.0
    total_awarded_points: float = 0.0
    question_results: list[QuestionResult] = field(default_factory=list)

    @property
    def percentage(self) -> float:
        if self.total_max_points == 0:
            return 0.0
        return round(
            (self.total_awarded_points / self.total_max_points) * 100, 2,
        )


# ---------------------------------------------------------------------------
# MCQ grading (deterministic, no network)
# ---------------------------------------------------------------------------

def _grade_mcq(item: GradingItem) -> QuestionResult:
    """
    Grade a multiple-choice question by exact set-match of selected vs correct
    option IDs.

    Rules:
      • Exact match  → full ``points``.
      • Mismatch     → ``-negative_points`` (floored at 0).
      • Skipped      → 0.
    """
    if not item.has_answer or item.candidate_answer is None:
        return QuestionResult(
            question_id=item.question_id,
            session_question_id=item.session_question_id,
            question_type=item.question_type,
            title=item.title,
            content=item.content,
            candidate_answer=item.candidate_answer,
            max_points=item.points,
            awarded_points=0.0,
            status=QuestionGradingStatus.skipped,
        )

    selected = set(item.candidate_answer.get("selectedOptionIds", []))
    correct = set(item.correct_option_ids or [])

    if selected == correct:
        return QuestionResult(
            question_id=item.question_id,
            session_question_id=item.session_question_id,
            question_type=item.question_type,
            title=item.title,
            content=item.content,
            candidate_answer=item.candidate_answer,
            max_points=item.points,
            awarded_points=item.points,
            status=QuestionGradingStatus.correct,
        )

    # Incorrect — wrong answer starts at 0 and subtracts negative_points.
    # The final question score must never drop below 0.
    awarded = max(0.0, 0.0 - item.negative_points)

    return QuestionResult(
        question_id=item.question_id,
        session_question_id=item.session_question_id,
        question_type=item.question_type,
        title=item.title,
        content=item.content,
        candidate_answer=item.candidate_answer,
        max_points=item.points,
        awarded_points=awarded,
        status=QuestionGradingStatus.incorrect,
    )


# ---------------------------------------------------------------------------
# Short-answer batch preparation
# ---------------------------------------------------------------------------

def _prepare_sa_batch_item(item: GradingItem) -> dict[str, Any] | None:
    """
    Convert a short-answer GradingItem into the dict shape expected by the
    Hugging Face API, or return ``None`` if the item should be skipped.
    """
    if not item.has_answer or item.candidate_answer is None:
        return None

    student_text = item.candidate_answer.get("text")
    if not student_text:
        return None

    keywords: list[str] = []
    if item.evaluation_criteria:
        keywords = item.evaluation_criteria.keywords

    return {
        "question_id": item.question_id,
        "student_text": student_text,
        "expected_answer": item.expected_answer or "",
        "keywords": keywords,
    }


# ---------------------------------------------------------------------------
# Main grading pipeline
# ---------------------------------------------------------------------------

async def grade_exam(event_id: str, payload: GradingPayload) -> ExamGradeReport:
    """
    Full grading pipeline for a single exam session.

    Accepts a ``GradingPayload`` fetched from the candidate-service internal
    endpoint and an ``event_id`` from the original slim trigger event.

    Steps:
      1. Grade MCQs deterministically.
      2. Collect short-answer items → single AI batch request.
      3. Merge AI scores back and build the final report.
    """
    logger.info(
        "Grading exam session=%s  exam=%s  candidate=%s  items=%d",
        payload.session_id,
        payload.exam_id,
        payload.candidate_id,
        len(payload.items),
    )

    report = ExamGradeReport(
        event_id=event_id,
        session_id=payload.session_id,
        candidate_id=payload.candidate_id,
        exam_id=payload.exam_id,
        enterprise_id=payload.enterprise_id,
        enrollment_id=payload.enrollment_id,
    )

    # ----- 1. Deterministic MCQ pass ---------------------------------------
    sa_items: list = []
    sa_batch: list[dict] = []
    mcq_count = 0
    sa_count = 0

    for item in payload.items:
        if item.question_type in (QuestionType.MCQ, QuestionType.TrueFalse):
            if not item.correct_option_ids:
                logger.warning(
                    "MCQ/TF  q=%s  has no correct_option_ids — will be graded as incorrect",
                    item.question_id,
                )
            result = _grade_mcq(item)
            report.question_results.append(result)
            report.total_max_points += item.points
            report.total_awarded_points += result.awarded_points
            mcq_count += 1
            logger.debug(
                "MCQ/TF  q=%s  status=%s  awarded=%.2f / %.2f",
                item.question_id,
                result.status,
                result.awarded_points,
                item.points,
            )

        elif item.question_type == QuestionType.ShortAnswer:
            # Immediate skip for unanswered
            if not item.has_answer or item.candidate_answer is None:
                result = QuestionResult(
                    question_id=item.question_id,
                    session_question_id=item.session_question_id,
                    question_type=item.question_type,
                    title=item.title,
                    content=item.content,
                    candidate_answer=item.candidate_answer,
                    max_points=item.points,
                    awarded_points=0.0,
                    status=QuestionGradingStatus.skipped,
                )
                report.question_results.append(result)
                report.total_max_points += item.points
                logger.debug("SA/Essay   q=%s  skipped (no answer)", item.question_id)
                continue

            batch_item = _prepare_sa_batch_item(item)
            if batch_item is None:
                # candidate_answer exists but has no text
                logger.warning(
                    "SA/Essay   q=%s  has answer but empty text — scoring as skipped",
                    item.question_id,
                )
                result = QuestionResult(
                    question_id=item.question_id,
                    session_question_id=item.session_question_id,
                    question_type=item.question_type,
                    title=item.title,
                    content=item.content,
                    candidate_answer=item.candidate_answer,
                    max_points=item.points,
                    awarded_points=0.0,
                    status=QuestionGradingStatus.skipped,
                )
                report.question_results.append(result)
                report.total_max_points += item.points
                continue

            sa_count += 1
            sa_items.append(item)
            sa_batch.append(batch_item)

        elif item.question_type == QuestionType.Essay:
            # Come back to this
            logger.info(
                "Essay q=%s — skipping for now",
                item.question_id,
            )
            result = QuestionResult(
                question_id=item.question_id,
                session_question_id=item.session_question_id,
                question_type=item.question_type,
                title=item.title,
                max_points=item.points,
                awarded_points=0.0,
                status=QuestionGradingStatus.skipped,
            )
            report.question_results.append(result)
            report.total_max_points += item.points
            continue
        else:
            logger.warning(
                "Unknown question_type '%s' for q=%s — skipping.",
                item.question_type,
                item.question_id,
            )

    # ----- 3. AI batch call ------------------------------------------------
    ai_scores: dict[str, float] = {}
    if sa_batch:
        logger.info(
            "Dispatching %d short-answer items to AI for evaluation.",
            len(sa_batch),
        )
        ai_scores = await evaluate_short_answers(sa_batch)

    # ----- 4. Merge AI scores into report ----------------------------------
    for item in sa_items:
        score_pct = ai_scores.get(item.question_id, 0.0)
        awarded = round(score_pct * item.points, 2)

        result = QuestionResult(
            question_id=item.question_id,
            session_question_id=item.session_question_id,
            question_type=item.question_type,
            title=item.title,
            content=item.content,
            candidate_answer=item.candidate_answer,
            max_points=item.points,
            awarded_points=awarded,
            status=QuestionGradingStatus.ai_graded,
        )
        report.question_results.append(result)
        report.total_max_points += item.points
        report.total_awarded_points += awarded
        logger.debug(
            "SA   q=%s  pct=%.2f  awarded=%.2f / %.2f",
            item.question_id,
            score_pct,
            awarded,
            item.points,
        )

    logger.info(
        "Grading complete  session=%s  mcq=%d  sa=%d  "
        "total=%.2f / %.2f  (%.1f%%)",
        payload.session_id,
        mcq_count,
        sa_count,
        report.total_awarded_points,
        report.total_max_points,
        report.percentage,
    )
    return report
