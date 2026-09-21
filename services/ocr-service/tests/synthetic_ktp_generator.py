import os
import random
from typing import Dict, Tuple
from PIL import Image, ImageDraw, ImageFont, ImageFilter, ImageEnhance
import numpy as np
import cv2

SAMPLE_DATASETS = [
    {
        "nik": "3273011508050001",
        "provinsi": "PROVINSI JAWA BARAT",
        "kota": "KOTA BANDUNG",
        "nama": "BUDI SANTOSO",
        "tempat_lahir": "BANDUNG",
        "tanggal_lahir": "15-08-2005",
        "jenis_kelamin": "LAKI-LAKI",
        "alamat": "JL. MERDEKA NO. 45",
        "rt_rw": "003/007",
        "kel_desa": "BABAKAN CIAMIS",
        "kecamatan": "SUMUR BANDUNG",
        "agama": "ISLAM",
        "status_perkawinan": "BELUM KAWIN",
        "pekerjaan": "PELAJAR/MAHASISWA",
        "kewarganegaraan": "WNI"
    },
    {
        "nik": "3171015203040002",
        "provinsi": "PROVINSI DKI JAKARTA",
        "kota": "KOTA JAKARTA PUSAT",
        "nama": "SITI AMINAH",
        "tempat_lahir": "JAKARTA",
        "tanggal_lahir": "12-03-2004",  # Perempuan: tgl 12 + 40 = 52
        "jenis_kelamin": "PEREMPUAN",
        "alamat": "JL. CEMPAKA PUTIH TENGAH NO. 12",
        "rt_rw": "005/002",
        "kel_desa": "CEMPAKA PUTIH TIMUR",
        "kecamatan": "CEMPAKA PUTIH",
        "agama": "ISLAM",
        "status_perkawinan": "BELUM KAWIN",
        "pekerjaan": "MAHASISWA",
        "kewarganegaraan": "WNI"
    },
    {
        "nik": "3374012011030003",
        "provinsi": "PROVINSI JAWA TENGAH",
        "kota": "KOTA SEMARANG",
        "nama": "ANDI WIJAYA",
        "tempat_lahir": "SEMARANG",
        "tanggal_lahir": "20-11-2003",
        "jenis_kelamin": "LAKI-LAKI",
        "alamat": "JL. PAHLAWAN NO. 88",
        "rt_rw": "002/004",
        "kel_desa": "MUGASSARI",
        "kecamatan": "SEMARANG SELATAN",
        "agama": "KRISTEN",
        "status_perkawinan": "BELUM KAWIN",
        "pekerjaan": "KARYAWAN SWASTA",
        "kewarganegaraan": "WNI"
    }
]


def get_default_font(size: int):
    # Coba gunakan font sistem umum jika ada, atau default PIL
    common_fonts = [
        "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
        "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
        "/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf",
        "C:\\Windows\\Fonts\\arialbd.ttf",
        "C:\\Windows\\Fonts\\arial.ttf"
    ]
    for font_path in common_fonts:
        if os.path.exists(font_path):
            try:
                return ImageFont.truetype(font_path, size)
            except Exception:
                pass
    return ImageFont.load_default()


def create_synthetic_ktp_image(data: dict, width: int = 1000, height: int = 630) -> Image.Image:
    """
    Membuat citra KTP sintetis buatan (DILARANG MENGGUNAKAN DATA ATAU FOTO KTP ASLI)
    """
    img = Image.new("RGB", (width, height), color=(195, 225, 245))
    draw = ImageDraw.Draw(img)

    # Tambahkan pola garis gelombang latar belakang sintetis
    for y in range(0, height, 15):
        draw.line([(0, y), (width, y)], fill=(180, 215, 238), width=1)
    for x in range(0, width, 25):
        draw.line([(x, 0), (x, height)], fill=(185, 218, 240), width=1)

    font_header = get_default_font(26)
    font_nik = get_default_font(34)
    font_labels = get_default_font(18)
    font_values = get_default_font(18)

    # Header KTP
    prov_text = data["provinsi"]
    kota_text = data["kota"]
    draw.text((width // 2, 28), prov_text, fill=(20, 20, 20), font=font_header, anchor="mm")
    draw.text((width // 2, 60), kota_text, fill=(20, 20, 20), font=font_header, anchor="mm")

    # Baris NIK
    draw.text((50, 105), "NIK", fill=(0, 0, 0), font=font_nik)
    draw.text((220, 105), f": {data['nik']}", fill=(0, 0, 0), font=font_nik)

    # Field-field data
    fields_to_draw = [
        ("Nama", data["nama"]),
        ("Tempat/Tgl Lahir", f"{data['tempat_lahir']}, {data['tanggal_lahir']}"),
        ("Jenis Kelamin", data["jenis_kelamin"]),
        ("Alamat", data["alamat"]),
        ("  RT/RW", data["rt_rw"]),
        ("  Kel/Desa", data["kel_desa"]),
        ("  Kecamatan", data["kecamatan"]),
        ("Agama", data["agama"]),
        ("Status Perkawinan", data["status_perkawinan"]),
        ("Pekerjaan", data["pekerjaan"]),
        ("Kewarganegaraan", data["kewarganegaraan"]),
        ("Berlaku Hingga", "SEUMUR HIDUP")
    ]

    start_y = 158
    line_spacing = 33
    for idx, (label, val) in enumerate(fields_to_draw):
        curr_y = start_y + (idx * line_spacing)
        draw.text((50, curr_y), label, fill=(10, 10, 10), font=font_labels)
        draw.text((270, curr_y), f": {val}", fill=(10, 10, 10), font=font_values)

    # Pasfoto sintetis (kotak ilustrasi abstrak non-manusia asli)
    photo_box = (width - 240, 140, width - 40, 420)
    draw.rectangle(photo_box, fill=(50, 110, 190), outline=(20, 40, 80), width=3)
    # Siluet sintetis abstrak
    draw.ellipse([width - 170, 180, width - 110, 240], fill=(230, 200, 175))
    draw.chord([width - 210, 250, width - 70, 415], start=0, end=180, fill=(40, 60, 120))

    # Tanda tangan fiktif sintetis
    draw.text((width - 190, 440), "KOTA BANDUNG", fill=(20, 20, 20), font=get_default_font(14))
    draw.text((width - 180, 460), "15-08-2020", fill=(20, 20, 20), font=get_default_font(14))
    draw.line([(width - 210, 520), (width - 180, 495), (width - 150, 530), (width - 70, 500)], fill=(10, 10, 40), width=3)

    return img


def apply_gaussian_blur(img: Image.Image, radius: float = 2.5) -> Image.Image:
    return img.filter(ImageFilter.GaussianBlur(radius=radius))


def apply_darkness(img: Image.Image, factor: float = 0.45) -> Image.Image:
    enhancer = ImageEnhance.Brightness(img)
    return enhancer.enhance(factor)


def apply_glare(img: Image.Image) -> Image.Image:
    """Tambahkan hotspot silau cahaya di atas teks KTP"""
    overlay = Image.new("RGBA", img.size, (255, 255, 255, 0))
    draw = ImageDraw.Draw(overlay)
    # Lingkaran silau putih transparan di area tengah kiri
    cx, cy = int(img.width * 0.35), int(img.height * 0.45)
    r = 130
    draw.ellipse([cx - r, cy - r, cx + r, cy + r], fill=(255, 255, 255, 170))
    return Image.alpha_composite(img.convert("RGBA"), overlay).convert("RGB")


def apply_tilt_and_perspective(img: Image.Image, angle: float = 12.0) -> Image.Image:
    """Tempatkan kartu di atas background meja dengan kemiringan rotasi"""
    cv_img = np.array(img)
    h, w = cv_img.shape[:2]

    # Buat kanvas meja yang lebih luas
    canvas_h, canvas_w = int(h * 1.5), int(w * 1.5)
    canvas = np.full((canvas_h, canvas_w, 3), (120, 110, 105), dtype=np.uint8)

    # Rotasikan kartu
    center = (w // 2, h // 2)
    M = cv2.getRotationMatrix2D(center, angle, 0.9)
    rotated = cv2.warpAffine(cv_img, M, (w, h), borderValue=(120, 110, 105))

    # Tempatkan di tengah kanvas
    offset_y = (canvas_h - h) // 2
    offset_x = (canvas_w - w) // 2
    canvas[offset_y:offset_y + h, offset_x:offset_x + w] = rotated

    return Image.fromarray(canvas)


def generate_benchmark_dataset(output_dir: str):
    os.makedirs(output_dir, exist_ok=True)

    dataset_manifest = []

    for idx, sample in enumerate(SAMPLE_DATASETS):
        base_img = create_synthetic_ktp_image(sample)
        name_prefix = f"ktp_sample_{idx + 1}"

        # 1. Clean
        clean_path = os.path.join(output_dir, f"{name_prefix}_clean.jpg")
        base_img.save(clean_path, quality=95)
        dataset_manifest.append({"path": clean_path, "variant": "clean", "ground_truth": sample})

        # 2. Blurry (buram ringan/sedang yang bisa diperbaiki)
        blur_img = apply_gaussian_blur(base_img, radius=1.1)
        blur_path = os.path.join(output_dir, f"{name_prefix}_blurry.jpg")
        blur_img.save(blur_path, quality=90)
        dataset_manifest.append({"path": blur_path, "variant": "blurry", "ground_truth": sample})

        # 3. Dark (redup/gelap)
        dark_img = apply_darkness(base_img, factor=0.45)
        dark_path = os.path.join(output_dir, f"{name_prefix}_dark.jpg")
        dark_img.save(dark_path, quality=90)
        dataset_manifest.append({"path": dark_path, "variant": "dark", "ground_truth": sample})

        # 4. Glare (silau)
        glare_img = apply_glare(base_img)
        glare_path = os.path.join(output_dir, f"{name_prefix}_glare.jpg")
        glare_img.save(glare_path, quality=90)
        dataset_manifest.append({"path": glare_path, "variant": "glare", "ground_truth": sample})

        # 5. Tilted (miring)
        tilted_img = apply_tilt_and_perspective(base_img, angle=12.0)
        tilted_path = os.path.join(output_dir, f"{name_prefix}_tilted.jpg")
        tilted_img.save(tilted_path, quality=90)
        dataset_manifest.append({"path": tilted_path, "variant": "tilted", "ground_truth": sample})

        # 6. Severe Blur (wajib retake_required)
        sev_blur_img = apply_gaussian_blur(base_img, radius=6.0)
        sev_blur_path = os.path.join(output_dir, f"{name_prefix}_severe_blur.jpg")
        sev_blur_img.save(sev_blur_path, quality=80)
        dataset_manifest.append({"path": sev_blur_path, "variant": "severe_blur", "ground_truth": sample, "expect_retake": True})

    return dataset_manifest


if __name__ == "__main__":
    out = os.path.join(os.path.dirname(__file__), "fixtures")
    manifest = generate_benchmark_dataset(out)
    print(f"Generated {len(manifest)} synthetic KTP test fixtures in {out}")
