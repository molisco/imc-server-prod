# Plano de Implementação: Redesign Responsivo do Web Player & Desconexão Inteligente

Este plano detalha a reformulação visual completa e moderna do **Web Player Corporativo** (`customer_player.html`), tornando-o 100% responsivo para smartphones, tablets e computadores, aliado à lógica de encerramento instantâneo de conexões em caso de perda de internet, travamento ou fechamento do navegador.

---

## 🎯 Objetivos

1. **⚡ Desconexão e Queda de Conexão em Tempo Real:**
   - Detectar perda de internet (`window.offline`), congelamento de buffer ou encerramento da aba/app móvel e derrubar imediatamente o socket de streaming no Icecast.
   - Configuração de `client-timeout` ultra-baixo (5s) e TCP KeepAlive no Icecast para eliminar ouvintes fantasmas caso o aparelho seja desligado ou perca o sinal de rede.

2. **📱 Redesign Completo & Responsivo (Mobile-First):**
   - Layout moderno estilo *Glassmorphism* com tema dark refinado, cards translúcidos, tipografia nítida e contrastes harmoniosos.
   - Menu de abas deslizável e adaptativo para telas pequenas (smartphones), médias (tablets) e grandes (desktops).
   - Controles de reprodução ergonômicos e *touch-friendly* (botões maiores e confortáveis para o toque dos dedos no celular).
   - Mini-player fixo / flutuante quando o operador estiver navegando nas abas de Placas, Avisos ou Histórico no celular.
   - Preservação de **100% das funcionalidades existentes** (Player, Troca de Estilos com contagem regressiva, Chamada de Placas IA, Chimes, Avisos Pré-Formatados, Histórico Recente, Grade de Blocos, Eventos do Dia, Catálogo de Músicas e Terminal de Logs).

---

## 🏗️ Proposta de Arquitetura e Mudanças

### 1. Engine de Conexão & Resiliência (`customer_player.html` / `icecast.xml`)

- **Eventos de Rede do Navegador:**
  - `window.addEventListener('offline')`: Dispara `stopAudioStream()`, exibe aviso visual no player e zera sockets.
  - `window.addEventListener('online')`: Tenta reconexão suave e atualiza status.
  - `window.addEventListener('pagehide')` e `beforeunload`: Envia `TCP FIN` de fechamento imediato.
  - `window.addEventListener('freeze')` (Page Lifecycle API mobile): Encerra streaming para não manter conexões presas em apps suspensos.
- **Watchdog Anti-Congelamento:**
  - Se o áudio parar de avançar o `currentTime` por mais de 4 segundos enquanto estiver com intenção de tocar, executa `stopAudioStream()` e reconecta de forma limpa.
- **Icecast Timeout:**
  - Ajuste de `client-timeout` para 5 segundos.

---

### 2. Design System & Interface Mobile/Desktop (`customer_player.html`)

- **Top Navbar:**
  - Header compacto com Logo, Nome da Loja / Matriz, Badge de Status da Rádio (`Online`/`Offline`) e botão Sair com confirmação rápida.
- **Navegação em Abas (Touch Tabs):**
  - Barra de abas com rolagem horizontal suave no celular (`overflow-x: auto` com *scroll-snap*) e botões em pílula (*Pills*) com ícones nítidos:
    - 📻 **Player**
    - 🚗 **Chamar Placa**
    - 🔔 **Chimes**
    - 📢 **Avisos Imediatos**
    - 📜 **Histórico Recente**
    - 📅 **Grade de Blocos**
    - ⏰ **Eventos Marcados**
    - 🎵 **Rotação & Acervo**
    - 🖥️ **Logs**
- **Aba Principal: Player & Sobreposição de Estilo:**
  - Arte de Capa com disco de vinil girando dinamicamente e anel de brilho (*ambient glow*).
  - Título da faixa e Artista em destaque com auto-scroll (*marquee*) se o texto for longo.
  - Barra de progresso estilizada com tempo decorrido e restante.
  - Botão Play/Pause grande centralizado com feedback tátil e visual.
  - Card de **Troca Rápida de Estilo** com seletor de estilo, pílulas de tempo (30m, 1h, 2h, 4h) e banner de contagem regressiva em tempo real.
- **Tabelas Responsivas:**
  - Tabelas de Grade, Histórico, Eventos e Acervo adaptadas para modo card/compacto no mobile, evitando cortes e barras de rolagem desnecessárias.

---

## 📋 Arquivos Afetados

| Componente | Arquivo | Ação |
| :--- | :--- | :--- |
| **Interface & Lógica Web** | `web/templates/customer_player.html` | **[MODIFY]** Redesign completo + watchdog + lifecycle |
| **Streaming Server** | `/etc/icecast.xml` (Docker) | **[MODIFY]** `client-timeout` de 5 segundos |

---

## 🧪 Plano de Verificação

1. **Teste de Responsividade:**
   - Visualizar em viewport Mobile (360px a 430px de largura - iPhone / Galaxy), Tablet (768px a 1024px - iPad) e Desktop (1920px).
   - Testar rolagem suave das abas e acessibilidade dos botões de toque.
2. **Teste de Desconexão Abrupta & Travamento:**
   - Simular modo offline no navegador e verificar se o socket no Icecast cai em segundos.
   - Simular fechamento de aba e logout e validar contagem de ouvintes no `/radios/status`.
3. **Teste de Funcionalidades:**
   - Testar Play/Pause, Skip, Troca Rápida de Estilo, Chamada de Placa, Chimes e Avisos Pré-formatados.
