import urllib.request
import urllib.parse

session_json = '{"user_id":"00000000-0000-0000-0000-000000000000","tenant_id":null,"permission_level_id":1,"user_name":"neoroute"}'
cookie_val = urllib.parse.quote(session_json)

req = urllib.request.Request('http://localhost:8080/dashboard/grids/audio')
req.add_header('Cookie', f'imc_session={cookie_val}')

try:
    with urllib.request.urlopen(req) as response:
        html = response.read().decode('utf-8')
        with open('rendered.html', 'w', encoding='utf-8') as f:
            f.write(html)
        print("Success! Rendered HTML saved to rendered.html")
except Exception as e:
    print(f"Error fetching page: {e}")
