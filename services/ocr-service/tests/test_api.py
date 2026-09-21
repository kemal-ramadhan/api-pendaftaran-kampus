import pytest
from httpx import AsyncClient, ASGITransport
from app.main import app
from app.config import settings
from tests.synthetic_ktp_generator import create_synthetic_ktp_image, SAMPLE_DATASETS
import io


@pytest.mark.asyncio
async def test_health_endpoints():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as ac:
        res = await ac.get("/health")
        assert res.status_code == 200
        data = res.json()
        assert data["success"] is True
        assert data["data"]["status"] == "ok"

        res_ready = await ac.get("/health/ready")
        assert res_ready.status_code == 200
        assert res_ready.json()["success"] is True


@pytest.mark.asyncio
async def test_internal_auth_protection():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as ac:
        # Panggilan tanpa token
        res = await ac.get("/internal/ocr/jobs/dummy-job")
        assert res.status_code == 401
        assert res.json()["error_code"] == "UNAUTHORIZED"

        # Panggilan dengan token salah
        res = await ac.get(
            "/internal/ocr/jobs/dummy-job",
            headers={"X-Internal-Token": "wrong-token"}
        )
        assert res.status_code == 401


@pytest.mark.asyncio
async def test_ktp_ocr_job_flow():
    transport = ASGITransport(app=app)
    headers = {
        "X-Internal-Token": settings.internal_token,
        "X-Request-ID": "test-req-12345"
    }

    # Buat KTP sintetis in-memory
    pil_img = create_synthetic_ktp_image(SAMPLE_DATASETS[0])
    buf = io.BytesIO()
    pil_img.save(buf, format="JPEG", quality=95)
    img_bytes = buf.getvalue()

    async with AsyncClient(transport=transport, base_url="http://test") as ac:
        # Submit OCR Job
        files = {"file": ("ktp_sample.jpg", img_bytes, "image/jpeg")}
        res = await ac.post("/internal/ocr/ktp", headers=headers, files=files)
        assert res.status_code == 202
        resp_data = res.json()
        assert resp_data["success"] is True
        job_id = resp_data["data"]["job_id"]
        assert job_id is not None

        # Ambil status job
        res_job = await ac.get(f"/internal/ocr/jobs/{job_id}", headers=headers)
        assert res_job.status_code == 200
        job_info = res_job.json()["data"]
        assert job_info["job_id"] == job_id
