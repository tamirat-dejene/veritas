from pydantic import BaseModel, Field
from uuid import UUID
from datetime import datetime
from typing import List, Optional, Union
from enum import Enum


class GradingStatus(str, Enum):
    pending = "pending"
    graded = "graded"
    reviewed = "reviewed"
    disputed = "disputed"

class QuestionGradingStatus(str, Enum):
    correct = "correct"
    incorrect = "incorrect"
    partial = "partial"
    skipped = "skipped"
    ai_graded = "ai_graded"
    human_review = "human_review"

class QuestionType(str, Enum):
    MCQ = "MCQ"
    TrueFalse = "TrueFalse"
    ShortAnswer = "ShortAnswer"
    Essay = "Essay"

# ---------------------------------------------------------------------------
# Candidate answer sub-models
# (shared between domain responses and the internal grading payload models)
# ---------------------------------------------------------------------------

class MCQCandidateAnswer(BaseModel):
    """Answer payload for multiple-choice / true-false questions."""
    selectedOptionIds: List[str] = Field(default_factory=list, description="IDs of the options selected by the candidate.")


class TextCandidateAnswer(BaseModel):
    """Answer payload for short-answer and essay questions."""
    text: Optional[str] = Field(None, description="Free-text answer written by the candidate.")


class QuestionOption(BaseModel):
    """Option details for MCQ / TrueFalse questions."""
    id: str
    content: str


class QuestionGradeResponse(BaseModel):
    question_id: UUID
    session_question_id: UUID
    question_type: QuestionType
    title: str
    content: str
    candidate_answer: Optional[Union[MCQCandidateAnswer, TextCandidateAnswer]] = Field(
        None,
        description="The candidate's submitted answer. MCQ/TrueFalse questions use `MCQCandidateAnswer` (selectedOptionIds); ShortAnswer/Essay questions use `TextCandidateAnswer` (text)."
    )
    max_points: float
    awarded_points: float
    status: QuestionGradingStatus
    options: Optional[List[QuestionOption]] = Field(None, description="All options available for this question.")
    correct_option_ids: Optional[List[str]] = Field(None, description="IDs of the correct options.")


class GraderInfo(BaseModel):
    id: str
    type: str  # "system" or "human"


class CandidateInfo(BaseModel):
    """Enriched candidate profile details."""
    id: UUID
    first_name: str
    last_name: str
    email: Optional[str] = None


class GraderUserInfo(BaseModel):
    """Enriched human grader user profile details."""
    id: UUID
    first_name: str
    last_name: str
    email: str
    role: str


class GraderInfoExtended(BaseModel):
    """Extended grader info containing user profile details if type is human."""
    id: str
    type: str  # "system" or "human"
    user_details: Optional[GraderUserInfo] = None


class GradeResultResponse(BaseModel):
    id: UUID
    session_id: UUID
    exam_id: UUID
    candidate_id: UUID
    enrollment_id: UUID
    total_max_points: float
    total_awarded_points: float
    percentage: float
    graded_by: GraderInfo
    status: GradingStatus
    is_tampered: bool
    version: int
    created_at: datetime
    updated_at: datetime


class PaginatedGradeResults(BaseModel):
    results: List[GradeResultResponse]
    total: int
    limit: int
    offset: int


class GradeDetailResponse(BaseModel):
    id: UUID
    session_id: UUID
    exam_id: UUID
    candidate_id: UUID
    candidate_info: Optional[CandidateInfo] = None
    enrollment_id: UUID
    total_max_points: float
    total_awarded_points: float
    percentage: float 
    graded_by: GraderInfoExtended
    status: GradingStatus
    is_tampered: bool
    version: int
    created_at: datetime
    updated_at: datetime
    question_results: List[QuestionGradeResponse]


class ManualOverrideRequest(BaseModel):
    new_score: float = Field(..., ge=0.0, description="The new total awarded score.")
    reason: str = Field(..., min_length=5, max_length=500, description="Reason for updating the student's score.")


class QuestionManualOverrideRequest(BaseModel):
    new_score: float = Field(..., ge=0.0, description="The new awarded score for this question.")
    reason: str = Field(..., min_length=5, max_length=500, description="Reason for updating the student's question score.")


class QuestionManualOverrideResponse(BaseModel):
    session_id: UUID
    session_question_id: UUID
    previous_question_score: float
    new_question_score: float
    previous_total_score: float
    new_total_score: float
    new_percentage: float
    status: str
    message: str = "Question grade manually overridden successfully."


class ManualOverrideResponse(BaseModel):
    session_id: UUID
    previous_score: float
    new_score: float
    new_percentage: float
    status: GradingStatus
    message: str = "Grade manually overridden successfully."


class AuditLogResponse(BaseModel):
    id: UUID
    action: str
    actor_id: Optional[UUID] = None
    actor_role: str
    old_values: Optional[dict] = None
    new_values: dict
    changed_fields: Optional[List[str]] = None
    ip_address: Optional[str] = None
    reason: Optional[str] = None
    created_at: datetime


class GradingStatusResponse(BaseModel):
    """Lightweight response for the status-polling endpoint.

    Callers can poll ``GET /grading/results/{session_id}/status`` after
    exam submission.  The ``status`` field progresses:
      pending  → grading is still in progress
      graded   → automated grading finished
      reviewed → a human manually overrode the score
      disputed → the result is under dispute
    """
    session_id: UUID
    status: GradingStatus
    graded_by: str
    percentage: float
    updated_at: datetime
