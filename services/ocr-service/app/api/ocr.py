import asyncio
import time
from fastapi import APIRouter, UploadFile, File, BackgroundTasks, Request
from app.config import settings
from app.response import success_response, error_response
from app.queue.job_queue import job_store
from app.core.ocr_engine import process_ktp_pipeline, get_ocr_engine

router = APIRouter(prefix="/internal/ocr", tags=["Internal OCR"])

ALLOWED_EXTENSIONS = {".jpg", ".jpeg", ".png", ".pdf"}
MAX_FILE_BYTES = settings.ocr_max_file_mb * 1024 * 1024


def run_ocr_task(job_id: str, file_bytes: bytes, filename: str):
    """Background worker untuk memproses gambar KTP"""
    engine = get_ocr_engine()
    result = process_ktp_pipeline(file_bytes, filename=filename, engine=engine)

    fields_json = None
    if result.fields:
        fields_json = {k: v.model_dump() for k, v in result.fields.items()}

    data = {
        "job_id": job_id,
        "status": result.status,
        "quality_score": result.quality_score,
        "enhanced": result.enhanced,
        "reason": result.reason,
        "reason_message": result.reason_message,
        "fields": fields_json,
        "completed_at": time.time()
    }
    job_store.save_job(job_id, data)


@router.post("/ktp")
async def create_ktp_ocr_job(
    request: Request,
    background_tasks: BackgroundTasks,
    file: UploadFile = File(...)
):
    # 1. Validasi ekstensi
    filename = file.filename or "upload.jpg"
    ext = "." + filename.split(".")[-1].lower() if "." in filename else ""
    if ext not in ALLOWED_EXTENSIONS:
        return error_response(
            message=f"Format file '{ext}' tidak didukung. Harap unggah file JPG, PNG, atau PDF.",
            error_code="FILE_TYPE_NOT_ALLOWED",
            status_code=422
        )

    # 2. Baca isi file dan validasi ukuran
    content = await file.read()
    if len(content) > MAX_FILE_BYTES:
        return error_response(
            message=f"Ukuran file melebihi batas maksimal {settings.ocr_max_file_mb} MB.",
            error_code="FILE_TOO_LARGE",
            status_code=422
        )

    if len(content) < 100:
        return error_response(
            message="File gambar kosong atau korup.",
            error_code="INVALID_FILE",
            status_code=422
        )

    # 3. Buat Job
    job_id = job_store.create_job()

    # 4. Dispatch ke background task
    background_tasks.add_task(run_ocr_task, job_id, content, filename)

    return success_response(
        data={"job_id": job_id, "status": "processing"},
        message="Job OCR berhasil dibuat dan sedang diproses",
        status_code=202
    )


@router.get("/jobs/{job_id}")
async def get_ocr_job_status(job_id: str):
    job = job_store.get_job(job_id)
    if not job:
        return error_response(
            message="Job OCR tidak ditemukan atau telah kedaluwarsa.",
            error_code="NOT_FOUND",
            status_code=404
        )

    return success_response(
        data=job.model_dump(),
        message="Status job OCR berhasil diambil"
    )
