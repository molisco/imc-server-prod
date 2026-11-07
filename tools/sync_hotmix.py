import os
import re
import sys
import requests
import psycopg2

def main():
    email = os.getenv("HOTMIX_EMAIL")
    password = os.getenv("HOTMIX_PASSWORD")
    db_url = os.getenv("DATABASE_URL", "postgresql://neoroute:%40%40Mo1isc00%40%40@127.0.0.1:5432/imc_server")

    if not email or not password:
        print("ERROR: HOTMIX_EMAIL and HOTMIX_PASSWORD environment variables are required.")
        print("Usage: HOTMIX_EMAIL=user@example.com HOTMIX_PASSWORD=secret python3 sync_hotmix.py")
        sys.exit(1)

    print(f"Logging in to Hotmix as {email}...")
    session = requests.Session()
    
    # 1. Fetch CSRF Token
    csrf_url = "https://web.hotmixradio.com/api/auth/csrf"
    try:
        csrf_resp = session.get(csrf_url, timeout=10)
        csrf_resp.raise_for_status()
        csrf_token = csrf_resp.json().get("csrfToken")
        print("Fetched CSRF token successfully.")
    except Exception as e:
        print(f"ERROR: Failed to fetch CSRF token from {csrf_url}: {e}")
        sys.exit(1)

    # 2. Authenticate
    signin_url = "https://web.hotmixradio.com/api/auth/callback/credentials"
    payload = {
        "csrfToken": csrf_token,
        "username": email,
        "email": email,
        "password": password,
        "redirect": "false",
        "json": "true"
    }
    
    try:
        login_resp = session.post(signin_url, data=payload, timeout=10)
        login_resp.raise_for_status()
        login_data = login_resp.json()
        if "error" in login_data:
            print(f"ERROR: Authentication failed: {login_data['error']}")
            sys.exit(1)
        print("Authentication successful!")
    except Exception as e:
        print(f"ERROR: Failed to authenticate: {e}")
        sys.exit(1)

    # 3. Retrieve Session Token
    session_url = "https://web.hotmixradio.com/api/auth/session"
    try:
        session_resp = session.get(session_url, timeout=10)
        session_resp.raise_for_status()
        session_data = session_resp.json()
        token = session_data.get("accessToken")
        if not token:
            print("ERROR: Could not retrieve accessToken from NextAuth session.")
            sys.exit(1)
        print("Retrieved NextAuth access token successfully.")
    except Exception as e:
        print(f"ERROR: Failed to retrieve session: {e}")
        sys.exit(1)

    # 4. Fetch radios API with Authorization header
    api_url = "https://checkout.hotmixradio.com/hotmix/api/en/radios"
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    try:
        api_resp = session.get(api_url, headers=headers, timeout=15)
        api_resp.raise_for_status()
        radios = api_resp.json()
        print(f"Fetched {len(radios)} radios from API.")
    except Exception as e:
        print(f"ERROR: Failed to fetch radios from API: {e}")
        sys.exit(1)

    # 5. Filter in-store radios
    instore_radios = []
    for radio in radios:
        categories = radio.get("categories", [])
        is_instore = False
        for cat in categories:
            if cat.get("slug") == "in-store":
                is_instore = True
                break
        if is_instore:
            instore_radios.append(radio)

    print(f"Found {len(instore_radios)} in-store radios in API payload.")

    stations = [
        "Best-of",
        "Cocktail & Dinner Groove",
        "Cool & Calm",
        "Cozy & Coffee",
        "Gentle Luxury",
        "Just Lounge",
        "On the Rocks",
        "Shopping Vibes",
        "Spa Garden",
        "Workout"
    ]

    # Map station name -> stream URL based on keywords
    mapping = {}
    for name in stations:
        clean_name = re.sub(r'[^a-zA-Z0-9]', '', name).lower()
        # "and" vs "&" normalization
        clean_name = clean_name.replace("and", "").replace("amp", "")
        
        matched_url = None
        for radio in instore_radios:
            r_name = radio.get("name", "")
            r_url = radio.get("url", "")
            
            clean_radio_name = re.sub(r'[^a-zA-Z0-9]', '', r_name).lower()
            clean_radio_name = clean_radio_name.replace("and", "")
            
            # Check if keywords match
            if clean_name in clean_radio_name or clean_radio_name in clean_name:
                matched_url = r_url
                break
            
            # Fallback heuristics
            if "cozy" in clean_name and "cozy" in clean_radio_name:
                matched_url = r_url
                break
            if "calm" in clean_name and "calm" in clean_radio_name:
                matched_url = r_url
                break
            if "luxury" in clean_name and "luxury" in clean_radio_name:
                matched_url = r_url
                break
            if "lounge" in clean_name and "lounge" in clean_radio_name:
                matched_url = r_url
                break
            if "rocks" in clean_name and "rocks" in clean_radio_name:
                matched_url = r_url
                break
            if "shopping" in clean_name and "shopping" in clean_radio_name:
                matched_url = r_url
                break
            if "spa" in clean_name and "spa" in clean_radio_name:
                matched_url = r_url
                break
            if "workout" in clean_name and "workout" in clean_radio_name:
                matched_url = r_url
                break
            if "cocktail" in clean_name and "cocktail" in clean_radio_name:
                matched_url = r_url
                break
            if "best" in clean_name and "best" in clean_radio_name:
                matched_url = r_url
                break
                
        if matched_url:
            # Resolve HTTP 302 redirects to get the final 200 OK stream URL
            try:
                head_resp = requests.head(matched_url, allow_redirects=True, timeout=5, headers={"User-Agent": "Mozilla/5.0"})
                if head_resp.url:
                    matched_url = head_resp.url
            except Exception:
                pass
            mapping[name] = matched_url
            print(f"Mapped: '{name}' -> {matched_url}")
        else:
            print(f"WARNING: Could not automatically map station '{name}'")

    if not mapping:
        print("ERROR: No stations were mapped. Check the API response parsing.")
        sys.exit(1)

    # 6. Update Postgres Database
    print("Connecting to database to update stream URLs...")
    try:
        conn = psycopg2.connect(db_url)
        cur = conn.cursor()
        
        updated_count = 0
        for name, url in mapping.items():
            cur.execute("""
                UPDATE stores 
                SET external_stream_url = %s, updated_at = NOW() 
                WHERE external_stream_name = %s
            """, (url, name))
            updated_count += cur.rowcount
            
        conn.commit()
        cur.close()
        conn.close()
        print(f"Successfully updated {updated_count} store record(s) in the database with fresh stream URLs.")
    except Exception as e:
        print(f"ERROR: Database update failed: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
