"""
Context middleware — extracts identity headers forwarded by the API Gateway.

The gateway sets X-Subject-Id (candidate UUID) and X-Enterprise-Id on every
authenticated request. These are added to request.state for use in handlers.
"""
from fastapi import Request, HTTPException
from starlette.middleware.base import BaseHTTPMiddleware
from uuid import UUID


class IdentityMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next):
        subject_id = request.headers.get("X-Subject-Id")
        enterprise_id = request.headers.get("X-Enterprise-Id")

        request.state.candidate_id = None
        request.state.enterprise_id = None

        if subject_id:
            try:
                request.state.candidate_id = UUID(subject_id)
            except ValueError:
                pass

        if enterprise_id:
            try:
                request.state.enterprise_id = UUID(enterprise_id)
            except ValueError:
                pass

        return await call_next(request)


def get_candidate_id(request: Request) -> UUID:
    cid = getattr(request.state, "candidate_id", None)
    if not cid:
        raise HTTPException(status_code=401, detail="Candidate identity missing")
    return cid


def get_enterprise_id(request: Request) -> UUID:
    eid = getattr(request.state, "enterprise_id", None)
    if not eid:
        raise HTTPException(status_code=401, detail="Enterprise identity missing")
    return eid
