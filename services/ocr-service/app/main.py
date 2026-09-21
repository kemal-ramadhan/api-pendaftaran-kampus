from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

from app.config import settings
from app.middleware import RequestIDMiddleware, InternalAuthMiddleware
from app.api.health import router as health_router
from app.api.ocr import router as ocr_router
from app.response import error_response


def create_app() -> FastAPI:
    app = FastAPI(
        title="PMB OCR Service",
        description="Layanan Optical Character Recognition KTP & Peningkatan Kualitas Gambar",
        version="1.0.0",
        docs_url="/docs" if settings.app_env == "development" else None,
        redoc_url=None
    )

    # Pasang Middlewares (urutan: RequestID dulu baru InternalAuth)
    app.add_middleware(InternalAuthMiddleware)
    app.add_middleware(RequestIDMiddleware)

    # Daftarkan Router
    app.include_router(health_router)
    app.include_router(ocr_router)

    # Global Exception Handler
    @app.exception_handler(Exception)
    async def global_exception_handler(request: Request, exc: Exception):
        return error_response(
            message=f"Terjadi kesalahan internal pada layanan OCR: {str(exc)}",
            error_code="INTERNAL_SERVER_ERROR",
            status_code=500
        )

    return app


app = create_app()
