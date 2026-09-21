import re
from datetime import datetime
from typing import Dict, Tuple, Optional
from app.models.schemas import OcrField
from app.core.wilayah_data import is_valid_wilayah_code


DIGIT_REPLACEMENTS = {
    'O': '0', 'o': '0', 'D': '0',
    'I': '1', 'l': '1', '|': '1', 'i': '1', '!': '1',
    'Z': '2', 'z': '2',
    'A': '4',
    'S': '5', 's': '5',
    'G': '6', 'b': '6',
    'T': '7',
    'B': '8',
    'q': '9', 'g': '9'
}


def sanitize_digit_string(raw: str) -> str:
    """Koreksi karakter yang sering salah dibaca OCR pada posisi angka"""
    cleaned = []
    for ch in raw:
        if ch.isdigit():
            cleaned.append(ch)
        elif ch in DIGIT_REPLACEMENTS:
            cleaned.append(DIGIT_REPLACEMENTS[ch])
    return "".join(cleaned)


def parse_nik_metadata(nik: str) -> Tuple[bool, Optional[str], Optional[str], Optional[str]]:
    """
    Validasi dan dekonstruksi NIK 16 digit:
    Returns: (is_valid, tanggal_lahir_iso, jenis_kelamin, error_reason)
    """
    if len(nik) != 16 or not nik.isdigit():
        return False, None, None, "NIK harus tepat 16 digit angka"

    wilayah_code = nik[:6]
    if not is_valid_wilayah_code(wilayah_code):
        return False, None, None, f"Kode wilayah {wilayah_code} tidak terdaftar di Kemendagri"

    day_val = int(nik[6:8])
    month_val = int(nik[8:10])
    year_2d = int(nik[10:12])

    if month_val < 1 or month_val > 12:
        return False, None, None, f"Bulan lahir NIK {month_val} tidak valid"

    jenis_kelamin = "L"
    birth_day = day_val
    if day_val > 40:
        jenis_kelamin = "P"
        birth_day = day_val - 40

    if birth_day < 1 or birth_day > 31:
        return False, None, None, f"Tanggal lahir NIK {birth_day} tidak valid"

    # Perkiraan tahun (asumsi rentang 1940-2025 untuk calon pendaftar PMB)
    current_year_2d = datetime.now().year % 100
    if year_2d <= current_year_2d:
        full_year = 2000 + year_2d
    else:
        full_year = 1900 + year_2d

    try:
        birth_date = datetime(full_year, month_val, birth_day)
        birth_date_str = birth_date.strftime("%Y-%m-%d")
    except ValueError:
        return False, None, None, "Kombinasi tanggal kalender NIK tidak valid"

    return True, birth_date_str, jenis_kelamin, None


def parse_ktp_text(text: str, default_confidence: float = 0.90) -> Dict[str, OcrField]:
    """
    Parser ekstraksi field KTP dari teks OCR
    """
    lines = [line.strip() for line in text.split("\n") if line.strip()]
    full_text = " ".join(lines).upper()

    fields: Dict[str, OcrField] = {}

    # 1. Ekstraksi NIK
    nik_match = re.search(r'(?:NIK|N1K|NlK|N!K)[\s:;.-]*([0-9Ool|!ZzASsGBbTqg\s]{15,22})', full_text)
    nik_value = ""
    nik_needs_review = False
    nik_conf = default_confidence

    if nik_match:
        cand = sanitize_digit_string(nik_match.group(1))
        if len(cand) == 16:
            nik_value = cand
        elif len(cand) > 16:
            nik_value = cand[:16]
        else:
            nik_value = cand
    else:
        # Cari urutan 16 digit berturut-turut di seluruh teks
        all_digit_seq = re.findall(r'\b\d{16}\b', full_text)
        if all_digit_seq:
            nik_value = all_digit_seq[0]
        else:
            # Cari calon digit dengan substitusi karakter
            tokens = full_text.split()
            for tok in tokens:
                san = sanitize_digit_string(tok)
                if len(san) == 16:
                    nik_value = san
                    break

    is_nik_valid = False
    nik_birth_date = None
    nik_gender = None
    if nik_value:
        is_nik_valid, nik_birth_date, nik_gender, _ = parse_nik_metadata(nik_value)
        if not is_nik_valid:
            nik_needs_review = True
            nik_conf = max(0.5, nik_conf - 0.2)
    else:
        nik_needs_review = True
        nik_conf = 0.0

    fields["nik"] = OcrField(
        value=nik_value,
        confidence=round(nik_conf, 2),
        needs_review=nik_needs_review
    )

    # Helper pencari field berbasis regex baris
    def find_field_value(patterns, text_lines):
        for pattern in patterns:
            for idx, line in enumerate(text_lines):
                m = re.search(pattern, line, re.IGNORECASE)
                if m:
                    val = m.group(1).strip()
                    val = re.sub(r'^[:;\s.-]+', '', val).strip()
                    if val:
                        return val, idx
                    # Cek baris berikutnya jika baris ini hanya label
                    if idx + 1 < len(text_lines):
                        next_line = text_lines[idx + 1].strip()
                        next_line = re.sub(r'^[:;\s.-]+', '', next_line).strip()
                        if next_line and not any(re.search(p, next_line, re.IGNORECASE) for p in [
                            r'tempat', r'tanggal', r'jenis', r'alamat', r'rt', r'rw', r'kel', r'kec', r'agama'
                        ]):
                            return next_line, idx + 1
        return "", -1

    # 2. Nama
    nama_val, _ = find_field_value([
        r'NAMA[\s:;.-]+([A-Z\s\',.]+)',
        r'NAME[\s:;.-]+([A-Z\s\',.]+)'
    ], lines)
    if not nama_val:
        # Fallback cari baris tepat setelah baris NIK
        for idx, line in enumerate(lines):
            if "NIK" in line.upper() and idx + 1 < len(lines):
                candidate = lines[idx + 1].strip()
                candidate = re.sub(r'^[:;\s.-]+', '', candidate)
                if len(candidate) > 2 and not candidate.isdigit():
                    nama_val = candidate
                    break

    # Bersihkan noise nama
    nama_val = re.sub(r'[^A-Z\s\',.]', '', nama_val.upper()).strip()
    fields["nama"] = OcrField(
        value=nama_val,
        confidence=default_confidence if nama_val else 0.0,
        needs_review=not bool(nama_val) or len(nama_val) < 3
    )

    # 3. Tempat & Tanggal Lahir
    ttl_raw, _ = find_field_value([
        r'TEMPAT\s*/?\s*TGL\s*LAHIR[\s:;.-]+(.*)',
        r'TEMPAT[\s:;.-]+(.*)',
        r'LAHIR[\s:;.-]+(.*)'
    ], lines)

    tempat_lahir = ""
    tanggal_lahir = ""
    ttl_needs_review = False

    if ttl_raw:
        # Format umum: JAKARTA, 15-08-2005 atau BANDUNG, 15 08 2005
        parts = re.split(r'[,:]', ttl_raw, maxsplit=1)
        if len(parts) >= 2:
            tempat_lahir = re.sub(r'[^A-Z\s]', '', parts[0].upper()).strip()
            date_raw = parts[1].strip()
        else:
            date_match = re.search(r'(\d{1,2}[-\s/]\d{1,2}[-\s/]\d{4})', ttl_raw)
            if date_match:
                date_raw = date_match.group(1)
                tempat_lahir = ttl_raw[:date_match.start()].strip()
                tempat_lahir = re.sub(r'[^A-Z\s]', '', tempat_lahir.upper()).strip()
            else:
                date_raw = ""
                tempat_lahir = ttl_raw.strip()

        # Parse tanggal lahir ke YYYY-MM-DD
        if date_raw:
            date_nums = re.findall(r'\d+', date_raw)
            if len(date_nums) == 3:
                d, m, y = int(date_nums[0]), int(date_nums[1]), int(date_nums[2])
                try:
                    dt = datetime(y, m, d)
                    tanggal_lahir = dt.strftime("%Y-%m-%d")
                except ValueError:
                    ttl_needs_review = True
    elif nik_birth_date:
        # Fallback dari tanggal lahir NIK jika tervalidasi
        tanggal_lahir = nik_birth_date
        ttl_needs_review = True

    # Cross-check tanggal lahir dengan NIK
    if tanggal_lahir and nik_birth_date and tanggal_lahir != nik_birth_date:
        ttl_needs_review = True

    fields["tempat_lahir"] = OcrField(
        value=tempat_lahir,
        confidence=default_confidence if tempat_lahir else 0.5,
        needs_review=not bool(tempat_lahir)
    )
    fields["tanggal_lahir"] = OcrField(
        value=tanggal_lahir,
        confidence=default_confidence if tanggal_lahir and not ttl_needs_review else 0.7,
        needs_review=ttl_needs_review or not bool(tanggal_lahir)
    )

    # 4. Jenis Kelamin
    jk_raw, _ = find_field_value([
        r'JENIS\s*KELAMIN[\s:;.-]+([A-Z-]+)',
        r'KELAMIN[\s:;.-]+([A-Z-]+)'
    ], lines)

    jenis_kelamin = ""
    jk_conf = default_confidence
    if "PEREMPUAN" in jk_raw.upper() or "WANITA" in jk_raw.upper():
        jenis_kelamin = "P"
    elif "LAKI" in jk_raw.upper() or "PRIA" in jk_raw.upper():
        jenis_kelamin = "L"
    elif nik_gender:
        jenis_kelamin = nik_gender
        jk_conf = 0.8

    jk_needs_review = False
    if nik_gender and jenis_kelamin and nik_gender != jenis_kelamin:
        jk_needs_review = True

    fields["jenis_kelamin"] = OcrField(
        value=jenis_kelamin,
        confidence=jk_conf if jenis_kelamin else 0.0,
        needs_review=jk_needs_review or not bool(jenis_kelamin)
    )

    # 5. Alamat
    alamat_val, _ = find_field_value([
        r'ALAMAT[\s:;.-]+(.*)',
        r'ADDRESS[\s:;.-]+(.*)'
    ], lines)
    fields["alamat"] = OcrField(
        value=alamat_val.upper(),
        confidence=default_confidence if alamat_val else 0.5,
        needs_review=not bool(alamat_val)
    )

    # 6. RT/RW
    rtrw_val, _ = find_field_value([
        r'RT\s*/?\s*RW[\s:;.-]+(.*)',
        r'RT[\s:;.-]*(\d{1,3}[\s/]+\d{1,3})'
    ], lines)
    rtrw_match = re.search(r'(\d{1,3})\s*[/]\s*(\d{1,3})', rtrw_val)
    if rtrw_match:
        formatted_rtrw = f"{int(rtrw_match.group(1)):03d}/{int(rtrw_match.group(2)):03d}"
    else:
        formatted_rtrw = rtrw_val.strip()

    fields["rt_rw"] = OcrField(
        value=formatted_rtrw,
        confidence=default_confidence if formatted_rtrw else 0.5,
        needs_review=not bool(formatted_rtrw)
    )

    # 7. Kel/Desa
    kel_val, _ = find_field_value([
        r'KEL\s*/?\s*DESA[\s:;.-]+(.*)',
        r'KELURAHAN[\s:;.-]+(.*)',
        r'DESA[\s:;.-]+(.*)'
    ], lines)
    kel_val = re.sub(r'[^A-Z0-9\s]', '', kel_val.upper()).strip()
    fields["kel_desa"] = OcrField(
        value=kel_val,
        confidence=default_confidence if kel_val else 0.5,
        needs_review=not bool(kel_val)
    )

    # 8. Kecamatan
    kec_val, _ = find_field_value([
        r'KECAMATAN[\s:;.-]+(.*)',
        r'KEC[\s:;.-]+(.*)'
    ], lines)
    kec_val = re.sub(r'[^A-Z0-9\s]', '', kec_val.upper()).strip()
    fields["kecamatan"] = OcrField(
        value=kec_val,
        confidence=default_confidence if kec_val else 0.5,
        needs_review=not bool(kec_val)
    )

    # 9. Agama
    agama_val, _ = find_field_value([
        r'AGAMA[\s:;.-]+(.*)',
        r'RELIGION[\s:;.-]+(.*)'
    ], lines)
    agama_standard = ""
    for ag in ["ISLAM", "KRISTEN", "KATOLIK", "HINDU", "BUDDHA", "KONGHUCU"]:
        if ag in agama_val.upper():
            agama_standard = ag
            break
    if not agama_standard and agama_val:
        agama_standard = agama_val.upper()

    fields["agama"] = OcrField(
        value=agama_standard,
        confidence=default_confidence if agama_standard else 0.5,
        needs_review=not bool(agama_standard)
    )

    # 10. Status Perkawinan
    status_kawin, _ = find_field_value([
        r'STATUS\s*PERKAWINAN[\s:;.-]+(.*)',
        r'STATUS[\s:;.-]+(.*)'
    ], lines)
    fields["status_perkawinan"] = OcrField(
        value=status_kawin.upper(),
        confidence=default_confidence if status_kawin else 0.5,
        needs_review=False
    )

    # 11. Pekerjaan
    pekerjaan_val, _ = find_field_value([
        r'PEKERJAAN[\s:;.-]+(.*)'
    ], lines)
    fields["pekerjaan"] = OcrField(
        value=pekerjaan_val.upper(),
        confidence=default_confidence if pekerjaan_val else 0.5,
        needs_review=False
    )

    # 12. Kewarganegaraan
    kewarganegaraan_val, _ = find_field_value([
        r'KEWARGANEGARAAN[\s:;.-]+(.*)'
    ], lines)
    kwn = "WNI"
    if "WNA" in kewarganegaraan_val.upper():
        kwn = "WNA"
    fields["kewarganegaraan"] = OcrField(
        value=kwn,
        confidence=0.95,
        needs_review=False
    )

    return fields
