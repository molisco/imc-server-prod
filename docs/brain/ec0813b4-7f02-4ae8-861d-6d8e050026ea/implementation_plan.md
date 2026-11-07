# Plano de Implementação: Cadastro de Placas (Avisos de Voz & Veículos)

## Visão Geral
Transformar o antigo módulo de "Pacotes de Voz" no novo **Cadastro de Placas**, proporcionando um cadastro completo em formato de **Ficha de Conjunto de Áudios**, com tipificação/categoria, descrição detalhada de instruções de gravação/transcrição, escopo de distribuição com a flag **"Uso Global"** (ou seleção de lojas específicas da matriz) e gerenciamento dos arquivos de áudio pertencentes a cada placa.

---

## 💡 Mudanças Propostas

### 1. Banco de Dados (PostgreSQL)
- **Atualização na tabela `voice_packs` (agora representando o Cadastro de Placas):**
  - Adicionar coluna `description TEXT DEFAULT ''` (instruções e informações da placa).
  - Adicionar coluna `category VARCHAR(100) DEFAULT 'Estacionamento / Placas'` (tipificação do conjunto).
  - Adicionar coluna `is_global BOOLEAN DEFAULT false` (flag de Uso Global).
- **Criar tabela de associação `voice_pack_stores`:**
  - `voice_pack_id UUID REFERENCES voice_packs(id) ON DELETE CASCADE`
  - `store_id UUID REFERENCES stores(id) ON DELETE CASCADE`
  - Chave primária composta `(voice_pack_id, store_id)` para distribuição direcionada por loja.

---

### 2. Interface do Usuário (Templates & Navegação)
- **Renomeação global no menu lateral e cabeçalhos:**
  - Mudar de `🗣️ Pacotes de Voz` para `🏷️ Cadastro de Placas`.
- **Reformulação da Tela/Modal `web/templates/voice_packs.html` (Ficha de Cadastro de Placas):**
  - **Ficha Agrupada de Cadastro:**
    - Campo **Nome do Conjunto/Placa** (Ex: *Chamada de Veículos Estacionamento - Voz Feminina*).
    - Campo **Tipificação / Categoria** (Ex: *Aviso de Veículo / Placas*, *Chamada de Atendimento*, *Aviso Operacional*).
    - Campo **Descrição / Detalhes do Conjunto** (Textarea para informações de gravação, frases transcritas e observações).
    - Checkbox **Uso Global** (se marcado, o conjunto fica disponível para todas as lojas).
    - **Seleção de Lojas** (se "Uso Global" estiver desmarcado, permite selecionar lojas específicas da matriz).
  - **Visualização da Ficha & Gerenciamento de Áudios:**
    - Card em formato de ficha com contadores de arquivos, player de áudio `<audio>` para cada trecho, tamanhos e botões para adicionar/excluir arquivos do conjunto.
    - **Guia Visual de Gravação:** Seção informativa explicando como os arquivos devem ser gravados e nomeados para composição automática dos avisos no player.

---

### 3. Backend Go (`imc-server`)
- **Modelos (`internal/models/warning_assets.go`):**
  - Atualizar a struct `VoicePack` com `Description`, `Category`, `IsGlobal` e `TargetStoreIDs []string`.
- **Handlers (`internal/handlers/warning_asset_handler.go`):**
  - Atualizar `HandleVoicePacksPage` para carregar a lista de lojas da matriz e as lojas vinculadas a cada pacote.
  - Atualizar `HandleCreateVoicePack` para processar a ficha completa (nome, categoria, descrição, flag `is_global`, IDs das lojas selecionadas e upload múltiplo de arquivos).
  - Adicionar handler de edição/gerenciamento de arquivos individuais de uma placa (`HandleUpdateVoicePack` e `HandleDeleteVoicePackFile`).
- **Sincronismo com as Lojas (`internal/handlers/store_handler.go`):**
  - Atualizar a query do `SyncStoreData` para incluir conjuntos de placas com `is_global = true` ou que estejam vinculados à loja solicitante.

---

## 🧪 Plano de Verificação

### Testes Automatizados e de Banco
1. Executar migração SQL para adicionar colunas `description`, `category`, `is_global` e a tabela `voice_pack_stores`.
2. Verificar compilação do servidor Go (`go build ./cmd/server`).

### Teste Manual de Fluxo
1. Acessar [http://192.168.15.250:8080/dashboard/voice-packs](http://192.168.15.250:8080/dashboard/voice-packs).
2. Cadastrar uma nova placa com flag "Uso Global" preenchida, categoria e descrição.
3. Cadastrar uma placa sem "Uso Global" direcionada apenas para a loja `Shopping Cidade Sorocaba`.
4. Verificar se a sincronização da loja exibe os pacotes atribuídos corretamente.
