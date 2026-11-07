# Walkthrough: Liberação e Ajuste do Acesso Externo aos Players de Rádio

Corrigimos com sucesso o problema de acesso externo aos transmissores de rádio (Icecast) via **Caddy Reverse Proxy**.

---

## 🛠️ Alterações Realizadas

### Go Backend (`imc-server`)
- **[docker_orchestrator.go](file:///p:/home/neoroute/imc-server/internal/utils/docker_orchestrator.go)**:
  - Atualizamos a função `RebuildAndReloadCaddy()` expandindo a regra `@icecast` para aceitar qualquer sufixo de tenant (`/m*`), lojas (`/loja*`), transmissões (`/stream*`, `/live*`) e formatos de áudio (`/*.ogg`, `/*.mp3`, `/*.aac`, `/*.m3u`, `/*.pls`).
- **[Caddyfile](file:///p:/home/neoroute/imc-server/Caddyfile)**:
  - Atualizamos a regra `@icecast` no arquivo central do repositório.

### Servidor Linux (`192.168.15.250`)
- **`/home/neoroute/caddy/Caddyfile`**:
  - Aplicamos as novas diretivas no Caddyfile ativo.
  - Executamos `docker exec imc-caddy caddy reload --config /etc/caddy/Caddyfile` para atualizar os roteamentos em tempo real sem interrupção de serviço.
- **`imc-server`**:
  - Recompilamos o binário do servidor central com o gerador Caddy corrigido e reiniciamos o serviço systemd (`imc-server.service`).

---

## 🧪 Validação dos Resultados

1. **Roteamento e Streaming via Caddy Proxy**:
   - `https://imc-audioteste.duckdns.org/mcf4f_l001` -> **200 OK (Stream Ogg/Opus do Icecast)**
   - `https://imc-audioteste.duckdns.org/m0001_l1` -> **200 OK (Stream Ogg/Opus do Icecast)**
   - `http://imc-audioteste.duckdns.org/status-json.xsl` -> **200 OK (Status de rádios)**

2. **Verificação de Cabeçalhos de Áudio (`OggS / OpusHead`)**:
   - Testamos a leitura dos dados binários dos streams via `curl` e `xxd`, confirmando a entrega contínua dos pacotes de áudio da rádio.

---

## 🌐 URLs de Acesso Externo para os Players de Rádio

- **Loja Sorocaba (`mcf4f_l001`):** `https://imc-audioteste.duckdns.org/mcf4f_l001`
- **Matriz 01 (`m0001_l1`):** `https://imc-audioteste.duckdns.org/m0001_l1`
- **Qualquer nova loja cadastrada:** `https://imc-audioteste.duckdns.org/<mount_point>`
