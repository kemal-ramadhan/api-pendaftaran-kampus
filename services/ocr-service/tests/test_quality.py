import numpy as np
import cv2
import pytest
from app.core.image_quality import assess_image_quality
from app.core.image_enhancer import enhance_ktp_image


def test_quality_sharp_image():
    # Buat citra sintetis beresolusi 800x600 dengan teks tajam
    img = np.full((600, 800, 3), 200, dtype=np.uint8)
    cv2.putText(img, "PROVINSI JAWA BARAT", (100, 100), cv2.FONT_HERSHEY_SIMPLEX, 1.2, (0, 0, 0), 3)
    cv2.putText(img, "NIK: 3273011508050001", (100, 200), cv2.FONT_HERSHEY_SIMPLEX, 1.0, (0, 0, 0), 2)

    qa = assess_image_quality(img)
    assert qa.is_acceptable is True
    assert qa.overall_score >= 70.0
    assert qa.rejection_reason is None


def test_quality_blurry_rejection():
    # Buat citra dengan blur ekstrem
    img = np.full((600, 800, 3), 180, dtype=np.uint8)
    cv2.putText(img, "NIK: 3273011508050001", (100, 200), cv2.FONT_HERSHEY_SIMPLEX, 1.0, (0, 0, 0), 2)
    blurry = cv2.GaussianBlur(img, (51, 51), 0)

    qa = assess_image_quality(blurry)
    assert qa.is_acceptable is False
    assert qa.rejection_reason == "BLURRY"


def test_quality_too_dark_rejection():
    # Citra hampir hitam pekat
    img = np.full((600, 800, 3), 15, dtype=np.uint8)
    qa = assess_image_quality(img)
    assert qa.is_acceptable is False
    assert qa.rejection_reason == "TOO_DARK"


def test_quality_glare_detection():
    # Citra dengan hotspot silau putih pekat > 30% area
    img = np.full((600, 800, 3), 140, dtype=np.uint8)
    img[100:500, 100:600] = 255  # Kotak silau putih pekat

    qa = assess_image_quality(img)
    assert qa.glare_score < 40.0


def test_quality_low_resolution_rejection():
    # Dimensi sisi terpendek 300 px (< 400 px)
    img = np.full((300, 450, 3), 150, dtype=np.uint8)
    qa = assess_image_quality(img)
    assert qa.is_acceptable is False
    assert qa.rejection_reason == "LOW_RESOLUTION"


def test_enhance_ktp_image():
    # Citra agak redup dan kontras rendah
    img = np.full((500, 700, 3), 100, dtype=np.uint8)
    cv2.putText(img, "3273011508050001", (80, 200), cv2.FONT_HERSHEY_SIMPLEX, 1.0, (30, 30, 30), 2)

    enhanced = enhance_ktp_image(img)
    assert enhanced.shape[0] >= img.shape[0]
    assert enhanced.shape[1] >= img.shape[1]
    # Standard deviasi (kontras) citra hasil perbaikan harus lebih tinggi daripada aslinya
    assert np.std(enhanced) >= np.std(img)
