"""
Face verification handler — POST /face/verify

Compares current webcam frame against the face reference registered at session start.
"""
from fastapi import APIRouter, Request, HTTPException

from app.domain.models import FaceVerifyRequest, FaceVerifyResponse
from app.domain.errors import FaceNotRegisteredError, NoClearFaceError, InternalServiceError
from app.middleware.context import get_candidate_id, get_enterprise_id

router = APIRouter()


@router.post("/face/verify", response_model=FaceVerifyResponse)
async def verify_face(body: FaceVerifyRequest, request: Request):
    """
    Periodic identity verification for candidates.

    Fetches the face reference from candidate-service, runs DeepFace comparison,
    logs any anomalies (face_not_detected, multiple_faces, identity_mismatch),
    and publishes proctoring.identity.verified to Kafka.
    """
    candidate_id = get_candidate_id(request)
    enterprise_id = get_enterprise_id(request)
    face_uc = request.app.state.face_uc

    try:
        return await face_uc.verify_face(
            session_id=body.session_id,
            candidate_id=candidate_id,
            enterprise_id=enterprise_id,
            req=body,
        )
    except FaceNotRegisteredError as exc:
        raise HTTPException(status_code=404, detail=str(exc))
    except NoClearFaceError as exc:
        raise HTTPException(status_code=422, detail=str(exc))
    except InternalServiceError as exc:
        raise HTTPException(status_code=502, detail=str(exc))
    except Exception as exc:
        raise HTTPException(status_code=500, detail="Face verification failed") from exc
