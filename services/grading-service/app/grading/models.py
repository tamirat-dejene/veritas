"""
Pydantic models for the exam.session.ready_for_grading event payload.

These models serve as the single source of truth for event-parsing logic,
completely decoupled from any HTTP client or network concerns.
"""
from __future__ import annotations

from datetime import datetime
from typing import Optional, Union

from app.domain.models import QuestionType, MCQCandidateAnswer, TextCandidateAnswer
from pydantic import BaseModel, Field


# ---------------------------------------------------------------------------
# Evaluation criteria
# ---------------------------------------------------------------------------

class EvaluationCriteria(BaseModel):
    """Free-form evaluation criteria attached to short-answer questions."""
    keywords: list[str] = Field(default_factory=list)
    min_length: Optional[int] = None

    class Config:
        extra = "allow"  # future-proof for additional criteria fields


# ---------------------------------------------------------------------------
# Single exam item (question + candidate answer)
# ---------------------------------------------------------------------------

class GradingOption(BaseModel):
    """Option payload representation for multiple-choice/true-false questions."""
    id: str
    content: str


class GradingItem(BaseModel):
    """One question/answer pair inside the grading payload."""
    question_id: str
    session_question_id: str
    question_type: QuestionType
    content: str
    title: str
    topic: str
    media_url: Optional[str] = None

    # Scoring
    points: float
    negative_points: float = 0.0

    # True evaluation criteria (from Exam Service)
    expected_answer: Optional[str] = None
    evaluation_criteria: Optional[EvaluationCriteria] = None
    correct_option_ids: Optional[list[str]] = None
    options: list[GradingOption] = Field(default_factory=list, description="All options available for this question.")

    # Candidate's actual answer (from Candidate Service)
    has_answer: bool = False
    candidate_answer: Optional[Union[MCQCandidateAnswer, TextCandidateAnswer]] = None


# ---------------------------------------------------------------------------
# Slim trigger event envelope (v3.0)
# ---------------------------------------------------------------------------

class ExamReadyForGradingEvent(BaseModel):
    """
    Slim Kafka trigger event for ``exam.session.ready_for_grading`` (v3.0).

    Carries only identifiers and session metadata.  The grading-service fetches
    the full GradingPayload from the candidate-service internal HTTP endpoint.

    Only version "3.0" is accepted; older events are rejected.
    """
    event_id: str
    event_type: str
    version: str
    timestamp: datetime
    trace_id: Optional[str] = None

    # Context identifiers
    enterprise_id: str
    exam_id: str
    session_id: str
    candidate_id: str
    enrollment_id: str

    # Metadata
    status: str
    started_at: datetime
    submitted_at: Optional[datetime] = None
    terminated_at: Optional[datetime] = None
    auto_submitted: bool = False
    termination_reason: Optional[str] = None


# ---------------------------------------------------------------------------
# Full grading payload (returned by candidate-service internal endpoint)
# ---------------------------------------------------------------------------

class GradingPayload(BaseModel):
    """
    Full grading data returned by ``GET /internal/sessions/{id}/grading-payload``
    on the candidate-service.

    This is what the grading engine consumes instead of the (now slim) Kafka event.
    """
    session_id: str
    enterprise_id: str
    exam_id: str
    candidate_id: str
    enrollment_id: str
    candidate_name: str = "Test Candidate"
    candidate_email: str = "candidate@test.com"
    exam_title: str = "Test Exam"

    status: str
    started_at: datetime
    submitted_at: Optional[datetime] = None
    terminated_at: Optional[datetime] = None
    auto_submitted: bool = False
    termination_reason: Optional[str] = None

    items: list[GradingItem] = Field(default_factory=list)
