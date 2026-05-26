import logging
from uuid import UUID
from typing import Optional, List
from fastapi import APIRouter, Request, Query, HTTPException, Depends

from app.domain.models import (
    PaginatedGradeResults,
    GradeDetailResponse,
    ManualOverrideRequest,
    ManualOverrideResponse,
    QuestionManualOverrideRequest,
    QuestionManualOverrideResponse,
    AuditLogResponse
)
from app.middleware.context import get_enterprise_id, get_user_id, get_user_role
from app.repository.grading_repository import DataTamperingError

router = APIRouter(prefix="/grading/results", tags=["grading"])
logger = logging.getLogger("grading.handler")


@router.get("", response_model=PaginatedGradeResults)
async def list_graded_students(
    request: Request,
    exam_id: Optional[UUID] = Query(None, description="Filter results by exam ID"),
    limit: int = Query(10, ge=1, le=100, description="Number of items to retrieve"),
    offset: int = Query(0, ge=0, description="Offset for pagination")
):
    """
    Get a paginated list of examinees and their overall grades.
    Access restricted to admins of the specific enterprise.
    """
    enterprise_id = get_enterprise_id(request)
    grading_uc = request.app.state.grading_uc

    results, total = await grading_uc.list_graded_students(
        enterprise_id=enterprise_id,
        exam_id=exam_id,
        limit=limit,
        offset=offset
    )

    return PaginatedGradeResults(
        results=results,
        total=total,
        limit=limit,
        offset=offset
    )


@router.get("/{session_id}", response_model=GradeDetailResponse)
async def get_grade_detail(session_id: UUID, request: Request):
    """
    Get the detailed grading breakdown for an exam session.
    Verifies data integrity and enforces multi-tenant boundaries.
    """
    enterprise_id = get_enterprise_id(request)
    grading_uc = request.app.state.grading_uc

    try:
        detail = await grading_uc.get_grade_detail(session_id)
    except DataTamperingError as exc:
        raise HTTPException(
            status_code=409,
            detail="DATABASE CORRUPTION DETECTED: Grade result row checksum mismatch."
        ) from exc

    if not detail:
        raise HTTPException(status_code=404, detail="Grading result not found")

    # Enforce tenant isolation
    if detail["enterprise_id"] != enterprise_id:
        raise HTTPException(status_code=403, detail="Access denied to this resource")

    return detail


@router.post("/{session_id}/override", response_model=ManualOverrideResponse)
async def override_grade(
    session_id: UUID,
    body: ManualOverrideRequest,
    request: Request
):
    """
    Manually override a student's final grade.
    Computes a new cryptographic checksum and writes to an append-only audit trail.
    """
    enterprise_id = get_enterprise_id(request)
    actor_id = get_user_id(request)
    actor_role = get_user_role(request)
    grading_uc = request.app.state.grading_uc

    # Get client IP address
    ip_address = request.client.host if request.client else None

    # First, fetch to verify ownership
    detail = await grading_uc.get_grade_detail(session_id)
    if not detail:
        raise HTTPException(status_code=404, detail="Grading result not found")

    if detail["enterprise_id"] != enterprise_id:
        raise HTTPException(status_code=403, detail="Access denied to this resource")

    try:
        update_result = await grading_uc.update_grade_manually(
            session_id=session_id,
            new_score=body.new_score,
            actor_id=actor_id,
            actor_role=actor_role,
            reason=body.reason,
            ip_address=ip_address
        )
    except DataTamperingError as exc:
        raise HTTPException(
            status_code=409,
            detail="Tamper detection block: cannot edit a database row that fails integrity verification."
        ) from exc
    except Exception as exc:
        logger.exception("Failed to manually update grade: %s", exc)
        raise HTTPException(status_code=500, detail=str(exc))

    return ManualOverrideResponse(
        session_id=update_result["session_id"],
        previous_score=update_result["previous_score"],
        new_score=update_result["new_score"],
        new_percentage=update_result["new_percentage"],
        status=update_result["status"]
    )


@router.post("/{session_id}/questions/{session_question_id}/override", response_model=QuestionManualOverrideResponse)
async def override_question_grade(
    session_id: UUID,
    session_question_id: UUID,
    body: QuestionManualOverrideRequest,
    request: Request
):
    """
    Manually override a specific question's score.
    Computes a new cryptographic checksum and writes to an append-only audit trail.
    """
    enterprise_id = get_enterprise_id(request)
    actor_id = get_user_id(request)
    actor_role = get_user_role(request)
    
    if actor_role not in ("enterprise_admin", "enterprise_staff", "system"):
         raise HTTPException(status_code=403, detail="Insufficient role to perform this action.")
         
    grading_uc = request.app.state.grading_uc

    # Get client IP address
    ip_address = request.client.host if request.client else None

    # First, fetch to verify ownership
    detail = await grading_uc.get_grade_detail(session_id)
    if not detail:
        raise HTTPException(status_code=404, detail="Grading result not found")

    if detail["enterprise_id"] != enterprise_id:
        raise HTTPException(status_code=403, detail="Access denied to this resource")

    try:
        update_result = await grading_uc.update_question_grade_manually(
            session_id=session_id,
            session_question_id=session_question_id,
            new_question_score=body.new_score,
            actor_id=actor_id,
            actor_role=actor_role,
            reason=body.reason,
            ip_address=ip_address
        )
    except DataTamperingError as exc:
        raise HTTPException(
            status_code=409,
            detail="Tamper detection block: cannot edit a database row that fails integrity verification."
        ) from exc
    except ValueError as exc:
        raise HTTPException(
            status_code=404,
            detail=str(exc)
        )
    except Exception as exc:
        logger.exception("Failed to manually update question grade: %s", exc)
        raise HTTPException(status_code=500, detail=str(exc))

    return QuestionManualOverrideResponse(
        session_id=update_result["session_id"],
        session_question_id=update_result["session_question_id"],
        previous_question_score=update_result["previous_question_score"],
        new_question_score=update_result["new_question_score"],
        previous_total_score=update_result["previous_total_score"],
        new_total_score=update_result["new_total_score"],
        new_percentage=update_result["new_percentage"],
        status=update_result["status"]
    )


@router.get("/{session_id}/logs", response_model=List[AuditLogResponse])
async def get_audit_logs(session_id: UUID, request: Request):
    """
    Get the immutable audit history / edit logs for an exam session's grade.
    """
    enterprise_id = get_enterprise_id(request)
    grading_uc = request.app.state.grading_uc

    # Verify ownership
    detail = await grading_uc.get_grade_detail(session_id)
    if not detail:
        raise HTTPException(status_code=404, detail="Grading result not found")

    if detail["enterprise_id"] != enterprise_id:
        raise HTTPException(status_code=403, detail="Access denied to this resource")

    logs = await grading_uc.get_audit_logs(session_id)
    return logs
