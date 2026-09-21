from typing import Any, Optional
from fastapi.responses import JSONResponse


def success_response(
    data: Any = None,
    message: str = "Sukses",
    status_code: int = 200,
    meta: Optional[dict] = None
) -> JSONResponse:
    content = {
        "success": True,
        "message": message,
        "data": data,
    }
    if meta is not None:
        content["meta"] = meta
    return JSONResponse(status_code=status_code, content=content)


def error_response(
    message: str,
    error_code: str = "INTERNAL_SERVER_ERROR",
    status_code: int = 500,
    errors: Optional[Any] = None
) -> JSONResponse:
    content = {
        "success": False,
        "message": message,
        "error_code": error_code,
    }
    if errors is not None:
        content["errors"] = errors
    return JSONResponse(status_code=status_code, content=content)
