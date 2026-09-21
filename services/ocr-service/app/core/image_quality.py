import cv2
import numpy as np
from app.models.schemas import QualityAssessment


def assess_image_quality(image: np.ndarray, min_dimension: int = 640) -> QualityAssessment:
    """
    Menghitung skor kualitas gambar (0-100) berdasarkan:
    1. Ketajaman (Laplacian Variance)
    2. Kecerahan (Mean Luminance)
    3. Kontras (Std Dev Luminance)
    4. Silau/Pantulan Cahaya (Glare Pixel Ratio)
    5. Resolusi (Dimensi gambar)
    """
    h, w = image.shape[:2]
    if len(image.shape) == 3:
        gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
    else:
        gray = image.copy()

    # 1. Resolusi
    min_side = min(h, w)
    if min_side < 400:
        res_score = max(10.0, (min_side / 640.0) * 60.0)
    elif min_side < 640:
        res_score = 60.0 + ((min_side - 400) / 240.0) * 30.0
    else:
        res_score = 100.0

    # 2. Ketajaman (Laplacian Variance)
    lap_var = cv2.Laplacian(gray, cv2.CV_64F).var()
    # Variansi < 30 biasanya sangat buram
    # Variansi > 120 cukup tajam
    if lap_var < 30:
        sharpness_score = max(5.0, (lap_var / 30.0) * 40.0)
    elif lap_var < 100:
        sharpness_score = 40.0 + ((lap_var - 30.0) / 70.0) * 40.0
    else:
        sharpness_score = min(100.0, 80.0 + ((lap_var - 100.0) / 100.0) * 20.0)

    # 3. Kecerahan
    mean_val = float(np.mean(gray))
    if mean_val < 50:
        brightness_score = max(10.0, (mean_val / 50.0) * 45.0)
    elif mean_val > 220:
        brightness_score = max(20.0, 100.0 - ((mean_val - 220) / 35.0) * 60.0)
    else:
        # Ideal di rentang 90 - 160
        dist = abs(mean_val - 125.0)
        brightness_score = max(50.0, 100.0 - (dist / 95.0) * 50.0)

    # 4. Kontras
    std_val = float(np.std(gray))
    if std_val < 25:
        contrast_score = max(10.0, (std_val / 25.0) * 50.0)
    elif std_val < 50:
        contrast_score = 50.0 + ((std_val - 25.0) / 25.0) * 35.0
    else:
        contrast_score = min(100.0, 85.0 + ((std_val - 50.0) / 40.0) * 15.0)

    # 5. Silau (Glare)
    glare_ratio = float(np.sum(gray > 248)) / float(gray.size)
    if glare_ratio > 0.15:
        glare_score = max(10.0, 100.0 - (glare_ratio / 0.3) * 90.0)
    elif glare_ratio > 0.05:
        glare_score = 60.0 - ((glare_ratio - 0.05) / 0.10) * 30.0
    else:
        glare_score = 100.0 - (glare_ratio / 0.05) * 15.0

    # Hitung Skor Keseluruhan Terbobot
    overall = (
        sharpness_score * 0.35 +
        brightness_score * 0.20 +
        contrast_score * 0.20 +
        glare_score * 0.15 +
        res_score * 0.10
    )
    overall = round(max(0.0, min(100.0, overall)), 1)

    # Cek alasan penolakan jika kualitas sangat parah sehingga tidak bisa diperbaiki
    is_acceptable = True
    rejection_reason = None
    rejection_message = None

    if min_side < 400:
        is_acceptable = False
        rejection_reason = "LOW_RESOLUTION"
        rejection_message = "Resolusi foto terlalu rendah, gunakan kamera lebih jelas (minimal 640 px)"
    elif mean_val < 35.0:  # Hampir hitam pekat
        is_acceptable = False
        rejection_reason = "TOO_DARK"
        rejection_message = "Foto terlalu gelap, gunakan pencahayaan yang cukup"
    elif glare_ratio > 0.25:  # Sebagian besar kartu tertutup putih silau
        is_acceptable = False
        rejection_reason = "GLARE"
        rejection_message = "Ada pantulan cahaya berlebih, hindari lampu langsung atau flash"
    elif lap_var < 15.0:  # Ekstrem blur yang tidak mungkin terbaca
        is_acceptable = False
        rejection_reason = "BLURRY"
        rejection_message = "Foto buram, pegang HP lebih stabil dan pastikan fokus"

    return QualityAssessment(
        sharpness_score=round(sharpness_score, 1),
        brightness_score=round(brightness_score, 1),
        contrast_score=round(contrast_score, 1),
        glare_score=round(glare_score, 1),
        resolution_score=round(res_score, 1),
        overall_score=overall,
        is_acceptable=is_acceptable,
        rejection_reason=rejection_reason,
        rejection_message=rejection_message
    )
