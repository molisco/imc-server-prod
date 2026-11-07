# 📻 Base de Conhecimento do Sistema IMC Rádio Corporativa
> **Documento Consolidado para Ingestão no Google NotebookLM**  
> **Versão:** 2.5 (Atualizada com Troca Rápida de Estilos, Bilhetagem e Estúdio de Locuções IA)  
> **Arquitetura:** SaaS Multi-Tenant + Edge Hybrid Playout (Go + Liquidsoap + PostgreSQL + SQLite)

---

## 1. Visão Geral da Arquitetura

O **IMC Rádio Corporativa** é uma plataforma profissional de áudio indoor e rádio indoor projetada para redes de varejo, shoppings, atacados, farmácias e franquias.

### Topologia Híbrida:
1. **IMC Server (Nuvem / Central):**
   - Backend em **Go (Golang)** de alta performance.
   - Banco de dados relacional **PostgreSQL 16**.
   - Painel administrativo web responsivo (Bootstrap 5, Vanilla JS, Web Audio API).
   - Gerenciamento de Tenants (Empresas/Matrizes), Lojas, Acervo Musical, Grades Semanais de Áudio, Campanhas Comerciais e Eventos.
   - Servidor Icecast2 integrado para streaming ao vivo na nuvem e monitoramento de ouvintes.
2. **IMC Audio Local (Player Edge na Loja / Docker):**
   - Engine autônomo escrito em **Go** rodando localmente na loja (ou em containers Docker dedicados por loja).
   - Banco de dados local ultraleve em **SQLite 3** (`local_cache.db`).
   - Servidor decodificador / descriptografador de mídia (Portas `5001..5005`).
   - **Liquidsoap Engine:** Playout de áudio profissional com crossfades, ducking, equalização, faders de volume e injeção de streams ao vivo (Portas Harbor `7001..7005`).
   - **Operação Offline-First:** Caso a internet caia, o player da loja continua tocando 100% da programação comercial e musical a partir do cache criptografado `.imc`.

---

## 2. Motor de Playout e Transições de Áudio

### A) Normalização de Volume e Proteção Acústica
- Todas as faixas do acervo passam por análise R128 no momento do upload no servidor central:
  - **LUFS Integrado:** Medição de sonoridade média.
  - **LRA (Loudness Range):** Faixa de dinâmica.
  - **True Peak & Risco de Ruído:** Prevenção de distorção harmônica.
- O engine de playout aplica ganho dinâmico (`CalculateAmplify`) garantindo que todas as músicas toquem no padrão uniforme de **-14 LUFS** sem oscilações bruscas no ambiente da loja.

### B) Transições entre Streaming Web e Acervo Local (Buffer Preload & Fade)
- Ao transicionar entre uma rádio web (Streaming Externo / Hotmix) e o acervo local de MP3:
  - O sistema conecta o stream em background e monitora o buffer até atingir 100% de preenchimento.
  - Executa um **crossfade suave de 2.0 segundos** diminuindo o volume da fonte anterior e elevando a nova fonte.
  - Previne estouros de áudio, silêncio e repetições em loop.

---

## 3. Motor de Grade Semanal, Comerciais e Eventos

### A) Estrutura da Grade Musical:
- **Weekly Master Grid:** Define o comportamento da rádio para cada dia da semana (Segunda a Domingo + Feriados).
- **Blocos Horários & Custom Styles:** Cada bloco define a duração musical (ex: 60 min) e o intervalo comercial (ex: 15 min), mapeando estilos específicos para faixas de horário.

### B) Distribuição de Comerciais e Ofertas:
- **Algoritmo de Rotação Equitativa:** Comerciais são distribuídos nos blocos planejados do dia (`planned_commercial_slots`), garantindo que todas as marcas alcancem sua meta diária de inserções sem repetições consecutivas.
- **Eventos com Horário Marcado:**
  - **Modo HARD:** Interrompe a música no exato segundo agendado (ideal para comunicados de abertura/fechamento de loja ou vinhetas pontuais).
  - **Modo SOFT:** Aguarda a música atual terminar naturalmente antes de disparar o evento.

### C) Sistema de Chimes / Gongos:
- Sinais sonoros institucionais disparados antes de anúncios ou chamadas de veículos no estacionamento (`TriggerVehicleWarning`).

---

## 4. Troca Rápida de Estilos Musicais com Temporizador (Manual Override)

### A) Regra de Negócio:
- Permite que o gerente da loja sobreponha a programação da grade semanal temporariamente (ex: colocar estilo *Pop Rock*, *MPB* ou uma *Rádio Streaming específica*) por um período pré-definido (de 15 minutos até 6 horas).
- **Seleção de Estilos Permitidos:** O dropdown lista apenas os estilos que compõem a grade da loja + os estilos extras previamente autorizados no cadastro da Grade no painel.

### B) Prioridade Estrita no Playout:
- Enquanto o temporizador estiver ativo, o scheduler prioriza **100% exclusivamente** o estilo selecionado, preservando normalmente a entrada dos comerciais e eventos nos seus horários.
- Ao expirar o tempo (ou ao clicar em *Voltar à Grade*), a rádio retorna automaticamente para a programação semanal.

### C) Ajuste Suave sem Interrupção (Same-Style Extension):
- Se o operador selecionar o **mesmo estilo que já está tocando** apenas para estender ou encurtar a duração (ex: de 30 min para 2 horas), o sistema **não corta a música nem reaciona fades**. O áudio continua tocando ininterruptamente e apenas o temporizador é atualizado.

### D) Rastreabilidade no Histórico Recente:
- Toda troca manual, extensão de tempo e cancelamento/retorno à grade gera um registro no **Histórico Recente** com selos visuais (`🎛️ Troca Manual` e `🟢 Grade Restaurada`).

---

## 5. Módulo: Gerador Automático de Locuções (Estúdio IA)

### A) Propósito:
Central unificada para criação de spots de rádio (Ofertas, Institucionais, Avisos Avulsos e Placas) com Inteligência Artificial em menos de 2 minutos.

### B) Pipeline de Áudio:
1. **Redação:** Texto livre ou otimizado com IA (Copywriting de Varejo).
2. **Locução:** Síntese neural via ElevenLabs API com catálogo de vozes homologadas.
3. **Mixagem com Trilha Royalty-Free:** Mixagem automática via `ffmpeg` com **Ducking** (a trilha abaixa para ~15% de volume enquanto o locutor fala e sobe no final).
4. **Gongo Dinâmico:** Não é queimado dentro do MP3; o playout executa o gongo dinamicamente.

### C) Franquia por Loja & Bilhetagem:
- **Herança da Matriz:** Cada Matriz define a quota default (ex: 20 spots ou 30.000 caracteres/mês), herdada pelas filiais e customizável por loja.
- **Zero Desperdício de Créditos:** Amostras de vozes e trilhas de fundo ficam em cache local no servidor (Custo R$ 0,00 para navegação/preview). A API da ElevenLabs só é acionada no clique final de renderização.
- **Tabela de Bilhetagem (`ai_spot_billing_logs`):** Auditoria com tenant, loja, usuário, caracteres consumidos, créditos ElevenLabs e custo estimado.

---

## 6. Dicionário de APIs Principais

| Endpoint | Método | Descrição |
| :--- | :--- | :--- |
| `/api/player/{id}/now-playing` | `GET` | Metadados em tempo real (música atual, progresso, tipo, override ativo). Polling a cada 3s. |
| `/api/player/{id}/diagnostics` | `GET` | Diagnóstico completo: histórico recente, blocos comerciais, eventos e estilos autorizados. |
| `/api/player/{id}/manual-override` | `POST` | Aplica sobreposição de estilo musical por `target_id` e `duration_minutes`. |
| `/api/player/{id}/manual-override/cancel` | `POST` | Cancela sobreposição e restaura grade semanal imediatamente. |
| `/api/v1/sync` | `GET` | Sincronismo do player da loja com servidor central (grade, estilos, campanhas, metas). |
| `/api/player/{id}/warning/vehicle` | `POST` | Dispara comunicado imediato de placa de veículo com gongo e síntese de voz. |

---

## 7. Esquema de Banco de Dados (Resumo)

### PostgreSQL (Servidor Central):
- `tenants` (Empresas/Matrizes, quotas default de spots/caracteres).
- `stores` (Lojas, quotas individuais, portas harbor/decrypter, status online).
- `weekly_master_grids` (Grades semanais, `allowed_manual_styles_json`).
- `music_styles` (Estilos MP3 e Rádios Streaming).
- `music_library` (Metadados das faixas, análise R128 LUFS/LRA/Risco).
- `commercials_and_events` (Comerciais, institucionais, eventos, metas diárias, datas de validade).
- `ai_spot_billing_logs` (Bilhetes de consumo ElevenLabs por loja/spot).

### SQLite 3 (Player Local na Loja):
- `media_library` (Faixas baixadas localmente com caminho `.imc`).
- `commercials_and_events` (Comerciais e eventos com status de download).
- `planned_commercial_slots` (Agenda diária calculada de blocos comerciais).
- `play_history` (Histórico de reprodução, músicas, comerciais e trocas manuais).
- `allowed_manual_styles` (Estilos autorizados para sobreposição na loja).
- `manual_override` (Estado de persistência da sobreposição temporária).

---

## 8. Matriz de Templates Especializados 2×3 e Injeção de Áudio

### A) Separação Estrita de Ambientes e Fontes:
- O sistema elimina scripts monolíticos de Liquidsoap e utiliza uma matriz especializada de templates:
  1. `server_acervo.liq.tmpl`: Servidor / Docker + 100% MP3 local -> Icecast/HLS.
  2. `server_streaming.liq.tmpl`: Servidor / Docker + 100% Web Rádio ao vivo -> Icecast/HLS com fader dinâmico.
  3. `server_hybrid.liq.tmpl`: Servidor / Docker + Misto Acervo + Web Rádio com chaveamento dinâmico.
  4. `local_acervo.liq.tmpl`: Loja física + 100% Acervo `.imc` offline -> Placa de som física.
  5. `local_streaming.liq.tmpl`: Loja física + 100% Web Rádio -> Placa de som física com injeção local.
  6. `local_hybrid.liq.tmpl`: Loja física + Misto Acervo + Web Rádio -> Placa de som física.

### B) Comutação Dinâmica de Streaming (`source.dynamic` & `self_sync=false`):
- Transição suave entre rádios online sem encerramento de processos ou quebra de pipes (`EPIPE`) no Icecast/HLS:
  - Fade-out de 1.5s -> Instanciação de `input.http(self_sync=false, nova_url)` -> Fade-in de 1.5s com metadados sincronizados.

### C) Injeção de Comerciais e Avisos Sonoros:
- O agendador do Go envia URIs anotadas (`annotate:id="...",type="commercial",liq_amplify="1.51":/app/midias/...`) para o endpoint Harbor `/push_commercial`.
- O Liquidsoap insere a faixa na fila prioritária `commercials_queue` com ganho de volume de +20%, sobrepondo o streaming/música e retornando automaticamente ao término.
- Avisos instantâneos e veículos utilizam ducking suave (`smooth_add(p=0.03)`), diminuindo a música de fundo para 3% enquanto o aviso toca.

---

## 9. Harvester e Gestão de Licenças FMA (Free Music Archive)

### A) Propósito e Conformidade Legal:
- Coleta automatizada de acervo musical livre de direitos conexos (Royalty-Free / Creative Commons) com comprovação jurídica indelével (Proof-of-License).
- **Filtro Estrito contra CC0 / Domínio Público (`--exclude-cc0`):** Exclui faixas CC0 ou sem licença comercial segura, preservando exclusivamente `CC BY`, `CC BY-SA` e `CC BY-ND`.

### B) Motor de Captura e Comprovantes Visuais 1:1:
- O harvester (`scripts/fma_harvester.py`) gera automaticamente o par de arquivos para cada faixa baixada:
  1. **Comprovante Visual (`.png`):** Captura em resolução Retina de alta fidelidade do modal oficial de download do FMA (incluindo o aviso em amarelo, o checkmark verde de uso comercial, botão azul *"I understand and agree"* e o card de assinatura Tribe of Noise PRO).
  2. **Certificado Estruturado (`.json`):** Metadados completos da faixa (Track ID, Title, Artist, License Code, License URL, Commercial Allowed, Page URL, Download URL, Sha256, Timestamp UTC).
- **Renderização Ultrarrápida com CSS Cacheado:** Utiliza Chromium headless local com timeout de segurança e folhas de estilo locais (`scripts/assets/fma_styles.css`), gerando a imagem de prova em ~1.5 segundos sem travamentos ou dependência de rede externa.

---

## 10. Arquitetura de Atualização Automática Zero-Touch (OTA) do imc-audio-local

### A) Orquestração Centralizada no Servidor:
- O `imc-server` atua como Release Manager:
  - Endpoint `/api/v1/terminal/updates/check` (polled a cada 15 min ou disparado via push WebSocket).
  - Distribui pacotes binários assinados com hash SHA-256 (`imc-audio-local-win64.zip`, `imc-audio-local-linux64.tar.gz`).

### B) Execução no Player Local (Windows & Linux):
- **Windows (Service / Task Scheduler):** Aplica a estratégia de renomeação de binário em execução (`imc-audio-local.exe` -> `imc-audio-local.old.exe`), extrai o novo executável e dispara o reinício do serviço.
- **Linux (Systemd Daemon):** Descompacta a nova versão e realiza o swap atômico do binário / link simbólico com `systemctl restart imc-audio-local`.
- **Watchdog & Rollback de Segurança:**
  - O daemon inicia um timer de verificação de 15 segundos.
  - Se o novo executável inicializar o banco local (`local_cache.db`), carregar a grade e responder ao health-check com sucesso, confirma o update e apaga o binário `.old`.
  - Se ocorrer falha de boot ou crash, o daemon desfaz a alteração imediatamente (rollback para a versão anterior) e notifica o `imc-server` via telemetria de erro.

---

## 11. Curadoria Musical Multidimensional & Ingestão Inteligente (IMC Music Studio)

### A) Taxonomia de Curadoria & Ortogonalidade Vocal:
- **"Instrumental" é Perfil Vocal, não Gênero:** A classificação separa estritamente o formato de execução (`Instrumental Puro`, `Vocal Feminino`, `Vocal Masculino`, `Dueto`, `Chopped`) dos gêneros musicais.
- **Dimensões Ortogonais de Curadoria:**
  1. `Macro-Gênero & Subgêneros`: ex: Pop (Synthpop, Indie Pop, Nu-Disco), Dance/Eletrônica (Deep House, Tropical House).
  2. `Perfil Vocal`: Instrumental vs. Vocais cantados.
  3. `BPM & Energia`: Andamento numérico exato e intensidade de 1 a 5.
  4. `Moods / Climas`: Tags múltiplas (Ensolarado, Elegante, Relaxante, Urbano, Descontraído, Vibrante).
  5. `Fit de Varejo / Segmento`: Moda Jovem, Cafeteria, Farmácia, Supermercado, Academia.
  6. `Dayparting`: Manhã, Almoço, Tarde de Vendas, Noite.

### B) Isolamento de Processamento (IMC Music Studio - Standalone / Offline):
- O processamento pesado de áudio (cálculo de BPM via `librosa`/`aubio`, detecção vocal, análise R128 a -14 LUFS e cálculo de pontos de corte `cue_in`/`cue_out` a -35dB) é retirado do servidor central e isolado no software de curadoria desktop.
- **Ciclo de Armazenamento e Criptografia:**
  1. **Estúdio (PC do Curador):** Processa MP3s em lote $\rightarrow$ Exporta MP3 tratado a -14 LUFS + `catalog_manifest.json`.
  2. **Servidor (`imc-server`):** Armazena o MP3 tratado e metadados no PostgreSQL com zero sobrecarga de CPU de conversão, permitindo pré-escuta e Web Player nativos.
  3. **Player da Loja (`imc-audio-local`):** Baixa o áudio pela API e grava no HD local como `.imc` criptografado para proteção antipirataria.
- **Comerciais no Painel Web:** Comerciais e avisos locais continuam gerenciados diretamente no `imc-server` (tempo de CPU < 200ms por spot) para garantir a agilidade operacional do lojista.

---

## 12. Melhorias Recentes de Engenharia Acústica (IMC Music Studio V2)

### A) Duração Efetiva & Validação para Som Ambiente (2 a 5 minutos)
- A duração musical considerada pelo sistema é calculada a partir dos pontos de transição de energia:
  $$\text{effective\_duration\_seconds} = \max(0.0, \text{cue\_out} - \text{cue\_in})$$
- Faixas com duração útil entre 120s e 300s (2 a 5 min) são validadas como ideais para programação contínua de loja. Ajustes manuais de marcadores no canvas recalculam instantaneamente a duração útil e o badge visual.

### B) Preenchimento Sonoro e Aptidão no Sinal Útil
- O `sound_fullness_score` (0 a 100) e a `ambience_fitness` (`Apto`, `Atenção`, `Inapto`) são avaliados estritamente sobre a fatia útil entre `cue_in` e `cue_out`, eliminando silêncios e ruídos de fade das pontas do arquivo bruto.

### C) Classificação Vocal Multi-Janela & Ortogonalidade de Idioma
- Para evitar falsos positivos de `Instrumental Puro` decorrentes de introduções longas, o motor analisa 3 janelas dinâmicas (15%, 45% e 70% do tempo), correlacionando crista de formantes (900 Hz a 3200 Hz) e rastreamento melódico contínuo de pitch ($f_0$ via YIN).
- Se a faixa possuir canto, o idioma é identificado via NLP de metadados e priors de gênero; se for `Instrumental Puro`, o idioma é fixado como `Instrumental`.

### D) Sugestão Comparativa de Taxonomia (Opção A vs. Opção B)
- O curador pode escolher entre a **Opção A** (Recomendação Acústica MIR por análise tímbrica de áudio) ou a **Opção B** (Catálogo Global & Charts por paradas históricas e MusicBrainz), com adoção em 1 clique e atalhos de reanálise individual por card.

### E) Registro Persistente de BPM para o Smart Scheduler do Player
- O andamento rítmico (BPM) medido é gravado no banco SQLite (`studio_catalog.db`), no `audio_spec` e no manifesto exportado (`catalog_manifest.json`), preparando o player de loja para filtragem por horário do dia (*dayparting*), transições suaves com *BPM matching* e adaptação por segmento de varejo.

### F) Acesso e Operação em Rede Local
- O servidor web do estúdio inicializa escutando em `0.0.0.0:5050`, permitindo que outros operadores na rede acessem a interface gráfica simultaneamente via IP da máquina.




