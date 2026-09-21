"""
Tabel Kode Wilayah Indonesia (Kemendagri)
2 digit pertama: Provinsi
4 digit berikutnya: Kabupaten/Kota (digit 3-4) dan Kecamatan (digit 5-6)
"""

PROVINSI_MAP = {
    "11": "ACEH",
    "12": "SUMATERA UTARA",
    "13": "SUMATERA BARAT",
    "14": "RIAU",
    "15": "JAMBI",
    "16": "SUMATERA SELATAN",
    "17": "BENGKULU",
    "18": "LAMPUNG",
    "19": "KEPULAUAN BANGKA BELITUNG",
    "21": "KEPULAUAN RIAU",
    "31": "DKI JAKARTA",
    "32": "JAWA BARAT",
    "33": "JAWA TENGAH",
    "34": "DAERAH ISTIMEWA YOGYAKARTA",
    "35": "JAWA TIMUR",
    "36": "BANTEN",
    "51": "BALI",
    "52": "NUSA TENGGARA BARAT",
    "53": "NUSA TENGGARA TIMUR",
    "61": "KALIMANTAN BARAT",
    "62": "KALIMANTAN TENGAH",
    "63": "KALIMANTAN SELATAN",
    "64": "KALIMANTAN TIMUR",
    "65": "KALIMANTAN UTARA",
    "71": "SULAWESI UTARA",
    "72": "SULAWESI TENGAH",
    "73": "SULAWESI SELATAN",
    "74": "SULAWESI TENGGARA",
    "75": "GORONTALO",
    "76": "SULAWESI BARAT",
    "81": "MALUKU",
    "82": "MALUKU UTARA",
    "91": "PAPUA",
    "92": "PAPUA BARAT",
    "93": "PAPUA SELATAN",
    "94": "PAPUA TENGAH",
    "95": "PAPUA PEGUNUNGAN",
    "96": "PAPUA BARAT DAYA",
}


def is_valid_wilayah_code(code_6: str) -> bool:
    """
    Validasi 6 digit awal NIK:
    - Harus tepat 6 digit numerik
    - 2 digit pertama harus terdaftar sebagai provinsi yang sah
    - 4 digit berikutnya tidak boleh '0000'
    """
    if not code_6 or len(code_6) != 6 or not code_6.isdigit():
        return False

    prov_code = code_6[:2]
    if prov_code not in PROVINSI_MAP:
        return False

    kab_kec = code_6[2:]
    if kab_kec == "0000":
        return False

    return True


def get_provinsi_name(code_2: str) -> str:
    return PROVINSI_MAP.get(code_2, "TIDAK DIKETAHUI")
