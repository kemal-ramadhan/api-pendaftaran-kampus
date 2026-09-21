import pytest
from app.core.ktp_parser import (
    sanitize_digit_string,
    parse_nik_metadata,
    parse_ktp_text
)
from app.core.wilayah_data import is_valid_wilayah_code


def test_sanitize_digit_string():
    assert sanitize_digit_string("32OlOI52O3O4OOO2") == "3201015203040002"
    assert sanitize_digit_string("3273-Ol l5-O8O5-OOOl") == "3273011508050001"
    assert sanitize_digit_string("B8S5Z2A4T7") == "8855224477"


def test_is_valid_wilayah_code():
    assert is_valid_wilayah_code("327301") is True  # Jabar, Bandung
    assert is_valid_wilayah_code("317101") is True  # DKI, Jakpus
    assert is_valid_wilayah_code("999999") is False # Provinsi 99 tidak ada
    assert is_valid_wilayah_code("320000") is False # 0000 invalid


def test_parse_nik_metadata_male():
    nik = "3273011508050001"
    valid, dob, gender, err = parse_nik_metadata(nik)
    assert valid is True
    assert dob == "2005-08-15"
    assert gender == "L"
    assert err is None


def test_parse_nik_metadata_female():
    # Wanita tanggal lahir 12 Maret 2004 -> 12 + 40 = 52
    nik = "3171015203040002"
    valid, dob, gender, err = parse_nik_metadata(nik)
    assert valid is True
    assert dob == "2004-03-12"
    assert gender == "P"
    assert err is None


def test_parse_ktp_text_extraction():
    raw_ocr = """
    PROVINSI JAWA BARAT
    KOTA BANDUNG
    NIK : 3273011508050001
    Nama : BUDI SANTOSO
    Tempat/Tgl Lahir : BANDUNG, 15-08-2005
    Jenis Kelamin : LAKI-LAKI
    Alamat : JL. MERDEKA NO. 45
    RT/RW : 003/007
    Kel/Desa : BABAKAN CIAMIS
    Kecamatan : SUMUR BANDUNG
    Agama : ISLAM
    Status Perkawinan: BELUM KAWIN
    Pekerjaan : PELAJAR/MAHASISWA
    Kewarganegaraan : WNI
    """
    fields = parse_ktp_text(raw_ocr)

    assert fields["nik"].value == "3273011508050001"
    assert fields["nik"].needs_review is False
    assert fields["nama"].value == "BUDI SANTOSO"
    assert fields["tempat_lahir"].value == "BANDUNG"
    assert fields["tanggal_lahir"].value == "2005-08-15"
    assert fields["jenis_kelamin"].value == "L"
    assert fields["alamat"].value == "JL. MERDEKA NO. 45"
    assert fields["rt_rw"].value == "003/007"
    assert fields["kel_desa"].value == "BABAKAN CIAMIS"
    assert fields["kecamatan"].value == "SUMUR BANDUNG"
    assert fields["agama"].value == "ISLAM"
    assert fields["kewarganegaraan"].value == "WNI"
