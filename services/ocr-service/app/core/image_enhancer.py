import cv2
import numpy as np


def order_points(pts: np.ndarray) -> np.ndarray:
    """
    Mengurutkan 4 titik koordinat: top-left, top-right, bottom-right, bottom-left
    """
    rect = np.zeros((4, 2), dtype="float32")
    s = pts.sum(axis=1)
    rect[0] = pts[np.argmin(s)]
    rect[2] = pts[np.argmax(s)]

    diff = np.diff(pts, axis=1)
    rect[1] = pts[np.argmin(diff)]
    rect[3] = pts[np.argmax(diff)]
    return rect


def four_point_transform(image: np.ndarray, pts: np.ndarray) -> np.ndarray:
    rect = order_points(pts)
    (tl, tr, br, bl) = rect

    # Hitung lebar maksimum baru
    widthA = np.sqrt(((br[0] - bl[0]) ** 2) + ((br[1] - bl[1]) ** 2))
    widthB = np.sqrt(((tr[0] - tl[0]) ** 2) + ((tr[1] - tl[1]) ** 2))
    maxWidth = max(int(widthA), int(widthB))

    # Hitung tinggi maksimum baru
    heightA = np.sqrt(((tr[0] - br[0]) ** 2) + ((tr[1] - br[1]) ** 2))
    heightB = np.sqrt(((tl[0] - bl[0]) ** 2) + ((tl[1] - bl[1]) ** 2))
    maxHeight = max(int(heightA), int(heightB))

    # Rasio aspek KTP ID-1 sekitar 1.58 : 1
    # Jika terbalik (vertikal), balikkan dimensi
    if maxHeight > maxWidth:
        maxWidth, maxHeight = maxHeight, maxWidth

    dst = np.array([
        [0, 0],
        [maxWidth - 1, 0],
        [maxWidth - 1, maxHeight - 1],
        [0, maxHeight - 1]
    ], dtype="float32")

    M = cv2.getPerspectiveTransform(rect, dst)
    warped = cv2.warpPerspective(image, M, (maxWidth, maxHeight))
    return warped


def detect_and_crop_ktp(image: np.ndarray) -> np.ndarray:
    """
    Deteksi kontur KTP, luruskan perspektif dan kemiringan (deskew).
    Jika kartu sudah di-crop penuh atau tidak ditemukan kontur 4 titik, kembalikan gambar asli.
    """
    h, w = image.shape[:2]
    gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY) if len(image.shape) == 3 else image

    # Blur ringan untuk mereduksi noise kontur
    blurred = cv2.GaussianBlur(gray, (5, 5), 0)
    edged = cv2.Canny(blurred, 50, 150)

    contours, _ = cv2.findContours(edged, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)
    contours = sorted(contours, key=cv2.contourArea, reverse=True)[:5]

    for c in contours:
        peri = cv2.arcLength(c, True)
        approx = cv2.approxPolyDP(c, 0.02 * peri, True)

        # Jika menemukan poligon 4 titik dan luasnya cukup signifikan (> 25% area)
        if len(approx) == 4 and cv2.contourArea(c) > (h * w * 0.25):
            pts = approx.reshape(4, 2)
            try:
                warped = four_point_transform(image, pts)
                return warped
            except Exception:
                pass

    # Fallback deskew rotasi kecil (kemiringan sudut garis horizontal)
    coords = np.column_stack(np.where(gray > 0))
    angle = cv2.minAreaRect(coords)[-1]
    if angle < -45:
        angle = -(90 + angle)
    elif angle > 45:
        angle = 90 - angle
    else:
        angle = -angle

    if abs(angle) > 1.0 and abs(angle) < 45.0:
        (h_img, w_img) = image.shape[:2]
        center = (w_img // 2, h_img // 2)
        M = cv2.getRotationMatrix2D(center, angle, 1.0)
        rotated = cv2.warpAffine(
            image, M, (w_img, h_img),
            flags=cv2.INTER_CUBIC,
            borderMode=cv2.BORDER_REPLICATE
        )
        return rotated

    return image


def enhance_ktp_image(image: np.ndarray) -> np.ndarray:
    """
    Peningkatan kualitas gambar secara NON-GENERATIF:
    1. Non-generative Super-resolution (Lanczos4 upscale jika dimensi kecil)
    2. Bilateral filtering / Denoising yang mempertahankan ketajaman tepi karakter
    3. CLAHE (Contrast Limited Adaptive Histogram Equalization) pada Luminance
    4. Unsharp Masking untuk menonjolkan batas karakter teks
    5. Reduksi efek glare lokal
    """
    img = image.copy()
    h, w = img.shape[:2]

    # 1. Non-generative Upscaling bila resolusi rendah
    if min(h, w) < 720:
        scale = min(2.0, 720.0 / float(min(h, w)))
        new_w = int(w * scale)
        new_h = int(h * scale)
        img = cv2.resize(img, (new_w, new_h), interpolation=cv2.INTER_LANCZOS4)

    # 2. Denoising: Bilateral filter menjaga ketajaman garis teks
    if len(img.shape) == 3:
        denoised = cv2.bilateralFilter(img, d=7, sigmaColor=50, sigmaSpace=50)
    else:
        denoised = cv2.bilateralFilter(img, d=7, sigmaColor=50, sigmaSpace=50)

    # 3. CLAHE pada kanal L (Luminance di LAB)
    if len(denoised.shape) == 3:
        lab = cv2.cvtColor(denoised, cv2.COLOR_BGR2LAB)
        l_channel, a_channel, b_channel = cv2.split(lab)
        clahe = cv2.createCLAHE(clipLimit=2.0, tileGridSize=(8, 8))
        l_enhanced = clahe.apply(l_channel)
        enhanced_lab = cv2.merge((l_enhanced, a_channel, b_channel))
        contrast_adjusted = cv2.cvtColor(enhanced_lab, cv2.COLOR_LAB2BGR)
    else:
        clahe = cv2.createCLAHE(clipLimit=2.0, tileGridSize=(8, 8))
        contrast_adjusted = clahe.apply(denoised)

    # 4. Sharpening: Unsharp Masking
    gaussian = cv2.GaussianBlur(contrast_adjusted, (0, 0), 2.0)
    sharpened = cv2.addWeighted(contrast_adjusted, 1.4, gaussian, -0.4, 0)

    return sharpened
