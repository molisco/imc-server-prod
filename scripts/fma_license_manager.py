#!/usr/bin/env python3
"""
FMA License Auditor & File Purge Manager
========================================
Identifica, conta e permite a exclusão segura de arquivos e licenças
associados a tipos específicos de licença (ex: CC0 / Public Domain).
"""

import os
import re
import sys
import json
import sqlite3
import argparse
import urllib.request
import urllib.parse
from html import unescape

DEFAULT_BASE_DIR = "/run/media/paulo-assis/BACKUP001" if os.path.exists("/run/media/paulo-assis/BACKUP001") else os.path.join(os.path.dirname(__file__), "downloads_fma")
DB_PATH = os.path.join(os.path.dirname(__file__), "fma_catalog.db")

USER_AGENT = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"

def get_fma_cc0_catalog(genre="Pop"):
    """Consulta o catálogo oficial do FMA para obter a lista completa de faixas CC0/Public Domain do gênero."""
    print(f"[*] Consultando o índice oficial do Free Music Archive para faixas com licença CC0 / Public Domain ({genre})...")
    cc0_tracks = set()
    page = 1
    total_pages = 1

    while True:
        url = f"https://freemusicarchive.org/search?adv=1&search-genre={genre}&music-filter-public-domain=1&pageSize=50&page={page}"
        req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
        try:
            with urllib.request.urlopen(req, timeout=15) as resp:
                html = resp.read().decode("utf-8", errors="ignore")
        except Exception as e:
            print(f"  [!] Erro ao consultar página {page}: {e}")
            break

        matches = re.findall(r'data-track-info=[\x27]([^\x27]+)[\x27]', html)
        if not matches:
            matches = re.findall(r'data-track-info=[\x22]([^\x22]+)[\x22]', html)
        if not matches:
            break

        for raw in matches:
            try:
                info = json.loads(unescape(raw))
                title = info.get("title", "").strip().lower()
                artist = info.get("artistName", "").strip().lower()
                if artist and title:
                    cc0_tracks.add((artist, title))
            except Exception:
                pass

        p_matches = re.findall(r'page=([0-9]+)', html)
        if p_matches:
            total_pages = max([int(p) for p in p_matches])

        print(f"  -> Coletando página {page} de {total_pages} ({len(cc0_tracks)} faixas CC0 catalogadas no FMA)...", end="\r")

        if page >= total_pages:
            break
        page += 1

    print(f"\n✔ Total de faixas com licença CC0/Public Domain identificadas no FMA: {len(cc0_tracks)}")
    return cc0_tracks

def scan_cc0_files(genre="POP", base_dir=DEFAULT_BASE_DIR):
    """Varre as pastas de músicas e licenças locais e encontra todos os arquivos vinculados a CC0."""
    genre_clean = genre.strip().upper()
    music_dir = os.path.join(base_dir, genre_clean)
    license_dir = os.path.join(base_dir, f"{genre_clean} - Licenses")

    if not os.path.exists(music_dir):
        print(f"[!] Diretório de músicas não encontrado: {music_dir}")
        return []

    cc0_catalog = get_fma_cc0_catalog(genre=genre)

    local_mp3_files = [f for f in os.listdir(music_dir) if f.lower().endswith(".mp3")]
    print(f"[*] Analisando {len(local_mp3_files)} arquivos locais em `{music_dir}`...")

    matched_items = []
    total_size_bytes = 0

    for mp3_filename in local_mp3_files:
        name_no_ext = mp3_filename[:-4]
        # Remove [Gêneros] do final para extrair Artista e Título
        name_no_genre = re.sub(r'\s*\[.*?\]$', '', name_no_ext).strip()

        if " - " in name_no_genre:
            parts = name_no_genre.split(" - ", 1)
            artist = parts[0].strip().lower()
            title = parts[1].strip().lower()

            if (artist, title) in cc0_catalog:
                mp3_path = os.path.join(music_dir, mp3_filename)
                png_path = os.path.join(license_dir, f"{name_no_ext}.png")
                txt_path = os.path.join(license_dir, f"{name_no_ext}.txt")

                size = os.path.getsize(mp3_path) if os.path.exists(mp3_path) else 0
                if os.path.exists(png_path):
                    size += os.path.getsize(png_path)
                if os.path.exists(txt_path):
                    size += os.path.getsize(txt_path)

                total_size_bytes += size

                matched_items.append({
                    "artist_title": f"{parts[0]} - {parts[1]}",
                    "mp3_path": mp3_path,
                    "png_path": png_path,
                    "txt_path": txt_path,
                    "size_bytes": size,
                    "mp3_filename": mp3_filename,
                    "base_name": name_no_ext
                })

    print(f"\n=========================================================================")
    print(f"📊 RESULTADO DA AUDITORIA DE LICENÇA (CC0 / PUBLIC DOMAIN)")
    print(f"=========================================================================")
    print(f"  Gênero Auditado:             {genre_clean}")
    print(f"  Total de Músicas no Disco:   {len(local_mp3_files)}")
    print(f"  Músicas com Licença CC0:     {len(matched_items)} faixas")
    print(f"  Músicas CC BY / Seguras:     {len(local_mp3_files) - len(matched_items)} faixas")
    print(f"  Arquivos Afetados:           {len(matched_items) * 3} arquivos (.mp3, .png, .txt)")
    print(f"  Espaço Ocupado por CC0:      {total_size_bytes / (1024*1024*1024):.2f} GB ({total_size_bytes / (1024*1024):.1f} MB)")
    print(f"=========================================================================\n")

    if matched_items:
        print("📌 Amostra das Primeiras 10 Faixas CC0 Encontradas:")
        for item in matched_items[:10]:
            print(f"  • {item['mp3_filename']}")
        print(f"  ... e mais {len(matched_items) - 10} arquivos.\n" if len(matched_items) > 10 else "")

    return matched_items

def delete_cc0_files(matched_items):
    """Executa a exclusão definitiva dos arquivos identificados."""
    if not matched_items:
        print("[!] Nenhuma faixa CC0 para excluir.")
        return

    print(f"[*] Iniciando exclusão de {len(matched_items)} faixas e seus respectivos comprovantes...")

    deleted_mp3 = 0
    deleted_png = 0
    deleted_txt = 0
    total_freed_bytes = 0

    # Atualiza banco de dados SQLite também
    conn = None
    if os.path.exists(DB_PATH):
        try:
            conn = sqlite3.connect(DB_PATH)
        except Exception:
            pass

    for item in matched_items:
        # 1. Apaga MP3
        if os.path.exists(item["mp3_path"]):
            total_freed_bytes += os.path.getsize(item["mp3_path"])
            os.remove(item["mp3_path"])
            deleted_mp3 += 1

        # 2. Apaga PNG
        if os.path.exists(item["png_path"]):
            total_freed_bytes += os.path.getsize(item["png_path"])
            os.remove(item["png_path"])
            deleted_png += 1

        # 3. Apaga TXT
        if os.path.exists(item["txt_path"]):
            total_freed_bytes += os.path.getsize(item["txt_path"])
            os.remove(item["txt_path"])
            deleted_txt += 1

        # 4. Remove do banco
        if conn:
            try:
                conn.cursor().execute("DELETE FROM tracks WHERE local_filepath = ? OR title LIKE ?", (item["mp3_path"], f"%{item['artist_title']}%"))
            except Exception:
                pass

    if conn:
        conn.commit()
        conn.close()

    print(f"\n=========================================================================")
    print(f"🗑️ EXCLUSÃO CONCLUÍDA COM SUCESSO!")
    print(f"=========================================================================")
    print(f"  Músicas (.mp3) Excluídas:      {deleted_mp3}")
    print(f"  Comprovantes (.png) Excluídos: {deleted_png}")
    print(f"  Arquivos (.txt) Excluídos:     {deleted_txt}")
    print(f"  Total de Arquivos Removidos:   {deleted_mp3 + deleted_png + deleted_txt}")
    print(f"  Espaço em Disco Liberado:      {total_freed_bytes / (1024*1024*1024):.2f} GB ({total_freed_bytes / (1024*1024):.1f} MB)")
    print(f"=========================================================================\n")

def main():
    parser = argparse.ArgumentParser(description="FMA License Manager: Scan and Purge CC0 Tracks")
    parser.add_argument("--genre", type=str, default="POP", help="Gênero a auditar (ex: POP, ELECTRONIC)")
    parser.add_argument("--base-dir", type=str, default=DEFAULT_BASE_DIR, help="Diretório base onde estão as músicas")
    parser.add_argument("--scan", action="store_true", help="Apenas conta e lista os arquivos CC0 sem apagar")
    parser.add_argument("--delete", action="store_true", help="Solicita confirmação e apaga os arquivos CC0")
    parser.add_argument("--force", action="store_true", help="Apaga sem pedir confirmação interativa")

    args = parser.parse_args()

    matched_items = scan_cc0_files(genre=args.genre, base_dir=args.base_dir)

    if not matched_items:
        print("[✨] Nenhuma faixa com licença CC0 encontrada. Seu acervo está limpo!")
        return

    if args.delete:
        if not args.force:
            print(f"⚠️  ATENÇÃO: Você está prestes a excluir {len(matched_items)} faixas (.mp3) e seus comprovantes (.png e .txt).")
            confirm = input("Deseja realmente confirmar a exclusão definitiva? Digite 'SIM' para confirmar: ")
            if confirm.strip().upper() not in ["SIM", "S", "YES", "Y"]:
                print("[!] Operação cancelada pelo usuário. Nenhum arquivo foi excluído.")
                return
        delete_cc0_files(matched_items)
    else:
        print("💡 DICA: Para apagar estes arquivos com segurança, execute:")
        print(f"  python3 scripts/fma_license_manager.py --genre '{args.genre}' --delete")

if __name__ == "__main__":
    main()
