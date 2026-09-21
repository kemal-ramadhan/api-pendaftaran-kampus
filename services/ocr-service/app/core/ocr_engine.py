import io
from abc import ABC, abstractmethod
from typing import Tuple, Optional, Dict
import cv2
import numpy as np
from PIL import Image
import pytesseract
import pypdfium2

from app.config import settings
from app.models.schemas import OcrField, JobStatusResponse
from app.core.image_quality import assess_image_quality
from app.core.image_enhancer import detect_and_crop_ktp, enhance_ktp_image
from app.core.ktp_parser import parse_ktp_text


class OcrEngine(ABC):
    @abstractmethod
    def extract_text(self, image: np.ndarray) -> Tuple[str, float]:
        """Ekstraksi teks dari gambar matriks OpenCV dan kembalikan (teks, confidence 0-1)"""
        pass


class TesseractEngine(OcrEngine):
    def __init__(self):
        # Cek apakah bahasa ind tersedia di tesseract
        try:
            langs = pytesseract.get_languages()
            if "ind" in langs:
                self.lang = "ind+eng"
            else:
                self.lang = "eng"
        except Exception:
            self.lang = "eng"

    def extract_text(self, image: np.ndarray) -> Tuple[str, float]:
        # Konversi BGR ke RGB untuk Tesseract
        if len(image.shape) == 3:
            rgb = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)
        else:
            rgb = image

        config = "--oem 3 --psm 6"
        data = pytesseract.image_to_data(rgb, lang=self.lang, config=config, output_type=pytesseract.Output.DICT)

        confidences = []
        words = []
        for i, conf in enumerate(data.get("conf", [])):
            text = data.get("text", [])[i].strip()
            if text:
                words.append(text)
                try:
                    c = float(conf)
                    if c >= 0:
                        confidences.append(c)
                except (ValueError, TypeError):
                    pass

        avg_conf = (sum(confidences) / len(confidences)) / 100.0 if confidences else 0.8
        full_text = pytesseract.image_to_string(rgb, lang=self.lang, config=config)
        return full_text, round(max(0.0, min(1.0, avg_conf)), 2)


class CloudVisionEngine(OcrEngine):
    def __init__(self):
        if not settings.ocr_cloud_enabled:
            raise RuntimeError("Cloud OCR dinonaktifkan sesuai kebijakan privasi lokal (UU PDP No. 27/2022).")

    def extract_text(self, image: np.ndarray) -> Tuple[str, float]:
        raise NotImplementedError("Cloud engine disabled.")


def get_ocr_engine() -> OcrEngine:
    if settings.ocr_engine == "cloud":
        return CloudVisionEngine()
    return TesseractEngine()


def decode_image_bytes(file_bytes: bytes, filename: str = "") -> np.ndarray:
    """Membaca file byte JPG/PNG atau PDF (halaman 1) menjadi array citra OpenCV BGR"""
    # Deteksi jika PDF
    if filename.lower().endswith(".pdf") or file_bytes.startswith(b"%PDF"):
        pdf = pypdfium2.PdfDocument(file_bytes)
        if len(pdf) == 0:
            raise ValueError("File PDF kosong")
        page = pdf[0]
        pil_image = page.render(scale=2.0).to_pil()
        rgb_img = np.array(pil_image)
        return cv2.cvtColor(rgb_img, cv2.COLOR_RGB2BGR)

    # Deteksi image biasa
    nparr = np.frombuffer(file_bytes, np.uint8)
    img = cv2.imdecode(nparr, cv2.IMREAD_COLOR)
    if img is None:
        raise ValueError("Gagal membaca format gambar. Pastikan file adalah JPG, PNG, atau PDF yang valid.")
    return img


def process_ktp_pipeline(file_bytes: bytes, filename: str = "", engine: Optional[OcrEngine] = None) -> JobStatusResponse:
    """
    Eksekusi pipeline lengkap OCR KTP:
    1. Decode gambar
    2. Deteksi & crop kontur KTP
    3. Cek kualitas awal
    4. Evaluasi apakah butuh foto ulang (retake_required)
    5. Peningkatan kualitas non-generatif jika skor < threshold
    6. Ekstraksi teks dengan OcrEngine
    7. Parsing & validasi field KTP
    """
    if engine is None:
        engine = get_ocr_engine()

    try:
        raw_img = decode_image_bytes(file_bytes, filename)
    except Exception as e:
        return JobStatusResponse(
            job_id="",
            status="retake_required",
            quality_score=0.0,
            enhanced=False,
            reason="CARD_NOT_FOUND",
            rejection_message=str(e),
            fields=None
        )

    # Deteksi dan crop kartu (deskewing)
    cropped_img = detect_and_crop_ktp(raw_img)

    # Cek kualitas citra
    initial_quality = assess_image_quality(cropped_img)

    # Jika kualitas citra terlalu parah untuk diperbaiki
    if not initial_quality.is_acceptable:
        return JobStatusResponse(
            job_id="",
            status="retake_required",
            quality_score=initial_quality.overall_score,
            enhanced=False,
            reason=initial_quality.rejection_reason,
            reason_message=initial_quality.rejection_message,
            fields=None
        )

    enhanced = False
    working_img = cropped_img
    final_quality_score = initial_quality.overall_score

    # Peningkatan kualitas non-generatif jika skor di bawah ambang batas (default 70)
    if initial_quality.overall_score < settings.ocr_quality_threshold:
        working_img = enhance_ktp_image(cropped_img)
        enhanced_quality = assess_image_quality(working_img)
        final_quality_score = max(initial_quality.overall_score, enhanced_quality.overall_score)
        enhanced = True

    # OCR extraction
    try:
        extracted_text, conf = engine.extract_text(working_img)
    except Exception as e:
        return JobStatusResponse(
            job_id="",
            status="failed",
            quality_score=final_quality_score,
            enhanced=enhanced,
            reason="ENGINE_ERROR",
            reason_message=f"Terjadi kesalahan pada mesin OCR: {str(e)}",
            fields=None
        )

    # Validasi apakah teks mencerminkan dokumen KTP
    if not extracted_text or len(extracted_text.strip()) < 10:
        return JobStatusResponse(
            job_id="",
            status="retake_required",
            quality_score=final_quality_score,
            enhanced=enhanced,
            reason="NOT_KTP",
            reason_message="Dokumen yang diunggah tidak terdeteksi sebagai KTP atau teks tidak terbaca",
            fields=None
        )

    # Parsing field
    fields = parse_ktp_text(extracted_text, default_confidence=conf)

    # Jika NIK dan Nama keduanya kosong, anggap bukan KTP yang dapat diidentifikasi
    if not fields.get("nik") or not fields["nik"].value:
        if not fields.get("nama") or not fields["nama"].value:
            return JobStatusResponse(
                job_id="",
                status="retake_required",
                quality_score=final_quality_score,
                enhanced=enhanced,
                reason="NOT_KTP",
                reason_message="Informasi NIK atau Nama KTP tidak terdeteksi dalam foto",
                fields=None
            )

    return JobStatusResponse(
        job_id="",
        status="completed",
        quality_score=final_quality_score,
        enhanced=enhanced,
        reason=None,
        reason_message=None,
        fields=fields
    )
