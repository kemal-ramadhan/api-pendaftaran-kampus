import os
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    app_env: str = "development"
    port: int = 8088
    internal_token: str = "super-secret-internal-token-change-in-production"
    redis_addr: str = "redis:6379"

    ocr_engine: str = "tesseract"
    ocr_quality_threshold: float = 70.0
    ocr_field_confidence_threshold: float = 0.85
    ocr_result_ttl: str = "15m"
    ocr_max_file_mb: int = 5
    ocr_cloud_enabled: bool = False
    worker_concurrency: int = 2

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore"
    )

    @property
    def ttl_seconds(self) -> int:
        ttl = self.ocr_result_ttl.lower().strip()
        if ttl.endswith("m"):
            return int(ttl[:-1]) * 60
        elif ttl.endswith("s"):
            return int(ttl[:-1])
        elif ttl.endswith("h"):
            return int(ttl[:-1]) * 3600
        return int(ttl)


settings = Settings()
