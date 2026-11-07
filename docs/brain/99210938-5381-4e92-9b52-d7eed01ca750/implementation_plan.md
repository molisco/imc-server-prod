# Plano de Implementação: Liberação e Ajuste do Acesso Externo às Rádios (Caddy Proxy & Playout)

Identificamos a causa raiz exata do problema de acesso externo aos players de rádio. O proxy reverso **Caddy** possuía uma regra de filtro (`@icecast`) limitada apenas a `/m0001*` e `/loja*`. Quando os players/lojas geravam pontos de montagem com outros sufixos (por exemplo `/mcf4f_l001`), o Caddy redirecionava a requisição para o backend central Go (`192.168.15.250:8080`) em vez do Icecast (`imc-icecast:8000`), resultando em erro de acesso (404 Not Found).

Além disso, as configurações do Caddy nos arquivos Go (`internal/utils/docker_orchestrator.go`) precisam ser atualizadas para que qualquer regeração automática preserve o suporte universal aos streams.

---

## Proposed Changes

### Central Brain Server (Go Backend)

#### [MODIFY] [docker_orchestrator.go](file:///p:/home/neoroute/imc-server/internal/utils/docker_orchestrator.go)
- Expandir o matcher `@icecast` na função `RebuildAndReloadCaddy()` para aceitar `/m*`, `/loja*`, `/status*`, `/admin*`, `/stream*`, `/live*`, `/*.ogg`, `/*.mp3`, `/*.aac`, `/*.m3u` e `/*.pls`.
- Adicionar compatibilidade para portas alternativas comuns (como `:8443`) nos blocos HTTP e HTTPS.

#### [MODIFY] [Caddyfile](file:///p:/home/neoroute/imc-server/Caddyfile)
- Atualizar o arquivo local `Caddyfile` do repositório com as novas regras abrangentes de roteamento Icecast.

---

### Linux Host Server (`192.168.15.250`)

#### [MODIFY] `/home/neoroute/caddy/Caddyfile`
- Atualizar o arquivo de configuração ativo do Caddy no servidor Linux.
- Recarregar as configurações no container via `docker exec imc-caddy caddy reload --config /etc/caddy/Caddyfile`.
- Recompilar o executável `imc-server` Go no Linux e reiniciar o serviço `imc-server.service`.

---

## Verification Plan

### Manual Verification
1. **Teste do Icecast local via Caddy (HTTP / HTTPS)**:
   - `curl -i http://imc-audioteste.duckdns.org/mcf4f_l001` -> Confirmar se o Icecast responde com streaming ou header HTTP 200/404 do Icecast (e não 404 do Go).
   - `curl -i http://imc-audioteste.duckdns.org/m0001_l1` -> Confirmar que a loja 001 continua operando normalmente.
   - `curl -i http://imc-audioteste.duckdns.org/status-json.xsl` -> Verificar a lista de rádios ativas no Icecast.
2. **Teste via Caddy Hot Reload**:
   - Verificar `docker logs imc-caddy` para garantir recarga limpa sem erros de sintaxe no Caddyfile.
