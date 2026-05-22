from pydantic import BaseModel, Field
from uuid import UUID
from datetime import datetime
from typing import List, Optional, Any


class QuestionGradeResponse(BaseModel):
    question_id: UUID
    session_question_id: UUID
    question_type: str
    title: str
    max_points: float
    awarded_points: float
    status: str


class GradeResultResponse(BaseModel):
    id: UUID
    session_id: UUID
    exam_id: UUID
    candidate_id: UUID
    enrollment_id: UUID
    total_max_points: float
    total_awarded_points: float
    percentage: float
    status: str
    graded_by: str
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
    enrollment_id: UUID
    total_max_points: float
    total_awarded_points: float
    percentage: float
    status: str
    graded_by: str
    is_tampered: bool
    version: int
    created_at: datetime
    updated_at: datetime
    question_results: List[QuestionGradeResponse]


class ManualOverrideRequest(BaseModel):
    new_score: float = Field(..., ge=0.0, description="The new total awarded score.")
    reason: str = Field(..., min_length=5, max_length=500, description="Reason for updating the student's score.")


class ManualOverrideResponse(BaseModel):
    session_id: UUID
    previous_score: float
    new_score: float
    new_percentage: float
    status: str
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
