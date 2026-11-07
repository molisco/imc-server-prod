# 📋 LISTA MESTRE DE TAREFAS E PENDÊNCIAS - SISTEMA IMC

Este documento contém o planejamento centralizado de **funcionalidades pendentes (TODO)** e o **histórico de recursos concluídos** para os projetos **IMC Server** (Painel/Centralizador) e **IMC Audio Local** (Player de Loja).

---

## 🟢 1. RECURSOS CONCLUÍDOS RECENTEMENTE (DONE)

- [x] **Conteinerização Multi-Tenant**: Orquestração Docker automática (`docker_orchestrator.go`) por loja com suporte a timezone (`TZ`) e recarga a quente do Caddy sem queda de conexão.
- [x] **Web Terminal Interativo**: Terminal PTY embarcado no painel do servidor via WebSocket (`terminal_handler.go`) para conexão com contêineres Docker e shells remotos.
- [x] **Aba de Eventos no Player Local**: Interface dedicada para gerenciamento e agendamento de eventos locais no `imc-audio-local`.
- [x] **Avisos de Veículos**: Módulo de chamada por placa de veículo com interrupção e ducking inteligente (`vehicle_warning.go`).
- [x] **Injeção de Avisos & Chimes (Din-Don)**: Suporte a pacotes de voz (`voice_packs`), vinhetas/chimes antes dos anúncios e avisos pré-formatados via Liquidsoap `input.harbor`.
- [x] **Controle Avançado de Volume & Ducking**: Aumento automático de volume em +20% para blocos comerciais e atenuação da música de fundo para 3% (`smooth_add`) durante avisos.
- [x] **Coletor e Auditor FMA com Comprovantes Digitais & JSON**: Ferramenta automatizada (`fma_harvester.py`) para catalogação e download furtivo com paginação de faixas comerciais do FMA (`CC BY`, `CC BY-SA`, `CC BY-ND`, exclusão de `CC0`), geração de comprovantes visuais (.png) idênticos ao site e arquivos (.json) estruturados com Hash SHA-256 e metadados completos.
- [x] **Gerenciador de Purga de Licenças CC0 (`fma_license_manager.py`)**: Script de auditoria, contagem e remoção segura de faixas em Domínio Público/CC0 e seus respectivos comprovantes e registros de banco.
- [x] **Migrador de Licenças para JSON (`migrate_licenses_to_json.py`)**: Conversão automatizada de arquivos legados .txt para .json com paridade 1:1 e proteção de limite de tamanho de nome de arquivo (240 bytes).
- [x] **Crash-Proofing e Null-Safety no Player**: Reforço no Web Player (`customer_player_handler.go`, `customer_player.html`) com watchdog contínuo de 3 segundos para evitar interrupções de áudio.
- [x] **Bancada de Curadoria & Ingestão Musical (IMC Music Studio)**:
  * Extração física de DSP (BPM, Perfil Vocal VAD, EBU R128 a -14 LUFS, Cue-In/Cue-Out a -35dB).
  * Classificador Inteligente de Gênero, Subgêneros (40+ especializações) e Heurísticas Festivas/Sazonais (Oktoberfest, Carnaval, Festa Junina, Natal).
  * Interface com Medidor de Certeza do Gênero (Gauge %), Chips clicáveis de Top Subgêneros e Modal de Curadoria em Lote (`POST /api/batch-curatorship`).
  * Exportador com validação de conformidade e manifesto `catalog_manifest.json`.

---

## 🟡 2. PENDÊNCIAS E TAREFAS PLANEJADAS (TODO)

### 🚀 F. Sistema de Auto-Atualização Automática (Zero-Touch OTA Updates)
- [ ] **Módulo de Distribuição Central no `imc-server`**:
  * [ ] Endpoints de catálogo e entrega de binários (`/api/v1/terminal/updates/check` e `/api/v1/terminal/updates/download/{version}`).
  * [ ] Painel Web de Gestão de Lojas com exibição da versão ativa de cada terminal (`Windows` / `Linux`) e status online/offline.
  * [ ] Disparo de atualização em tempo real via WebSocket (por loja individual ou em lote).
  * [ ] Agendamento automático de atualização fora do expediente da loja (ex: 03:00 da madrugada).
- [ ] **Cliente Auto-Atualizável no `imc-audio-local` (Windows & Linux)**:
  * [ ] Rotina de verificação periódica de nova versão (Pull) e escuta de comandos remotos (Push via WebSocket).
  * [ ] Download seguro em segundo plano com validação estrita de integridade via **Hash SHA-256** antes de executar.
  * [ ] Mecanismo de Hot-Swap atômico cross-platform:
    * **Linux**: Substituição atômica do binário em `/usr/local/bin/imc-audio-local` com recarga via `systemd`.
    * **Windows**: Técnica de renomeação atômica (`imc-audio-local.exe` $\rightarrow$ `imc-audio-local.old.exe`), inserção do novo executável e reinício de Windows Service.
  * [ ] **Watchdog de Boot & Rollback Automático**: Verificação de saúde de 15 segundos após a inicialização. Se a nova versão falhar, reverte imediatamente para a versão `.old` sem deixar a loja em silêncio.
  * [ ] **Ponto de Troca Suave (Zero-Disruption)**: Aguardar o término do bloco comercial ou transição de faixa para aplicar a troca.

---

### 🎨 G. Módulo de Curadoria & Ingestão Inteligente (IMC Music Studio) & Smart Playlists
- [x] **Software Standalone de Curadoria (Desktop / Estúdio de Ingestão)**:
  * [x] **Isolamento de CPU do Servidor**: Retirar todo o processamento pesado de áudio (FFmpeg, R128, BPM, Cues) do `imc-server`. O servidor apenas recebe metadados leves e os arquivos tratados prontos.
  * [x] **Extração Física Automatizada de Áudio (DSP)**:
    * Detecção precisa de **BPM / Andamento** via bibliotecas de áudio (`librosa`/`aubio`).
    * Detecção automática de **Perfil Vocal** (VAD / Separação Harmônica): `Instrumental Puro` vs `Vocal Feminino` vs `Vocal Masculino` vs `Dueto`.
    * Normalização uniforme a **-14 LUFS** (EBU R128) e detecção de silêncio para cálculo de **`cue_in` e `cue_out`** (-35dB).
    * Cálculo de **Duração Efetiva** baseada em Cues e validação para som ambiente (2 a 5 min).
    * Aferição de **Preenchimento Sonoro** (`sound_fullness_score`) e **Aptidão de Som Ambiente** (`ambience_fitness`) estritamente sobre o trecho útil entre Cues.
  * [x] **Classificação Inteligente por IA (Tagging)**:
    * Sugestão de **Macro-Gênero** (Pop, Dance, Jazz, MPB, Rock, etc.) e **Subgêneros / Especializações** (Synthpop, Deep House, Smooth Jazz, Acoustic Pop).
    * Classificação de **Mood / Clima** (`Ensolarado`, `Elegante`, `Relaxante`, `Descontraído`, `Urbano`, `Vibrante`).
    * Classificação de **Nível de Energia** (1 a 5) e **Segmentos de Varejo / Fit Comercial** (`Moda Jovem`, `Cafeteria`, `Farmácia`, `Academia`, `Supermercado`).
    * Sugestão de **Dayparting** (Manhã, Almoço, Tarde de Vendas, Noite).
    * Detecção de **Idioma** ortogonal (Inglês, Português, Espanhol, Alemão ou Instrumental).
    * Sugestão Comparativa de Taxonomia (**Opção A: MIR Acústico** vs **Opção B: Catálogo & Paradas**).
  * [x] **Interface do Curador (UI de Alta Produtividade)**:
    * Visualização em waveform de áudio, marcadores visuais de CUE arrastáveis com recalculo dinâmico, atalhos de teclado para aprovação rápida de lotes e ajuste fino de tags.
    * Botões de reanálise rápida individual (⟳) por card funcional.
    * Acesso multi-dispositivo pela rede local (`0.0.0.0:5050`).
  * [x] **Exportador 1-Clique**:
    * Geração de pacote final com `catalog_manifest.json` e arquivos MP3 normalizados a -14 LUFS, com sincronização automática via API REST para o `imc-server`.
- [ ] **Evolução do Agendador para Smart Playlists no `imc-server` & `imc-audio-local`**:
  * [ ] Eliminar dependência exclusiva de pastas físicas fixas.
  * [ ] Grades Semanais com **Regras Dinâmicas e Filtros Multidimensionais** (ex: *Bloco da Tarde = [BPM 110-125] + [Mood Ensolarado] + [Vocal Feminino] + [Energia >= 3]*).
  * [ ] Preservação da **Regra de Ouro** (menor contagem de execuções) e **Separação de 45 min por Artista** sobre os conjuntos filtrados dinamicamente.
  * [ ] Manter o fluxo de segurança: Servidor armazena MP3 tratado; Player da Loja baixa e salva criptografado como `.imc`.


---

### 🖥️ A. IMC Server (Centralizador & Painel Administrativo)
- [ ] **Módulo de Login & Perfis de Acesso (RBAC)**: Criar perfis dedicados como **Gestor de Áudio / Gerente de Loja** (visualização limpa de relatórios, Proof-of-Play, dashboard e chamadas) e **Administrador / TI** (acesso a logs avançados, infraestrutura e terminal).
- [ ] **Telemetria Centralizada & Logs de Saúde**: Criar rotas na API para receber telemetria periódica dos players (status do player, métricas de execução e alertas).
- [ ] **Relatório Auditável de Execução (Proof-of-Play)**: Tela para o Gestor de Áudio emitir comprovantes oficiais de exibição de comerciais com gráficos e filtros por loja/período.
- [ ] **Limites em Comerciais**: Ajustar limites de inserções no cadastro de comerciais e adicionar avisos visuais de alerta para sobrecarga no bloco do dia.
- [ ] **Simulador de Linha do Tempo**: Criar tela de simulação no servidor para o operador visualizar a linha do tempo exata da grade antes de enviar às lojas.
- [ ] **Customização Visual por Empresa/Tenant**: Permitir suporte a temas/logos customizados por marca/empresa no painel.
- [ ] **Geolocalização (Lat/Long)**: Adicionar campos de latitude e longitude no cadastro de Lojas e Matrizes para mapas e busca de temperatura local.
- [ ] **Configuração de Volume Padrão por Categoria**: Permitir ajustar no servidor o volume padrão relativo de Comerciais, Eventos e Músicas.
- [ ] **Tag de Música Longa**: Identificar automaticamente faixas > 5 minutos (CUE IN/OUT) e adicionar toggle no cadastro de mídias e grades para bloquear/permitir músicas longas por bloco.
- [ ] **Forçar/Remover Selo SONIC**: Adicionar chave manual no painel do servidor para forçar ou remover a flag SONIC de qualquer mídia.
- [ ] **Central de Suporte Remoto**: Tela de suporte no servidor para receber conexões de túnel do player local e espelhar logs/diagnósticos em tempo real.

---

### 📻 B. IMC Audio Local (Player de Áudio & Playout)
- [ ] **Interface Limpa "Modo Loja / Kiosk"**: Reestruturar a interface local do player para ocultar/proteger os logs técnicos da equipe da loja, exibindo uma UI limpa focada em "Tocando Agora", capa, status simples e chamadas.
- [ ] **Envio Assíncrono de Telemetria (Store and Forward)**: Salvar execuções no SQLite local e descarregar no servidor central via API assim que a conexão estiver ativa.
- [ ] **Serviço Nativo do Windows**: Configurar o binário do `imc-audio-local` para rodar automaticamente como um Windows Service em background.
- [ ] **Síntese Local de Áudio (Text-To-Speech)**: Integrar gerador TTS local (Edge-TTS, Piper ou SAPI) para anúncios em texto convertidos instantaneamente em voz com fade da música.
- [ ] **Transição Inteligente de Áudio (Crossfade Específico)**:
  * Fade-in / Fade-out suave entre Músicas.
  * Corte direto (sem fade-out) para Comerciais, garantindo que 100% do áudio do spot seja ouvido sem cortes iniciais/finais.
- [ ] **Limpeza Tardia de Cache (Garbage Collector)**: Manter músicas removidas da grade em cache local por até 7 dias para reaproveitamento rápido caso reapareçam, limpando após o período.
- [ ] **Conexão de Suporte Reversa (Túnel)**: Botão no player local para abrir túnel seguro WebSocket até o servidor central permitindo manutenção remota sem abrir portas na loja.
- [ ] **Controles Avançados de MP3**: Adicionar reprodutor de teste/pre-escuta local para mídias no painel do player.

---

### 🔒 C. Segurança, Auditoria & Acesso
- [ ] **Autenticação Segura no Liquidsoap**: Proteger endpoints do harbor HTTP (`:7000`) com tokens HMAC/JWT rotativos.
- [ ] **Políticas Dinâmicas de Log de Execução**: Garantir integridade de registros de auditoria mesmo em caso de falha temporária de energia na loja.

---

### 🔄 D. Programação Musical, Clima & Conteúdo
- [ ] **Classificador Inteligente de Gênero, BPM & Energia (`imc-tagger`)**: Módulo de IA/DSP para ler arquivos MP3 e detectar automaticamente: gênero musical principal/secundário, BPM exato, humor/mood e nível de energia/danceability para catalogação automática do acervo na tabela `media_library`.
- [ ] **Ordem Musical Fora do Expediente**: Configurar regras alternativas de reprodução (músicas ambiente/relaxantes) quando a loja estiver fechada.
- [ ] **Hora Certa Automática**: Módulo para síntese e disparo da hora exata nos intervalos programados.
- [ ] **Temperatura Local**: Integração com API de clima por geolocalização para vincular e disparar vinhetas de temperatura ambiente.
- [ ] **Notícias & Ofertas Expressas**: Suporte a vinhetas automatizadas de notícias rápidas e criação de spots comerciais expressos no painel.

---

### 📱 E. Planejamento Mobile & Extensões
- [ ] **Aplicativo Mobile (Lojas)**: App para celular/tablet para funcionar como microfone sem fio da loja (locuções ao vivo) e disparo de chamadas pontuais de ofertas.

---

### 🐧 H. Distribuição Linux Customizada (IMC Appliance OS / Kiosk OS)
- [ ] **1. Base Minimalista & Arquitetura de Baixo Consumo**:
  * [ ] Baseada em **Ubuntu Server LTS Minimal** (ou Debian Minimal) para hardware x86_64 legado e mini-PCs.
  * [ ] Custo zero de licenciamento (100% Open Source / R$ 0,00).
  * [ ] Remoção de pacotes desnecessários (display managers pesados, Bluetooth, impressoras, telemetria padrão) para pegada < 250 MB de RAM.
  * [ ] Suporte a drivers de som universais (ALSA / PipeWire headless) sem servidor gráfico X11/Wayland para nós que operam como playout puro.
- [ ] **2. Blindagem contra Quedas de Energia (Imunidade a Corrupção de Disco)**:
  * [ ] Raiz do sistema de arquivos somente leitura (**Read-Only Root com OverlayFS**), permitindo que o operador da loja puxe o computador da tomada sem corromper o SO.
  * [ ] Partição de dados persistentes (`/data` ou `/var/imc`) isolada, montada com journaling estrito (`ext4` com flag `data=journal` ou `noatime`).
  * [ ] Logs direcionados para memória RAM temporária via `tmpfs` / `zram` para poupar o ciclo de vida de pendrives e SSDs de baixo custo.
- [ ] **3. Instalador 1-Clique via Pendrive (Kiosk Auto-Installer)**:
  * [ ] Imagem ISO customizada gerada via **Cubic** (Custom Ubuntu ISO Creator) ou **Ubuntu Subiquity / Cloud-Init Autoinstall**.
  * [ ] Processo de instalação automatizado ("plug & forget"): ao dar boot pelo pendrive em PCs legados, executa particionamento automático e instalação sem perguntas técnicas complexas.
  * [ ] Assistente de primeiro boot (TUI leve com `whiptail`/`dialog` ou tela de setup local via navegador web) para associar a Loja, definir Token do `imc-server` e testar saída de áudio.
- [ ] **4. Rede Mesh & Conectividade Remota Zero-Config**:
  * [ ] Cliente **WireGuard** ou **Tailscale / Headscale** integrado de fábrica.
  * [ ] Conexão segura bidirecional com o `imc-server` atravessando CGNAT e roteadores de lojas sem necessidade de redirecionamento de portas (port forwarding) ou IP estático.
  * [ ] Acesso SSH administrativo seguro restrito à interface de VPN para suporte técnico e telemetria remota.
- [ ] **5. Ciclo de Vida e Atualizações em Dupla Camada (Dual-Layer Updates)**:
  * [ ] **Camada de Sistema Operacional**: `unattended-upgrades` configurado para aplicação silenciosa de patches de segurança críticos do Ubuntu sem intervenção manual.
  * [ ] **Camada de Aplicação (`imc-server` / `imc-audio-local`)**: Unidade `systemd` com reinício automático em falha (`Restart=always`), health-checks de inicialização e rotinas de substituição atômica de binário via OTA com rollback automático.

---
*Última atualização: Setembro de 2026*


