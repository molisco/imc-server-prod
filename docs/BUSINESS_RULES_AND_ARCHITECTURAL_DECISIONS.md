# IMC (Indoor Media Control) - Regras de Negócio e Decisões Arquiteturais Oficiais

Este documento consolida todas as definições técnicas, regras de negócio e decisões de engenharia acordadas e estabelecidas para a plataforma **IMC**.

---

## 1. Distribuição de Comerciais e Inteligência de Grade

### 1.1 Premissa Fundamental de Distribuição Diária
* A distribuição de comerciais é calculada para cobrir **o dia inteiro (00:00 às 23:59)**, garantindo o número contratado de inserções (`insertions_per_day`).
* **Regra de Entrada Tardia / Re-distribuição no Meio do Dia**:
  * Se um novo comercial for cadastrado ou a distribuição for recalculada às **14:00**, a matemática de distribuição é calculada considerando a grade completa do dia todo (para preservar o espaçamento homogêneo e a dispersão horária).
  * O agendamento executa e reproduz **somente os blocos do momento atual para frente** (ex: das 14:00 em diante).
  * Isso garante a consistência do espaçamento mínimo e do algoritmo de dispersão horária sem distorcer os blocos restantes do dia.

### 1.2 Espaçamento e Distribuição Homogênea
* Comerciais de um mesmo anunciante/campanha não devem ser alocados em blocos consecutivos sempre que houver slots alternativos disponíveis.
* Blocos comerciais respeitam a duração máxima configurada por bloco (ex: 2 a 3 minutos).

---

## 2. Playout & Matriz de Templates Especializados (Liquidsoap 2.4.0)

Para evitar regressões cruzadas e vazamentos de recursos entre ambientes, o sistema adota uma **Matriz 2×3** de templates especializados:

```
+---------------------------+----------------------------------------+---------------------------------------+
| Fonte de Áudio / Modo     | SERVIDOR (Web Player / Docker)         | MÁQUINA LOCAL (Hardware / PDV / Loja) |
+---------------------------+----------------------------------------+---------------------------------------+
| ACERVO LOCAL (100% MP3)   | Perfil A1.a: server_acervo.liq.tmpl    | Perfil A2.a: local_acervo.liq.tmpl    |
| HÍBRIDO (Acervo + Stream) | Perfil A1.b: server_hybrid.liq.tmpl    | Perfil A2.b: local_hybrid.liq.tmpl    |
| STREAMING RELAY (100%)    | Perfil A1.c: server_streaming.liq.tmpl | Perfil A2.c: local_streaming.liq.tmpl |
+---------------------------+----------------------------------------+---------------------------------------+
```

### 2.1 Perfis do Servidor (`SERVER_MODE = true`)
* **Saída**: Icecast (MP3 128k ou OGG Vorbis 64k).
* **Perfis**:
  1. `server_acervo.liq.tmpl` (A1.a): 100% Acervo MP3 local via `request.dynamic`. Zero decodificadores ou sockets de streaming externos (elimina totalmente `Avutil.Error` e reconexões fantasmas).
  2. `server_streaming.liq.tmpl` (A1.c): 100% Web Rádio ao vivo via `source.dynamic` e `input.http(self_sync=false)`. Faders de transição e sobreposição imediata de comerciais.
  3. `server_hybrid.liq.tmpl` (A1.b): Alternância dinâmica entre Acervo Local e Web Rádio conforme os blocos horários da grade musical ou sobreposição manual.

### 2.2 Perfis da Máquina Local (`SERVER_MODE = false`)
* **Saída**: Placa de som física direta (`output.pulseaudio` / `output.ao`).
* **Vantagem**: Zero processamento gasto com Icecast, 100% focado no hardware de áudio da loja.

### 2.3 Comutação Dinâmica de Streaming (`source.dynamic` & `self_sync=false`)
* Para permitir a troca dinâmica de rádios online (seja por sobreposição manual de 15/30/60 min ou por transição de grade):
  * É utilizado `source.dynamic(track_sensitive=false, fun () -> active_stream())`.
  * Ao trocar de stream, o Liquidsoap executa uma rotina assíncrona (`do_stream_transition`) com **Fade-out de 1.5s**, instanciação do novo `input.http(self_sync=false, url)` e **Fade-in de 1.5s** com atualização imediata de metadados.
  * A flag `self_sync=false` impede a quebra de sincronia do clock do Liquidsoap e elimina desconexões `Unix.EPIPE` no Icecast.

---

## 3. Inserção de Placas de Veículos, Avisos Instantâneos e Comerciais

### 3.1 Injeção via Harbor HTTP e URIs Anotadas
* O Go envia faixas com anotações de metadados e ganho (`annotate:id="...",type="commercial",liq_amplify="1.51":/caminho/arquivo.mp3`).
* Os endpoints `/push_commercial` e `/push_warning` do Liquidsoap recebem a URI e utilizam diretamente `request.create(target_path)`, sem validação por `file.exists` (que rejeitava URIs anotadas).

### 3.2 Ducking Suave em Avisos Sonoros
* **Fila Dedicada de Avisos**: `warnings_source = request.queue(id="warnings_queue")`.
* **Ducking**: Quando uma placa de veículo ou aviso é disparado via Harbor (`/push_warning` ou `/trigger_vehicle_warning`), a música/streaming de fundo é atenuada para **3% (`smooth_add(p=0.03)`)**.

### 3.3 Prioridade de Comerciais
* **Fila de Comerciais**: `request.queue(id="commercials_queue")` com boost automático de volume de **+20% (`base_volume * 1.2`)**.
* Em modo streaming, a sobreposição é gerenciada por `fallback(track_sensitive=false, [commercials_queue, stream_source])`.

---

## 4. Otimização de Banda para Streaming de Loja (OGG Vorbis Mono 64k)

* **Problema do MP3 Estéreo (128 kbps)**: Consome ~57.6 MB/hora (~1.38 GB/dia por player ligado 24h).
* **Solução OGG Vorbis Mono (64 kbps)**:
  * Como a sonorização de loja é essencialmente **áudio mono distribuído em arandelas/caixas acústicas**, o streaming mono em OGG Vorbis reduz o consumo de banda em **mais de 60%** (~21.6 MB/hora).
  * Totalmente compatível com HTML5 `<audio>` em todos os navegadores modernos (Chrome, Edge, Firefox, Safari).

---

## 5. Web Player e Resiliência de Áudio

* **Watchdog de 3 Segundos**: Um monitor client-side verifica a cada 3s se o `currentTime` do elemento de áudio está avançando. Em caso de travamento de buffer, reconecta automaticamente de forma imperceptível.
* **Persistência de Intenção do Usuário (`userIntentPlay`)**: Eventos transitórios de `pause` gerados pelo browser durante troca de faixas nunca desativam o player se o usuário deu Play.

---

## 6. Controle de Acesso e Isolamento RBAC

* **Perfil Operador (`permission_level_id = 3`)**: Estritamente isolado ao Web Player da sua respectiva loja (`/player/{store_id}`).
* **Proteção de Rotas Administrativas**: O middleware `RequireAdminOrProgrammer` bloqueia qualquer tentativa de navegação fora do Web Player por operadores e os redireciona com segurança.
* **Resolução Segura de Recursos**: Vinhetas, vozes e avisos pré-formatados são resolvidos a partir de registros do banco de dados para evitar referências a arquivos excluídos.

---

## 7. Modo Kiosk do Player & Experiência do Operador

* **Visualização da Fila & Próximos Blocos**: Exibição dos blocos e itens programados para tocar na esteira do dia.
* **Controle de Pulo de Músicas (`Skip`)**: O botão de pular música possui estado visual responsivo e permissão vinculada à configuração da grade musical da loja (habilitado por padrão).
* **Crash-Proofing Client-Side**: Todas as rotinas de manipulação do DOM e cálculos de métricas contam com guards `if (el)` e blocos `try-catch` isolados.

---

## 8. Curadoria Musical Multidimensional, Taxonomia e Ingestão Offline

### 8.1 Ortogonalidade: "Instrumental" é Formato Vocal, não Gênero Musical
* **Regra Taxonômica**: "Instrumental" não deve ser tratado como gênero musical isolado. É um **Perfil Vocal / Formato de Execução** (`Instrumental Puro`, `Vocal Feminino`, `Vocal Masculino`, `Dueto`, `Vocal Chopped/Samples`).
* **Vantagem de Combinação**: Permite cruzar qualquer gênero com perfil vocal (ex: `Gênero: Pop` + `Vocal: Instrumental` para clínicas e farmácias; `Gênero: Jazz` + `Vocal: Instrumental` para cafeterias).

### 8.2 Dimensões de Curadoria (Smart Playlists)
* **Macro-Gêneros & Subgêneros**: Estrutura hierárquica (ex: `Pop` $\rightarrow$ `Synthpop`, `Indie Pop`, `Nu-Disco`).
* **BPM & Energia**: Faixa de andamento (ex: 80-95 BPM relaxante, 115-126 BPM upbeat) e Nível de Energia (1 a 5).
* **Moods / Climas**: Tags múltiplas (`Ensolarado`, `Elegante`, `Relaxante`, `Urbano`, `Descontraído`, `Vibrante`).
* **Fit de Varejo & Dayparting**: Segmentos de loja (`Moda Jovem`, `Cafeteria`, `Farmácia`, `Supermercado`) e momentos do dia (`Manhã`, `Almoço`, `Tarde de Vendas`, `Noite`).

### 8.3 Isolamento de CPU e Ingestão Offline (IMC Music Studio)
* **Zero Concorrência de CPU no Servidor**: O processamento pesado de áudio (cálculo de BPM via `librosa`/`aubio`, detecção vocal, análise R128 a -14 LUFS, pontos de corte `cue_in`/`cue_out` a -35dB) é executado na ferramenta local de estúdio (**IMC Music Studio**).
* **Ciclo de Vida do Arquivo de Áudio**:
  1. **Estúdio (PC do Curador)**: Processa áudio bruto $\rightarrow$ Exporta MP3 tratado a -14 LUFS + manifesto JSON.
  2. **Servidor (`imc-server`)**: Armazena o MP3 tratado e metadados no PostgreSQL (habilita Web Player e preview web com 0% de sobrecarga de CPU).
  3. **Loja (`imc-audio-local`)**: Baixa pela API e grava no disco local criptografado como `.imc` (proteção anti-cópia na filial).

### 8.4 Gestão de Comerciais no Servidor Web
* Comerciais e comunicados rápidos continuam sendo gerenciados diretamente no painel web do `imc-server` para garantir agilidade imediata do varejo (ofertas de última hora e Estúdio de Locuções IA). Por terem curta duração (15s a 45s), o processamento leva < 200ms de CPU e não compromete a estabilidade do playout.

### 8.5 Duração Efetiva & Validação de Faixas para Som Ambiente
* **Tempo Efetivo**: O tempo musical de qualquer faixa não é a duração bruta do arquivo no disco, mas sim a duração calculada entre os pontos de transição:
  $$\text{effective\_duration\_seconds} = \max(0.0, \text{cue\_out} - \text{cue\_in})$$
* **Critério de Aceitação de Faixa**: Para programação contínua em som ambiente, faixas aptas devem ter duração útil entre **2 e 5 minutos** (120s a 300s). Faixas com menos de 2 minutos ou mais de 5 minutos recebem sinalização visual de alerta para o curador.
* **Sincronização em Tempo Real no Canvas**: Ao arrastar ou ajustar manualmente as barras verticais de `cue_in` e `cue_out`, a interface recalcula a duração útil, atualiza a badge de conformidade e recalcula o tempo total exibido.

### 8.6 Aferição de Preenchimento Sonoro e Aptidão no Trecho Útil
* O cálculo do `sound_fullness_score` (0 a 100) e da `ambience_fitness` (`Apto`, `Atenção`, `Inapto`) é executado estritamente sobre a amostra musical fatiada entre `cue_in` e `cue_out` (`useful_y = y[cue_in:cue_out]`), eliminando silêncios de borda, ruídos de fade e vazamentos que distorceriam as métricas espectrais.

### 8.7 Análise Vocal Multi-Janela & Identificação de Idioma Ortogonal
* **Amostragem Multi-Janela**: Para evitar falsos positivos de `Instrumental Puro` causados por introduções longas ou pontes instrumentais, o classificador analisa 3 janelas estratégicas da faixa (15%, 45% e 70%), avaliando a crista de formantes vocais (900 Hz a 3200 Hz via STFT) combinada com modulação melódica contínua de pitch humano ($f_0$ via YIN).
* **Ortogonalidade de Idioma**: Quando a faixa possui vocal (`Vocal Feminino`, `Vocal Masculino`, `Dueto`), o idioma é detectado via NLP contextual/metadados e priors de gênero (Inglês, Português, Espanhol, Alemão). Caso seja `Instrumental Puro`, o idioma é automaticamente fixado em `Instrumental`.

### 8.8 Sugestão Comparativa de Taxonomia (Opção A vs. Opção B)
* O Studio fornece ao curador duas visões complementares para decisão rápida:
  * **Opção A (Acústica MIR - Discogs-400)**: Sugestão baseada estritamente nas características físicas de áudio, formantes e andamento rítmico.
  * **Opção B (Catálogo Global & Charts)**: Sugestão baseada em histórico de paradas mundiais (Billboard, UK Singles, MusicBrainz).
* O curador pode adotar a Opção A ou Opção B com 1 clique, disparando a atualização coordenada dos subgêneros e do contexto comercial de varejo.
* **Atalhos de Reanálise (⟳)**: Cada bloco funcional do painel lateral (Métricas Sônicas, Inteligência de Mercado / Hits, Taxonomia & Perfil Vocal, Contexto Comercial / Varejo) possui botão de reanálise rápida individual no cabeçalho do card.

### 8.9 Registro do BPM para Playout Inteligente no IMC Player
* O BPM é calculado sobre o sinal útil, persistido na coluna `bpm` do SQLite (`studio_catalog.db`), no nó `audio_spec` e no manifesto exportado (`catalog_manifest.json`).
* Esses dados estão estruturados para suportar as seguintes funcionalidades no **IMC Player**:
  1. *Dayparting Rítmico*: Programação de faixas com BPM suave pela manhã (90-110) e acelerado em horários de pico (115-128).
  2. *BPM Matching*: Prevenção de quebras bruscas de energia entre faixas consecutivas (tolerância máxima de transição de $\pm 15\%$).
  3. *Segmentação por Varejo*: Filtragem de andamento adequada ao segmento da loja (clínicas vs. supermercados vs. academias).

---

## 9. Arquitetura de Produção, DNS, Landing Page & Isolamento de Ambientes

### 9.1 Isolamento entre Homologação/Testes e Produção
* **Ambiente de Homologação e Desenvolvimento (`/home/paulo-assis/imc-server`)**:
  * Hospedado no servidor local `192.168.15.250` (`/home/neoroute/imc-server`), montado via NFS4 no `/etc/fstab`.
  * Repositório Git: `git@github.com:molisco/imc-server.git`.
  * Utilizado para desenvolvimento contínuo, testes de novas funcionalidades e validação prévia.
* **Ambiente de Produção (`/home/paulo-assis/imc-server-prod`)**:
  * Hospedado no servidor dedicado público `200.141.23.44` (`imc-server-prod`, Ubuntu 26.04 LTS, `/home/neoroute/imc-server`), montado via `fuse.sshfs` criptografado no `/etc/fstab`.
  * Repositório Git: `git@github.com:molisco/imc-server-prod.git`.
  * **Política de Promoção Estrita**: O ambiente de produção jamais é atualizado automaticamente a partir de testes; só recebe sincronização e deploy quando expressamente validado e autorizado pelo gestor.

### 9.2 Topologia de Domínio Oficial (`indoormedia.com.br`) e DNS
Todos os subdomínios apontam via registro `A` (`IN`) para o IP de produção `200.141.23.44` com certificados HTTPS/TLS Let's Encrypt gerenciados automaticamente pelo container `imc-caddy`:
1. **`indoormedia.com.br` e `www.indoormedia.com.br`**:
   * Exibe a **Landing Page Comercial** (`web/landing/index.html`) com telemetria de construção, cronômetro de lançamento e formulário da Lista VIP.
2. **`app.indoormedia.com.br`**:
   * Roteia para o **Central Brain / Portal Administrativo IMC** (`/login`, gestão multi-tenant, grades musicais/comerciais, estúdio IA e relatórios).
3. **`player.indoormedia.com.br`**:
   * Roteia diretamente para o **Web Player e Operação de Loja** (`/player`, `/player/{id}`, chamadas de pátio/veículos e avisos instantâneos).
4. **`stream.indoormedia.com.br`**:
   * Subdomínio dedicado ao **Streaming de Áudio Contínuo (Icecast + Proxy Go)** sem timeout HTTP para transmissão 24/7.

### 9.3 Captação de Leads da Landing Page (`prelaunch_leads`)
* O formulário `#preLaunchForm` da Landing Page envia os dados via `POST /api/public/leads` para o `imc-server`, persistindo cada cadastro na tabela PostgreSQL `prelaunch_leads` (além de manter fallback local em `localStorage`).
* Administradores autenticados podem consultar a listagem consolidada de leads em tempo real através do endpoint protegido `GET /api/admin/leads`.



