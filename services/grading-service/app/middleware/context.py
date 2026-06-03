"""
Context middleware — extracts identity headers forwarded by the API Gateway.
"""
from uuid import UUID
from fastapi import Request, HTTPException
from starlette.middleware.base import BaseHTTPMiddleware


class IdentityMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next):
        user_id = request.headers.get("X-User-ID")
        subject_id = request.headers.get("X-Subject-Id")
        enterprise_id = request.headers.get("X-Enterprise-ID")
        user_role = request.headers.get("X-User-Role")

        request.state.user_id = None
        request.state.candidate_id = None
        request.state.enterprise_id = None
        request.state.user_role = "SystemAdmin"

        if user_id:
            try:
                request.state.user_id = UUID(user_id)
            except ValueError:
                pass

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

        if user_role:
            request.state.user_role = user_role

        return await call_next(request)


def get_user_id(request: Request) -> UUID:
    uid = getattr(request.state, "user_id", None)
    if not uid:
        raise HTTPException(status_code=401, detail="User identity missing")
    return uid


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


def get_user_role(request: Request) -> str:
    return getattr(request.state, "user_role", "SystemAdmin")
