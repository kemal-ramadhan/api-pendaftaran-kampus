import uuid
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from starlette.responses import Response
from app.config import settings
from app.response import error_response


class RequestIDMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next) -> Response:
        req_id = request.headers.get("X-Request-ID")
        if not req_id:
            req_id = str(uuid.uuid4())
        request.state.request_id = req_id

        response = await call_next(request)
        response.headers["X-Request-ID"] = req_id
        return response


class InternalAuthMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next) -> Response:
        path = request.url.path

        # Hanya lindungi rute internal
        if path.startswith("/internal/"):
            token = request.headers.get("X-Internal-Token")
            if not token or token != settings.internal_token:
                return error_response(
                    message="Akses internal tidak sah",
                    error_code="UNAUTHORIZED",
                    status_code=401
                )

        return await call_next(request)
