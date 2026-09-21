from typing import Optional, Dict
from pydantic import BaseModel, Field


class OcrField(BaseModel):
    value: str
    confidence: float = Field(default=1.0, ge=0.0, le=1.0)
    needs_review: bool = False


class QualityAssessment(BaseModel):
    sharpness_score: float = Field(default=0.0, ge=0.0, le=100.0)
    brightness_score: float = Field(default=0.0, ge=0.0, le=100.0)
    contrast_score: float = Field(default=0.0, ge=0.0, le=100.0)
    glare_score: float = Field(default=0.0, ge=0.0, le=100.0)
    resolution_score: float = Field(default=0.0, ge=0.0, le=100.0)
    overall_score: float = Field(default=0.0, ge=0.0, le=100.0)
    is_acceptable: bool = True
    rejection_reason: Optional[str] = None
    rejection_message: Optional[str] = None


class JobStatusResponse(BaseModel):
    job_id: str
    status: str  # processing, completed, retake_required, failed
    quality_score: float = 0.0
    enhanced: bool = False
    reason: Optional[str] = None
    reason_message: Optional[str] = None
    fields: Optional[Dict[str, OcrField]] = None
    created_at: Optional[float] = None
    completed_at: Optional[float] = None
