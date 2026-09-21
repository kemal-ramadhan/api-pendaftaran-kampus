import json
import time
import uuid
from typing import Optional, Dict
import redis
from app.config import settings
from app.models.schemas import JobStatusResponse, OcrField


class JobStore:
    def __init__(self):
        self._redis_client = None
        self._memory_store: Dict[str, dict] = {}
        self._connect_redis()

    def _connect_redis(self):
        try:
            host, port = "127.0.0.1", 6379
            if ":" in settings.redis_addr:
                parts = settings.redis_addr.split(":")
                host = parts[0]
                port = int(parts[1])
            self._redis_client = redis.Redis(
                host=host,
                port=port,
                db=0,
                decode_responses=True,
                socket_timeout=2.0
            )
            self._redis_client.ping()
        except Exception:
            self._redis_client = None

    def is_redis_connected(self) -> bool:
        if self._redis_client is None:
            self._connect_redis()
        if self._redis_client is None:
            return False
        try:
            return bool(self._redis_client.ping())
        except Exception:
            return False

    def create_job(self) -> str:
        job_id = str(uuid.uuid4())
        initial_data = {
            "job_id": job_id,
            "status": "processing",
            "quality_score": 0.0,
            "enhanced": False,
            "reason": None,
            "reason_message": None,
            "fields": None,
            "created_at": time.time(),
            "completed_at": None
        }
        self.save_job(job_id, initial_data)
        return job_id

    def save_job(self, job_id: str, data: dict):
        ttl = settings.ttl_seconds
        payload = json.dumps(data)

        if self.is_redis_connected():
            try:
                self._redis_client.setex(f"ocr:job:{job_id}", ttl, payload)
                return
            except Exception:
                pass

        # In-memory fallback
        self._memory_store[job_id] = {
            "data": data,
            "expires_at": time.time() + ttl
        }

    def get_job(self, job_id: str) -> Optional[JobStatusResponse]:
        raw_json = None
        if self.is_redis_connected():
            try:
                raw_json = self._redis_client.get(f"ocr:job:{job_id}")
            except Exception:
                pass

        if raw_json is None:
            entry = self._memory_store.get(job_id)
            if entry:
                if time.time() > entry["expires_at"]:
                    del self._memory_store[job_id]
                    return None
                data = entry["data"]
            else:
                return None
        else:
            try:
                data = json.loads(raw_json)
            except Exception:
                return None

        fields_dict = None
        if data.get("fields"):
            fields_dict = {
                k: OcrField(**v) if isinstance(v, dict) else v
                for k, v in data["fields"].items()
            }

        return JobStatusResponse(
            job_id=data["job_id"],
            status=data["status"],
            quality_score=data.get("quality_score", 0.0),
            enhanced=data.get("enhanced", False),
            reason=data.get("reason"),
            reason_message=data.get("reason_message"),
            fields=fields_dict,
            created_at=data.get("created_at"),
            completed_at=data.get("completed_at")
        )


job_store = JobStore()
