# Plano de Implementação - Estúdio de Locução com Inteligência Artificial (ElevenLabs) no IMC Server

Desenvolvimento de um **Estúdio de Locução IA** integrado diretamente à API do **ElevenLabs** no `imc-server`. A ferramenta permitirá que o operador digite um texto para comercial ou aviso, escolha vozes clonadas/nativas da sua conta do ElevenLabs, ajuste a entonação/estabilidade, gere a locução em alta qualidade (MP3), pré-escute no navegador e salve instantaneamente como um **Spot Comercial** ou **Aviso/Vinheta** para distribuição automática às lojas.

---

## 🎯 Funcionalidades Propostas

1. **Gerenciamento de Chave API (ElevenLabs)**:
   - Suporte a variável de ambiente `ELEVENLABS_API_KEY` ou campo de configuração no banco de dados.
2. **Catálogo Dinâmico de Vozes (`GET /api/v1/elevenlabs/voices`)**:
   - Consulta em tempo real à conta do usuário no ElevenLabs exibindo vozes nativas, vozes clonadas da empresa, amostras de áudio para teste, idiomas e IDs.
3. **Estúdio de Geração e Ajustes de Locução (`POST /api/v1/elevenlabs/generate`)**:
   - Caixa de texto interativa com contador de caracteres e suporte a marcações de pausa (`<break time="1s" />`).
   - Seleção de modelo (ex: `eleven_multilingual_v2` ou `eleven_flash_v2_5`).
   - Controles de fine-tuning de locução:
     - **Estabilidade (Stability)**: Locução mais consistente vs. mais expressiva/emocional.
     - **Clareza e Similaridade (Clarity / Similarity Boost)**: Fidelidade da voz.
     - **Exagero de Estilo (Style Exaggeration)**: Entonação dramática/comercial.
     - **Speaker Boost**: Ganho e presença da voz.
4. **Player de Pré-Escuta & Integração Direta**:
   - Player com forma de onda no navegador para conferência antes de salvar.
   - Botão **📢 "Salvar como Spot Comercial"** (Cadastra diretamente na tabela de comerciais do servidor).
   - Botão **🔊 "Salvar como Aviso / Vinheta"** (Salvar no acervo de avisos pré-formatados).
5. **Esteira Automática SONIC**:
   - Passagem automática do áudio gerado pela esteira de normalização e adição da flag SONIC.

---

## 🏗️ Arquitetura dos Componentes

### 1. Backend (Go - `imc-server`)
- **[NEW] [`internal/handlers/elevenlabs_handler.go`](file:///P:/home/neoroute/imc-server/internal/handlers/elevenlabs_handler.go)**:
  - Client HTTP para comunicação com a API do ElevenLabs (`https://api.elevenlabs.io/v1`).
  - Handlers para listar vozes, gerar áudio em stream/buffer e converter o áudio em comercial/aviso oficial.
- **[MODIFY] [`cmd/server/main.go`](file:///P:/home/neoroute/imc-server/cmd/server/main.go)**:
  - Registro das rotas `/elevenlabs-studio`, `/api/v1/elevenlabs/voices`, `/api/v1/elevenlabs/generate` e `/api/v1/elevenlabs/save`.

### 2. Frontend (HTML/JS/CSS)
- **[NEW] [`web/templates/elevenlabs_studio.html`](file:///P:/home/neoroute/imc-server/web/templates/elevenlabs_studio.html)**:
  - Interface do Estúdio de Locução com layout moderno, seletor visual de vozes com avatares, controles deslizantes e player de áudio integrado.
- **[MODIFY] [`web/templates/dashboard.html`](file:///P:/home/neoroute/imc-server/web/templates/dashboard.html)**:
  - Adição do item **🎙️ Estúdio de Locução (IA)** no menu de navegação da plataforma.

---

## 📋 Plano de Verificação

### 1. Testes de Integração com API
- Testar listagem de vozes atreladas à API Key do usuário.
- Validar geração de locução em Português do Brasil com o modelo `eleven_multilingual_v2`.

### 2. Testes de Fluxo Completo
- Digitar um texto de teste (ex: *"Atenção clientes, oferta especial no setor de hortifrúti só hoje!"*).
- Gerar o áudio, pré-escutar e clicar em "Salvar como Comercial".
- Verificar se o arquivo MP3 foi salvo em `web/static/uploads/commercials` e cadastrado no banco PostgreSQL.
- Confirmar se o comercial aparece na lista de veiculação e sincroniza com os players das lojas (`imc-audio-local`).
