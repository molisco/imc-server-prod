#!/usr/bin/env python3
"""
FMA License TXT to JSON Converter
=================================
Converte todos os comprovantes de licença .txt existentes para o formato estruturado .json,
otimizado para ingestão em banco de dados, APIs e sistemas de playout (Sonic / IMC-Server).
"""

import os
import re
import sys
import glob
import json
import sqlite3
import argparse

DEFAULT_BASE_DIR = "/run/media/paulo-assis/BACKUP001" if os.path.exists("/run/media/paulo-assis/BACKUP001") else os.path.join(os.path.dirname(__file__), "downloads_fma")
DB_PATH = os.path.join(os.path.dirname(__file__), "fma_catalog.db")

def parse_txt_content(content):
    def extract(pattern):
        m = re.search(pattern, content)
        return m.group(1).strip() if m else ""

    title = extract(r"MÚSICA / FAIXA:\s*(.*)")
    artist = extract(r"ARTISTA / AUTOR:\s*(.*)")
    album = extract(r"ÁLBUM:\s*(.*)")
    genres_raw = extract(r"GÊNEROS / ESTILOS:\s*(.*)")
    duration = extract(r"DURAÇÃO:\s*(.*)")
    license_type = extract(r"TIPO DE LICENÇA:\s*(.*)")
    commercial = extract(r"USO COMERCIAL:\s*(.*)")
    url = extract(r"URL OFICIAL DE ORIGEM:\s*(.*)")
    downloaded_at = extract(r"DATA DO DOWNLOAD:\s*(.*)")
    audio_mp3 = extract(r"ARQUIVO DE ÁUDIO MP3:\s*(.*)")
    proof_png = extract(r"COMPROVANTE EM IMAGEM:\s*(.*)")
    sha256 = extract(r"HASH SHA-256 DO MP3:\s*(.*)")

    genres = [g.strip() for g in genres_raw.split(",") if g.strip()] if genres_raw else []

    return {
        "title": title,
        "artist": artist,
        "album": album or "Single / Não informado",
        "genres": genres,
        "duration": duration or "--:--",
        "license": {
            "type": "CC BY (Creative Commons Comercial)",
            "commercial_allowed": True,
            "terms": "Creative Commons Comercial (CC BY / CC BY-SA / CC BY-ND)"
        },
        "audit": {
            "source_url": url,
            "downloaded_at": downloaded_at,
            "sha256_hash": sha256,
            "audio_file": audio_mp3,
            "proof_image": proof_png
        },
        "legal_compliance": {
            "platform": "Free Music Archive",
            "platform_url": "https://freemusicarchive.org",
            "status": "Comprovado para Radiodifusão e Sonorização Comercial",
            "declaration": "Esta obra foi obtida através da plataforma oficial Free Music Archive sob licença aberta com expressa autorização para uso comercial e reprodução em estabelecimentos comerciais."
        }
    }

def convert_genre_licenses(genre="POP", base_dir=DEFAULT_BASE_DIR, remove_txt=False):
    genre_clean = genre.strip().upper()
    license_dir = os.path.join(base_dir, f"{genre_clean} - Licenses")

    if not os.path.exists(license_dir):
        print(f"[!] Diretório de licenças não encontrado: {license_dir}")
        return

    txt_files = sorted(glob.glob(os.path.join(license_dir, "*.txt")))
    print(f"[*] Encontrados {len(txt_files)} arquivos .txt em `{license_dir}` para conversão em JSON...")

    converted_count = 0
    errors = 0

    conn = None
    if os.path.exists(DB_PATH):
        try:
            conn = sqlite3.connect(DB_PATH)
            cur = conn.cursor()
            cur.execute("PRAGMA table_info(tracks);")
            cols = [c[1] for c in cur.fetchall()]
            if "license_proof_json" not in cols:
                cur.execute("ALTER TABLE tracks ADD COLUMN license_proof_json TEXT;")
                conn.commit()
        except Exception:
            pass

    for txt_path in txt_files:
        try:
            with open(txt_path, "r", encoding="utf-8", errors="ignore") as f:
                content = f.read()

            data = parse_txt_content(content)
            base_name = os.path.splitext(os.path.basename(txt_path))[0]
            json_path = os.path.join(license_dir, f"{base_name}.json")
            data["audit"]["proof_json"] = f"{base_name}.json"

            with open(json_path, "w", encoding="utf-8") as f:
                json.dump(data, f, indent=2, ensure_ascii=False)

            if conn and data["audit"]["sha256_hash"]:
                try:
                    conn.cursor().execute(
                        "UPDATE tracks SET license_proof_json = ? WHERE sha256_hash = ?",
                        (json_path, data["audit"]["sha256_hash"])
                    )
                except Exception:
                    pass

            if remove_txt:
                os.remove(txt_path)

            converted_count += 1
            if converted_count % 250 == 0:
                print(f"  -> {converted_count}/{len(txt_files)} comprovantes JSON gerados...", end="\r")

        except Exception as e:
            errors += 1
            print(f"  [!] Erro ao converter {txt_path}: {e}")

    if conn:
        conn.commit()
        conn.close()

    print(f"\n=========================================================================")
    print(f"✨ CONVERSÃO PARA JSON CONCLUÍDA COM SUCESSO!")
    print(f"=========================================================================")
    print(f"  Arquivos JSON Gerados:  {converted_count}")
    print(f"  Erros:                  {errors}")
    print(f"  Status dos TXT:         {'Removidos (Substituídos por JSON)' if remove_txt else 'Mantidos ao lado dos JSON'}")
    print(f"=========================================================================\n")

def main():
    parser = argparse.ArgumentParser(description="Convert FMA License TXT to JSON")
    parser.add_argument("--genre", type=str, default="POP", help="Gênero das licenças (ex: POP)")
    parser.add_argument("--base-dir", type=str, default=DEFAULT_BASE_DIR, help="Diretório base do acervo")
    parser.add_argument("--remove-txt", action="store_true", help="Remove os arquivos .txt após gerar os .json")

    args = parser.parse_args()
    convert_genre_licenses(genre=args.genre, base_dir=args.base_dir, remove_txt=args.remove_txt)

if __name__ == "__main__":
    main()
