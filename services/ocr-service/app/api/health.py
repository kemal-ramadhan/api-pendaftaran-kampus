from fastapi import APIRouter
from app.response import success_response, error_response
from app.queue.job_queue import job_store

router = APIRouter(tags=["Health"])


@router.get("/health")
async def health_check():
    return success_response(
        data={"status": "ok", "service": "ocr-service"},
        message="Service is healthy"
    )


@router.get("/health/ready")
async def readiness_check():
    # Service tetap ready walau redis fallback in-memory, tetapi laporkan statusnya
    redis_ready = job_store.is_redis_connected()
    return success_response(
        data={
            "status": "ready",
            "service": "ocr-service",
            "redis_connected": redis_ready
        },
        message="Service is ready"
    )
