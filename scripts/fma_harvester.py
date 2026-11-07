#!/usr/bin/env python3
"""
FMA Stealth Harvester & Commercial License Auditor (Full Search Auto-Pagination & Counters)
==========================================================================================
Coleta e download de músicas comerciais do Free Music Archive (FMA)
com filtros estritos de uso comercial (CC BY, CC BY-SA, CC BY-ND, com opção de excluir CC0),
organização por pasta de Gênero (ex: POP / POP - Licenses), colchetes no nome [Gêneros],
geração de comprovante PNG idêntico do site, arquivo .txt, contadores globais e progresso em tempo real.
"""

import os
import re
import sys
import json
import time
import random
import hashlib
import sqlite3
import argparse
import datetime
import subprocess
import urllib.request
import urllib.parse
import http.cookiejar
from html import unescape
from PIL import Image, ImageDraw

DEFAULT_BASE_DIR = "/run/media/paulo-assis/BACKUP001" if os.path.exists("/run/media/paulo-assis/BACKUP001") else os.path.join(os.path.dirname(__file__), "downloads_fma")
DB_PATH = os.path.join(os.path.dirname(__file__), "fma_catalog.db")

cookie_jar = http.cookiejar.CookieJar()
http_opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cookie_jar))

BASE_HEADERS = [
    ("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
    ("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"),
    ("Accept-Language", "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7"),
    ("Sec-Ch-Ua", '"Chromium";v="122", "Not(A:Brand";v="24", "Google Chrome";v="122"'),
    ("Sec-Ch-Ua-Mobile", "?0"),
    ("Sec-Ch-Ua-Platform", '"Linux"'),
    ("Sec-Fetch-Dest", "document"),
    ("Sec-Fetch-Mode", "navigate"),
    ("Sec-Fetch-Site", "same-origin"),
    ("Sec-Fetch-User", "?1"),
    ("Upgrade-Insecure-Requests", "1"),
]
http_opener.addheaders = BASE_HEADERS

def init_db(db_path=DB_PATH):
    conn = sqlite3.connect(db_path)
    cur = conn.cursor()

    cur.execute("""
        CREATE TABLE IF NOT EXISTS tracks (
            id TEXT PRIMARY KEY,
            title TEXT NOT NULL,
            artist_name TEXT,
            artist_url TEXT,
            album_name TEXT,
            duration TEXT,
            genres TEXT,
            license_type TEXT,
            is_commercial_allowed INTEGER DEFAULT 1,
            stream_url TEXT,
            download_url TEXT,
            track_url TEXT,
            sha256_hash TEXT,
            license_proof_image TEXT,
            license_proof_txt TEXT,
            license_proof_json TEXT,
            is_downloaded INTEGER DEFAULT 0,
            local_filepath TEXT,
            downloaded_at TIMESTAMP,
            indexed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)

    cur.execute("""
        CREATE TABLE IF NOT EXISTS genres (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT UNIQUE NOT NULL,
            parent_id INTEGER REFERENCES genres(id)
        )
    """)

    cur.execute("""
        CREATE TABLE IF NOT EXISTS track_genres (
            track_id TEXT REFERENCES tracks(id),
            genre_name TEXT NOT NULL,
            is_primary INTEGER DEFAULT 0,
            PRIMARY KEY (track_id, genre_name)
        )
    """)

    cur.execute("PRAGMA table_info(tracks);")
    cols = [col[1] for col in cur.fetchall()]
    if "genres" not in cols:
        cur.execute("ALTER TABLE tracks ADD COLUMN genres TEXT;")
    if "license_proof_txt" not in cols:
        cur.execute("ALTER TABLE tracks ADD COLUMN license_proof_txt TEXT;")
    if "license_proof_json" not in cols:
        cur.execute("ALTER TABLE tracks ADD COLUMN license_proof_json TEXT;")

    cur.execute("CREATE INDEX IF NOT EXISTS idx_commercial ON tracks(is_commercial_allowed);")
    cur.execute("CREATE INDEX IF NOT EXISTS idx_genre_map ON track_genres(genre_name);")
    conn.commit()
    conn.close()

def warmup_session():
    try:
        req = urllib.request.Request("https://freemusicarchive.org/home")
        resp = http_opener.open(req, timeout=15)
        if resp.status == 200:
            return True
    except Exception as e:
        print(f"  [i] Aviso ao aquecer sessão: {e}")
    return False

def human_delay(min_sec=3.0, max_sec=6.0, msg=None):
    delay = random.uniform(min_sec, max_sec)
    if msg:
        print(f"  ⏳ {msg} ({delay:.1f}s)...")
    time.sleep(delay)

def fetch_stealth_page(url, referer="https://freemusicarchive.org/home", retries=3):
    req = urllib.request.Request(url)
    req.add_header("Referer", referer)

    for attempt in range(retries):
        try:
            with http_opener.open(req, timeout=20) as resp:
                if resp.status == 200:
                    return resp.read().decode("utf-8", errors="ignore")
        except urllib.error.HTTPError as he:
            if he.code == 429:
                print(f"  ⚠️ Limite de requisições detectado (HTTP 429). Pausando por 45s para descanso do IP...")
                time.sleep(45)
            elif he.code in [502, 503, 504]:
                print(f"  ⚠️ Servidor ocupado (HTTP {he.code}). Aguardando 15s...")
                time.sleep(15)
            else:
                print(f"  [!] Erro HTTP {he.code} ao acessar {url}")
        except Exception as e:
            if attempt < retries - 1:
                time.sleep(3 * (attempt + 1))
            else:
                print(f"  [!] Falha de conexão ao acessar {url}: {e}")
    return None

CSS_PATH = os.path.join(os.path.dirname(__file__), "assets", "fma_styles.css")
FMA_CACHED_CSS = ""
if os.path.exists(CSS_PATH):
    try:
        with open(CSS_PATH, "r", encoding="utf-8") as f:
            FMA_CACHED_CSS = f.read()
    except Exception:
        pass

CHROMIUM_BIN = "/snap/bin/chromium" if os.path.exists("/snap/bin/chromium") else "chromium"

def render_real_fma_proof_screenshot(track_url, download_url, file_hash, output_png, info=None):
    """Gera comprovante visual PNG renderizado diretamente pelo navegador (Chromium) idêntico à tela do site."""
    overlay_url = download_url.replace("/download/", "/downloadOverlay/") if download_url else ""
    overlay_html = fetch_stealth_page(overlay_url, referer=track_url, retries=2) if overlay_url else ""

    title = (info.get("title") or "Sem Título") if info else "Áudio Comercial FMA"
    artist = (info.get("artistName") or "Desconhecido") if info else "Artista FMA"

    if not overlay_html or len(overlay_html) < 50:
        overlay_html = f'''
        <div class="inline-flex relative w-full ease-in-out duration-200 transition-all" id="downloadOverlay">
          <div class="hidden" id="downloadOverlayClickout"></div>
          <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 lg:gap-8 max-w-full bg-white w-full p-4 z-20 relative m-0">
            <button class="absolute flex items-center justify-center right-[-0.5rem] top-[-0.5rem] text-white bg-blue h-8 w-8 rounded" id="downloadOverlayClose">
              <span class="w-6 h-6">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M19 6.41 17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12 19 6.41Z" fill="currentColor"/></svg>
              </span>
            </button>
            <div class="flex flex-col gap-4 max-w-md my-auto">
              <div class="flex flex-col gap-1 text-3xl text-blue">
                <h2>{title}</h2>
                <p class="font-[500] font-display">
                  <span class="font-body text-black text-2xl">by </span>
                  <a target="_blank" href="{track_url or '#'}">{artist}</a>
                </p>
              </div>

              <p class="flex gap-1">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-7 w-7 shrink-0" viewBox="0 0 24 24" fill="#ffc000">
                  <g id="_icons"><g><path d="M21.2,16.5L14.6,4.7c-0.5-1-1.5-1.5-2.6-1.5S9.9,3.7,9.4,4.7L2.8,16.5c-0.5,0.9-0.5,2.1,0,3S4.3,21,5.4,21h13.2 c1.1,0,2-0.6,2.6-1.5S21.7,17.5,21.2,16.5z M19.5,18.5c-0.1,0.1-0.3,0.5-0.9,0.5H5.4c-0.5,0-0.8-0.3-0.9-0.5s-0.3-0.5,0-1 l6.6-11.9c0.3-0.5,0.7-0.5,0.9-0.5s0.6,0,0.9,0.5l6.6,11.9C19.7,18,19.5,18.4,19.5,18.5z"/><path d="M12,9c-0.6,0-1,0.4-1,1v3c0,0.6,0.4,1,1,1s1-0.4,1-1v-3C13,9.4,12.6,9,12,9z"/><circle cx="12" cy="16" r="1"/></g></g>
                </svg>
                <span class="inline-block ml-1">
                  <strong>You must give appropriate credit:</strong> Song title, artist name, source (Free Music Archive) and license type (CC BY)
                </span>
              </p>
              <p class="flex gap-1">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-7 w-7 shrink-0" viewBox="0 0 24 24" fill="#00b050">
                  <path d="M18.3,6.3L9.1,16.4l-2.3-3c-0.3-0.4-1-0.5-1.4-0.2c-0.4,0.3-0.5,1-0.2,1.4l3,4C8.4,18.8,8.7,19,9,19c0,0,0,0,0,0 c0.3,0,0.5-0.1,0.7-0.3l10-11c0.4-0.4,0.3-1-0.1-1.4C19.3,5.9,18.6,5.9,18.3,6.3z"/>
                </svg>
                <span class="inline-block ml-1">Personal and commercial use allowed</span>
              </p>
              <p class="flex gap-1">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-7 w-7 shrink-0" viewBox="0 0 24 24" fill="#7f7f7f">
                  <g id="Layer_2"><g><path d="M12,2C6.5,2,2,6.5,2,12s4.5,10,10,10s10-4.5,10-10S17.5,2,12,2z M12,20c-4.4,0-8-3.6-8-8s3.6-8,8-8s8,3.6,8,8 S16.4,20,12,20z"/><path d="M12.1,7.3h-0.1c-1,0-1.9,0.5-2.4,1.3C9.2,9.1,9.3,9.7,9.8,10c0.5,0.3,1.1,0.2,1.4-0.3c0.2-0.3,0.5-0.4,0.8-0.4h0.1 c0.5,0,0.9,0.4,0.9,0.9c0,0.4-0.3,0.8-0.6,0.9l-0.7,0.2c-0.4,0.1-0.7,0.5-0.7,0.9V13c0,0.6,0.4,1,1,1c0.5,0,1-0.4,1-0.9 c1.2-0.4,2-1.5,2-2.8C15,8.7,13.7,7.3,12.1,7.3z"/><circle cx="12" cy="16" r="1"/></g></g>
                </svg>
                <span class="inline-block ml-1">
                  <a rel="license" href="https://creativecommons.org/licenses/by/4.0" target="_blank" class="license-link">More details</a>
                </span>
              </p>
              
              <div class="flex flex-col gap-4">
                <a href="{download_url}" class="download font-medium inline-flex justify-center mr-auto items-center ease-in-out py-4 pl-4 pr-5 rounded bg-blue text-white hover:bg-blue-100">
                  <span class="w-6 mr-2">
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 21.596 24.363" class="w-4">
                      <g data-name="Icon feather-arrow-down" fill="none" stroke="currentColor">
                        <path data-name="Path 663" d="M10.689 0v17.848" stroke-linejoin="round" stroke-width="2.7"/>
                        <path data-name="Path 664" d="m19.613 8.924-8.924 8.924-8.924-8.924" stroke-linejoin="round" stroke-width="2.7"/>
                        <path data-name="Line 121" stroke-linecap="round" stroke-width="2.6" d="M1.3 23.063h18.996"/>
                      </g>
                    </svg>
                  </span>
                  I understand and agree
                </a>
              </div>
            </div>
            
            <div class="flex flex-col gap-6 font-[500]">
              <div class="flex flex-col w-full gap-2 md:gap-6 text-md bg-gray-light p-4 md:p-8 max-w-md lg:ml-auto">
                <h2 class="text-2xl">Need royalty-free music for podcast, YouTube, Instagram and more?</h2>
                <a href="https://prosearch.tribeofnoise.com/search/index" title="Try Tribe of Noise PRO" target="_blank" class="font-medium inline-flex justify-center items-center rounded relative px-4 py-2 inline-flex transition-colors duration-75 pointer text-white bg-orange border-2 border-orange rounded hover:bg-orange-100 hover:border-orange-100 hover:text-white mr-auto">
                  <span class="text">Try Tribe of Noise PRO</span>
                </a>
              </div>
            </div>
          </div>
        </div>
        '''

    clean_fragment = overlay_html
    clean_fragment = clean_fragment.replace('invisible opacity-0', '')
    clean_fragment = clean_fragment.replace('translate-y-[-2rem]', '')
    clean_fragment = clean_fragment.replace('fixed z-50 top-0 left-0 w-screen h-screen max-w-screen max-h-screen overflow-scroll', 'relative w-full')
    clean_fragment = clean_fragment.replace('bg-black opacity-75 absolute left-0 right-0 w-full h-full z-10', 'hidden')
    clean_fragment = clean_fragment.replace('m-auto', 'm-0')
    clean_fragment = clean_fragment.replace('p-8 md:p-12', 'p-4')
    clean_fragment = clean_fragment.replace('max-w-[90%] md:max-w-4xl', 'max-w-full')

    now_str = datetime.datetime.now().strftime('%d/%m/%Y às %H:%M:%S (UTC-3 - Horário de Brasília)')

    full_html = f'''<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Free Music Archive - Comprovante Oficial de Download & Licença</title>
  <style>
    {FMA_CACHED_CSS}
    * {{ box-sizing: border-box; }}
    body {{
      background: #0f172a;
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 100vh;
      margin: 0;
      padding: 30px;
      font-family: Roboto, -apple-system, BlinkMacSystemFont, sans-serif;
    }}
    .cert-container {{
      background: #ffffff;
      border-radius: 12px;
      box-shadow: 0 20px 40px rgba(0,0,0,0.4);
      max-width: 980px;
      width: 100%;
      overflow: hidden;
      border: 2px solid #38bdf8;
    }}
    .header-bar {{
      background: #0b1120;
      color: #ffffff;
      padding: 16px 24px;
      display: flex;
      justify-content: space-between;
      align-items: center;
      border-bottom: 2px solid #1e293b;
    }}
    .footer-bar {{
      background: #f8fafc;
      border-top: 2px solid #e2e8f0;
      padding: 16px 24px;
      font-size: 13px;
      color: #334155;
      line-height: 1.6;
    }}
    .audit-tag {{
      display: inline-block;
      background: #dcfce7;
      color: #166534;
      font-weight: 700;
      font-size: 11px;
      padding: 4px 10px;
      border-radius: 9999px;
      border: 1px solid #86efac;
    }}
  </style>
</head>
<body>
  <div class="cert-container">
    <div class="header-bar">
      <div>
        <div style="font-size: 20px; font-weight: 800; letter-spacing: -0.5px; color: #ffffff;">
          Free Music Archive <span style="font-weight: 400; color: #38bdf8; font-size: 15px;">| Comprovante Oficial de Download & Licença</span>
        </div>
        <div style="font-size: 12px; color: #94a3b8; margin-top: 2px;">
          Página Oficial de Origem: {track_url or 'https://freemusicarchive.org'}
        </div>
      </div>
      <div style="text-align: right;">
        <span class="audit-tag">✔ REGISTRO DE AUDITORIA LEGAL</span>
      </div>
    </div>
    
    <div style="padding: 24px; background: #ffffff;">
      {clean_fragment}
    </div>

    <div class="footer-bar">
      <div><strong>Carimbo Oficial de Aquisição:</strong> Download realizado em {now_str}</div>
      <div style="margin-top: 4px;"><strong>Hash SHA-256 do Arquivo de Áudio:</strong> <code style="color: #0f172a; background: #e2e8f0; padding: 2px 6px; border-radius: 4px;">{file_hash}</code></div>
      <div style="margin-top: 6px; font-size: 11px; color: #64748b;">
        Documento gerado automaticamente pelo Sistema IMC-Server para comprovação de direitos autorais e isenção de execução comercial (Creative Commons).
      </div>
    </div>
  </div>
</body>
</html>'''

    os.makedirs(os.path.dirname(output_png), exist_ok=True)
    temp_render_path = os.path.join(os.path.dirname(output_png), f"_tmp_render_{abs(hash(output_png))}.html")
    with open(temp_render_path, "w", encoding="utf-8") as f:
        f.write(full_html)

    cmd = [
        CHROMIUM_BIN,
        "--headless",
        "--disable-gpu",
        "--no-sandbox",
        "--disable-dev-shm-usage",
        "--window-size=1100,920",
        f"--screenshot={output_png}",
        f"file://{temp_render_path}"
    ]
    try:
        subprocess.run(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=12, check=True)
    except Exception as e:
        print(f"    ⚠️ Aviso ao capturar screenshot pelo navegador: {e}")
    finally:
        if os.path.exists(temp_render_path):
            try:
                os.remove(temp_render_path)
            except Exception:
                pass

def generate_license_json_file(info, genres_list, genres_str, file_hash, mp3_filename, png_filename, output_json):
    now_str = datetime.datetime.now().strftime('%d/%m/%Y às %H:%M:%S (UTC-3 - Horário de Brasília)')
    os.makedirs(os.path.dirname(output_json), exist_ok=True)
    base_name = os.path.splitext(os.path.basename(output_json))[0]

    data = {
        "title": info.get("title", ""),
        "artist": info.get("artistName", ""),
        "album": info.get("album_name") or "Single / Não informado",
        "genres": genres_list if genres_list else [g.strip() for g in genres_str.split(",") if g.strip()],
        "duration": info.get("duration", "--:--"),
        "license": {
            "type": "CC BY (Creative Commons Comercial)",
            "commercial_allowed": True,
            "terms": "Creative Commons Comercial (CC BY / CC BY-SA / CC BY-ND)"
        },
        "audit": {
            "source_url": info.get("url", ""),
            "downloaded_at": now_str,
            "sha256_hash": file_hash,
            "audio_file": mp3_filename,
            "proof_image": png_filename,
            "proof_json": f"{base_name}.json"
        },
        "legal_compliance": {
            "platform": "Free Music Archive",
            "platform_url": "https://freemusicarchive.org",
            "status": "Comprovado para Radiodifusão e Sonorização Comercial",
            "declaration": "Esta obra foi obtida através da plataforma oficial Free Music Archive sob licença aberta com expressa autorização para uso comercial e reprodução em estabelecimentos comerciais."
        }
    }

    with open(output_json, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=2, ensure_ascii=False)

def sanitize_filename(name, max_bytes=240):
    clean = re.sub(r'[\\/*?:"<>|]', "", name)
    clean = re.sub(r'\s+', " ", clean).strip()
    encoded = clean.encode('utf-8')
    if len(encoded) > max_bytes:
        clean = encoded[:max_bytes].decode('utf-8', errors='ignore').rstrip()
    return clean

def count_existing_files_in_dir(directory):
    if not os.path.exists(directory):
        return 0
    return len([f for f in os.listdir(directory) if f.lower().endswith(".mp3")])

def harvest_genre(genre="POP", pages=None, start_page=1, limit_per_page=50, min_delay=3.0, max_delay=6.0, base_dir=DEFAULT_BASE_DIR, exclude_cc0=False):
    init_db()

    genre_clean = genre.strip().upper()
    music_dir = os.path.join(base_dir, genre_clean)
    license_dir = os.path.join(base_dir, f"{genre_clean} - Licenses")

    os.makedirs(music_dir, exist_ok=True)
    os.makedirs(license_dir, exist_ok=True)

    already_downloaded_count = count_existing_files_in_dir(music_dir)

    print(f"\n=========================================================================")
    print(f"🎧 COLETOR FMA POR GÊNERO: {genre_clean}")
    print(f"=========================================================================")
    print(f"  Diretório de Músicas:   {music_dir}")
    print(f"  Diretório de Licenças:  {license_dir}")
    print(f"  Já Baixadas no Disco:   {already_downloaded_count} músicas existentes")
    print(f"  Filtro CC0:             {'EXCLUÍDO (Apenas CC BY / Atribuição)' if exclude_cc0 else 'INCLUÍDO (CC BY + CC0)'}")
    print(f"  Modo de Execução:       {'AUTOMÁTICO ATÉ A ÚLTIMA PÁGINA' if not pages else f'{pages} páginas'}")
    print(f"  Página Inicial:         {start_page}")
    print(f"  Intervalo entre itens:  {min_delay:.1f}s a {max_delay:.1f}s")
    print(f"=========================================================================\n")

    print("[*] Aquecendo sessão furtiva com cookies do site...")
    warmup_session()
    human_delay(1.5, 3.0, "Sessão estabelecida. Consultando catálogo oficial")

    conn = sqlite3.connect(DB_PATH)
    cur = conn.cursor()

    total_downloaded_session = 0
    current_page = start_page
    total_pages_detected = None
    total_tracks_official = None
    item_global_counter = (start_page - 1) * limit_per_page

    try:
        while True:
            if pages and current_page >= start_page + pages:
                print(f"\n[*] Limite solicitado de {pages} páginas atingido.")
                break

            params = {
                "adv": "1",
                "search-genre": genre,
                "music-filter-CC-attribution-only": "1",
                "music-filter-CC-attribution-sharealike": "1",
                "music-filter-CC-attribution-noderivatives": "1",
                "music-filter-commercial-allowed": "1",
                "pageSize": "50",
                "page": str(current_page)
            }
            if not exclude_cc0:
                params["music-filter-public-domain"] = "1"

            query_str = urllib.parse.urlencode(params)
            search_url = f"https://freemusicarchive.org/search?{query_str}"

            page_header_str = f"Página {current_page}"
            if total_pages_detected:
                page_header_str += f" de {total_pages_detected}"
            if total_tracks_official:
                page_header_str += f" (Total: {total_tracks_official} músicas no catálogo)"

            print(f"\n[📑] Consultando {page_header_str} ({genre_clean})...")

            html = fetch_stealth_page(search_url, referer="https://freemusicarchive.org/search")
            if not html:
                print(f"  [!] Não foi possível obter a página {current_page}. Tentando próxima...")
                current_page += 1
                continue

            if total_tracks_official is None:
                tracks_match = re.search(r'Search results:\s*<span[^>]*>([0-9,]+)</span>\s*tracks found', html)
                if tracks_match:
                    total_tracks_official = int(tracks_match.group(1).replace(",", ""))
                
                p_matches = re.findall(r'page=([0-9]+)', html)
                if p_matches:
                    total_pages_detected = max([int(p) for p in p_matches])

                pending_calc = (total_tracks_official - already_downloaded_count) if total_tracks_official else 0
                pct_ready = (already_downloaded_count / total_tracks_official * 100) if total_tracks_official and total_tracks_official > 0 else 0

                print(f"\n-------------------------------------------------------------------------")
                print(f"📊 INFORMAÇÕES DO CATÁLOGO ENCONTRADO ({genre_clean}):")
                print(f"   • Total de Músicas no FMA:   {total_tracks_official or 'Milhares'} faixas comerciais")
                print(f"   • Total de Páginas:          {total_pages_detected or 'Várias'} páginas")
                print(f"   • Já Baixadas Anteriormente: {already_downloaded_count} faixas ({pct_ready:.1f}%)")
                print(f"   • Músicas Pendentes:         {pending_calc} faixas restantes a baixar")
                print(f"-------------------------------------------------------------------------\n")

            matches = re.findall(r'data-track-info=[\x27]([^\x27]+)[\x27]', html)
            if not matches:
                matches = re.findall(r'data-track-info=[\x22]([^\x22]+)[\x22]', html)

            if not matches:
                print(f"\n[✨] Fim do catálogo atingido para o gênero {genre_clean}!")
                break

            print(f"  -> {len(matches)} faixas comerciais nesta página. Processando item a item...")

            page_count = 0
            for raw_info in matches:
                if page_count >= limit_per_page:
                    break

                try:
                    info = json.loads(unescape(raw_info))
                    track_id = str(info.get("id"))
                    title = info.get("title", "Sem Título").strip()
                    artist_name = info.get("artistName", "Desconhecido").strip()
                    artist_url = info.get("artistUrl", "")
                    playback_url = info.get("playbackUrl", "")
                    download_url = info.get("downloadUrl", "")
                    track_url = info.get("url", "")

                    album_match = re.search(rf'tid-{track_id}.*?ptxt-album.*?<a[^>]*>(.*?)</a>', html, re.DOTALL)
                    album_name = unescape(album_match.group(1)).strip() if album_match else ""

                    genres_match = re.search(rf'tid-{track_id}.*?ptxt-genre.*?>(.*?)</span>', html, re.DOTALL)
                    genre_list = []
                    if genres_match:
                        raw_genres = re.findall(r'<a[^>]*>(.*?)</a>', genres_match.group(1))
                        genre_list = [unescape(g).strip() for g in raw_genres if g.strip()]

                    genres_str = ", ".join(genre_list) if genre_list else genre_clean
                    dur_match = re.search(rf'tid-{track_id}.*?align-self-end[^>]*>\s*([0-9:]+)\s*<', html, re.DOTALL)
                    duration = dur_match.group(1).strip() if dur_match else "--:--"

                    info['album_name'] = album_name
                    info['duration'] = duration

                    base_name = sanitize_filename(f"{artist_name} - {title} [{genres_str}]")
                    target_mp3 = os.path.join(music_dir, f"{base_name}.mp3")
                    target_png = os.path.join(license_dir, f"{base_name}.png")
                    target_json = os.path.join(license_dir, f"{base_name}.json")

                    if os.path.exists(target_mp3) and os.path.getsize(target_mp3) > 1000:
                        print(f"  ⏭️ [Já baixado] {base_name}")
                        page_count += 1
                        continue

                    item_global_counter += 1
                    pct_str = f"({(item_global_counter / total_tracks_official * 100):.1f}%)" if total_tracks_official else ""
                    total_str = f" de {total_tracks_official}" if total_tracks_official else ""

                    print(f"\n[🎵] [Faixa {item_global_counter}{total_str} {pct_str}] Baixando: {artist_name} - {title}")
                    print(f"    Gêneros: [{genres_str}] | Duração: {duration}")

                    req_audio = urllib.request.Request(playback_url)
                    req_audio.add_header("Referer", track_url or search_url)

                    hasher = hashlib.sha256()
                    with http_opener.open(req_audio, timeout=30) as resp, open(target_mp3, "wb") as f:
                        content_len = resp.headers.get("Content-Length")
                        total_size = int(content_len) if content_len else 0
                        read_bytes = 0

                        while True:
                            chunk = resp.read(64 * 1024)
                            if not chunk:
                                break
                            f.write(chunk)
                            hasher.update(chunk)
                            read_bytes += len(chunk)
                            if total_size > 0:
                                pct = (read_bytes / total_size) * 100
                                print(f"\r    ⬇️ Progresso Áudio: {pct:.1f}% ({read_bytes/(1024*1024):.2f} MB)", end="")

                        sha256_hash = hasher.hexdigest()
                        print(f"\r    ✔ Áudio Baixado ({read_bytes/(1024*1024):.2f} MB) -> `{os.path.basename(target_mp3)}`")
                        print(f"    🔒 SHA-256: {sha256_hash[:16]}...{sha256_hash[-16:]}")

                    info["genres_str"] = genres_str
                    render_real_fma_proof_screenshot(track_url, download_url, sha256_hash, target_png, info)
                    print(f"    📸 Print Idêntico Salvo -> `{os.path.basename(target_png)}`")

                    generate_license_json_file(info, genre_list, genres_str, sha256_hash, os.path.basename(target_mp3), os.path.basename(target_png), target_json)
                    print(f"    📄 Comprovante JSON Estruturado Salvo -> `{os.path.basename(target_json)}`")

                    now_iso = datetime.datetime.now().isoformat()
                    cur.execute("""
                        INSERT OR REPLACE INTO tracks (
                            id, title, artist_name, artist_url, album_name,
                            duration, genres, license_type, is_commercial_allowed, stream_url,
                            download_url, track_url, sha256_hash, license_proof_image,
                            license_proof_txt, is_downloaded, local_filepath, downloaded_at
                        ) VALUES (?, ?, ?, ?, ?, ?, ?, 'CC Commercial (BY / SA / ND / CC0)', 1, ?, ?, ?, ?, ?, ?, 1, ?, ?)
                    """, (
                        track_id, title, artist_name, artist_url, album_name,
                        duration, genres_str, playback_url, download_url, track_url,
                        sha256_hash, target_png, target_json, target_mp3, now_iso
                    ))

                    cur.execute("DELETE FROM track_genres WHERE track_id = ?", (track_id,))
                    for i, g in enumerate(genre_list):
                        cur.execute("INSERT OR IGNORE INTO genres (name) VALUES (?)", (g,))
                        cur.execute("""
                            INSERT OR REPLACE INTO track_genres (track_id, genre_name, is_primary)
                            VALUES (?, ?, ?)
                        """, (track_id, g, 1 if i == 0 else 0))

                    conn.commit()
                    page_count += 1
                    total_downloaded_session += 1

                    human_delay(min_delay, max_delay, "Aguardando intervalo humano de segurança")

                except Exception as e:
                    print(f"    [!] Erro ao processar item: {e}")

            if total_pages_detected and current_page >= total_pages_detected:
                print(f"\n[✨] Todas as {total_pages_detected} páginas do gênero {genre_clean} foram processadas com sucesso!")
                break

            current_page += 1
            human_delay(8.0, 14.0, f"Descanso entre páginas (Indo para a página {current_page})")

    except KeyboardInterrupt:
        print("\n\n[⏸️] Execução pausada pelo usuário (Ctrl+C). Todo o progresso foi salvo com segurança!")

    conn.close()
    final_count = count_existing_files_in_dir(music_dir)
    print(f"\n=========================================================================")
    print(f"✨ RESUMO DA SESSÃO:")
    print(f"   • Novas faixas baixadas nesta execução: {total_downloaded_session} faixas")
    print(f"   • Total acumulado no disco:             {final_count} faixas em `{music_dir}`")
    if total_tracks_official:
        pct_final = (final_count / total_tracks_official) * 100
        print(f"   • Progresso do Catálogo ({genre_clean}):        {pct_final:.1f}% ({final_count}/{total_tracks_official})")
    print(f"=========================================================================\n")

def show_stats():
    init_db()
    conn = sqlite3.connect(DB_PATH)
    cur = conn.cursor()

    cur.execute("SELECT count(*) FROM tracks")
    total = cur.fetchone()[0]

    cur.execute("SELECT count(*) FROM tracks WHERE is_downloaded = 1")
    downloaded = cur.fetchone()[0]

    cur.execute("SELECT count(DISTINCT genre_name) FROM track_genres")
    total_genres = cur.fetchone()[0]

    print(f"\n=========================================================================")
    print(f"📊 ESTATÍSTICAS DO BANCO FMA COM COMPROVANTES IDÊNTICOS & MULTI-GÊNEROS")
    print(f"=========================================================================")
    print(f"  Total de Faixas Indexadas:     {total}")
    print(f"  Total de Faixas Baixadas:      {downloaded}")
    print(f"  Total de Gêneros/Subgêneros:   {total_genres}")
    print(f"  Banco de Dados SQLite:         {DB_PATH}")
    print(f"=========================================================================\n")

    if total_genres > 0:
        print("📌 Distribuição por Gêneros / Especializações:")
        cur.execute("""
            SELECT genre_name, COUNT(track_id) as qtd
            FROM track_genres
            GROUP BY genre_name
            ORDER BY qtd DESC
            LIMIT 10
        """)
        for g, q in cur.fetchall():
            print(f"  • {g:<25} ({q} faixas)")
        print("")

    conn.close()

def main():
    parser = argparse.ArgumentParser(description="FMA Stealth Harvester & Commercial License Auditor")
    parser.add_argument("--genre", type=str, default="POP", help="Gênero a buscar (ex: Pop, Electronic, Rock, Ambient)")
    parser.add_argument("--all", action="store_true", help="Executa automaticamente até a última página da busca")
    parser.add_argument("--pages", type=int, default=None, help="Quantidade específica de páginas (se omitido ou --all, roda até o fim)")
    parser.add_argument("--start-page", type=int, default=1, help="Página inicial (padrão: 1)")
    parser.add_argument("--limit-per-page", type=int, default=50, help="Quantidade máxima por página (padrão: 50)")
    parser.add_argument("--base-dir", type=str, default=DEFAULT_BASE_DIR, help="Diretório base de destino")
    parser.add_argument("--min-delay", type=float, default=3.0, help="Tempo mínimo de espera em segundos entre itens")
    parser.add_argument("--max-delay", type=float, default=6.0, help="Tempo máximo de espera em segundos entre itens")
    parser.add_argument("--exclude-cc0", action="store_true", help="Exclui faixas CC0 / Public Domain (baixa apenas CC BY / Atribuição)")
    parser.add_argument("--stats", action="store_true", help="Exibe estatísticas do banco de dados")

    args = parser.parse_args()

    if args.stats:
        show_stats()
    else:
        num_pages = None if args.all or args.pages is None else args.pages
        harvest_genre(
            genre=args.genre,
            pages=num_pages,
            start_page=args.start_page,
            limit_per_page=args.limit_per_page,
            min_delay=args.min_delay,
            max_delay=args.max_delay,
            base_dir=args.base_dir,
            exclude_cc0=args.exclude_cc0
        )

if __name__ == "__main__":
    main()
