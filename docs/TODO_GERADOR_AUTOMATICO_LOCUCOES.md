# 🎙️ Especificação & TODO: Gerador Automático de Locuções

> **Módulo:** Gerador Automático de Locuções (Estúdio IA de Spots, Ofertas e Institucionais)  
> **Status:** Planejado / TODO  
> **Sistemas Envolvidos:** `imc-server` (Central / Web UI) e `imc-audio-local` (Engine de Playout Edge)

---

## 📌 1. Visão Geral do Módulo

O **Gerador Automático de Locuções** é uma central unificada de produção de áudio com inteligência artificial integrada ao sistema de rádio corporativa. Ele permite que gerentes e operadores de lojas criem anúncios profissionais de rádio (Ofertas, Institucionais, Avisos Avulsos e Chamadas de Placas) em menos de 2 minutos, sem necessidade de softwares externos de edição.

O módulo inclui:
- **Redação Otimizada por IA:** Melhoria de texto com gatilhos de vendas para varejo (Copywriting).
- **Locução com Vozes Fixas:** Síntese neural via ElevenLabs API com vozes padrão de estúdio.
- **Mixagem Automática de Trilha (Ducking):** Músicas de fundo royalty-free mixadas automaticamente via `ffmpeg` (a música abaixa quando o locutor fala).
- **Controle de Franquias por Loja:** Quota mensal configurável na Matriz e herdada pelas filiais.
- **Bilhetagem & Auditoria Detalhada:** Rastreabilidade total de créditos e caracteres consumidos na ElevenLabs.
- **Proteção de Créditos com Cache Local:** Amostras de vozes e trilhas ouvidas 100% no servidor local (Custo R$ 0,00 para navegação).

---

## 🎛️ 2. Tipos de Conteúdos Suportados

| Categoria | Descrição | Trilha Musical | Gongo / Chime | Destino no Playout |
| :--- | :--- | :--- | :--- | :--- |
| **🛒 Ofertas / Promoções** | Preços e liquidações de varejo. | Trilha animada / pop com ducking. | Opcional dinâmico pelo player. | Blocos comerciais com meta diária e data de validade. |
| **🏢 Institucionais** | Mensagens de marca, regras e cartões. | Trilha corporativa / acústica suave. | Opcional dinâmico pelo player. | Blocos comerciais institucionais. |
| **📢 Avisos Avulsos** | Comunicados operacionais e avisos gerais. | Voz limpa (Dry Voice) sem trilha. | Gongo dinâmico executado pelo player. | Horários agendados ou fila imediata. |
| **🚗 Chamadas de Placa** | Chamadas de veículos no estacionamento. | Voz limpa e pausada. | Gongo de estacionamento dinâmico. | Injeção imediata no ar (`TriggerVehicleWarning`). |

> **Nota de Arquitetura:** O sinal sonoro (Gongo/Chime) **não é embutido no MP3**. O engine de playout executa o gongo dinamicamente antes do áudio, economizando caracteres da API e garantindo flexibilidade.

---

## 💰 3. Estrutura de Franquia, Quota e Bilhetagem

### A) Franquia por Loja (Herança da Matriz)
- **Na Matriz (`tenants`):** Campo `default_monthly_spot_quota` (ex: 20 spots ou 30.000 caracteres/mês) e dia de renovação do ciclo.
- **Nas Lojas (`stores`):** Campos `monthly_spot_quota`, `spots_used_current_cycle` e `quota_reset_day`.
- **Validação de Saldo:** Antes de renderizar, o sistema verifica se a loja possui saldo disponível. Caso esgotado, exibe aviso de limite com opção de recarga.

### B) Bilhetagem & Auditoria (`ai_spot_billing_logs`)
Cada arquivo gerado gera um log imutável contendo:
- `id` (UUID)
- `tenant_id` e `store_id`
- `user_id` e `user_name`
- `content_type` (`OFFER`, `INSTITUTIONAL`, `WARNING`, `VEHICLE_PLATE`)
- `text_prompt` (Texto enviado para locução)
- `character_count` (Quantidade de caracteres faturáveis)
- `voice_id` e `voice_name`
- `elevenlabs_credits_used`
- `estimated_cost_usd` / `estimated_cost_brl`
- `background_track_id` (Se houve mixagem)
- `output_file_name` e `output_file_path`
- `duration_seconds`
- `created_at`

### C) Zero Desperdício de Créditos (Previews Locais)
- **Amostras de Vozes:** Arquivos MP3 estáticos pré-gravados para cada locutor servidos diretamente pelo servidor web. O cliente escuta quantas vezes quiser com **0 chamadas à API**.
- **Trilhas Royalty-Free:** Biblioteca de MP3 locais armazenada em `/app/midias/spot_trilhas/`. Preview instantâneo e gratuito.
- **Consumo Único:** A ElevenLabs só é chamada uma única vez ao clicar em *"Renderizar Spot Final"*.

---

## 🏗️ 4. Modelagem de Dados (PostgreSQL)

```sql
-- 1. Quotas na Matriz e Lojas
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS default_monthly_spot_quota INT DEFAULT 20;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS default_monthly_char_quota INT DEFAULT 30000;

ALTER TABLE stores ADD COLUMN IF NOT EXISTS monthly_spot_quota INT DEFAULT 20;
ALTER TABLE stores ADD COLUMN IF NOT EXISTS monthly_char_quota INT DEFAULT 30000;
ALTER TABLE stores ADD COLUMN IF NOT EXISTS spots_used_current_cycle INT DEFAULT 0;
ALTER TABLE stores ADD COLUMN IF NOT EXISTS chars_used_current_cycle INT DEFAULT 0;
ALTER TABLE stores ADD COLUMN IF NOT EXISTS quota_cycle_start DATE DEFAULT CURRENT_DATE;

-- 2. Banco de Vozes Pré-Homologadas
CREATE TABLE IF NOT EXISTS ai_voices (
    id TEXT PRIMARY KEY, -- ElevenLabs Voice ID
    name TEXT NOT NULL,
    gender TEXT NOT NULL, -- MASCULINO / FEMININO
    style_category TEXT NOT NULL, -- VAREJO, INSTITUCIONAL, ELEGANTE, JOVEM
    sample_file_url TEXT NOT NULL, -- URL do MP3 de demonstração local
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 3. Biblioteca de Trilhas de Fundo Royalty-Free
CREATE TABLE IF NOT EXISTS spot_background_tracks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    genre TEXT NOT NULL, -- VAREJO, POP, ACUSTICO, ELETRONICO, CORPORATIVO
    file_path TEXT NOT NULL,
    preview_url TEXT NOT NULL,
    default_ducking_volume FLOAT DEFAULT 0.15,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 4. Tabela de Bilhetagem & Auditoria
CREATE TABLE IF NOT EXISTS ai_spot_billing_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),
    store_id UUID REFERENCES stores(id),
    user_id UUID REFERENCES users(id),
    user_name TEXT NOT NULL,
    content_type TEXT NOT NULL,
    text_content TEXT NOT NULL,
    character_count INT NOT NULL,
    voice_id TEXT NOT NULL,
    voice_name TEXT NOT NULL,
    elevenlabs_credits_used INT NOT NULL,
    estimated_cost_brl NUMERIC(10, 4) DEFAULT 0.0,
    background_track_id UUID REFERENCES spot_background_tracks(id),
    output_file_name TEXT NOT NULL,
    duration_seconds INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

---

## 📝 5. TODO de Implementação

- [ ] **Fase 1: Infraestrutura de Banco e Migrations**
  - [ ] Executar migrations de quotas em `tenants` e `stores`.
  - [ ] Criar tabelas `ai_voices`, `spot_background_tracks` e `ai_spot_billing_logs`.
  - [ ] Popular banco inicial com as 6 vozes padrão e carregar 10 trilhas royalty-free.

- [ ] **Fase 2: Motor de Geração e Mixagem (`imc-server`)**
  - [ ] Implementar cliente HTTP para ElevenLabs TTS (`POST /v1/text-to-speech/{voice_id}`).
  - [ ] Implementar rotina de mixagem `ffmpeg` com filtro `amix` e `sidechaincompress` (ducking automático).
  - [ ] Implementar normalização de volume para padrão da rádio (-14 LUFS).
  - [ ] Implementar integrador direto que salva o arquivo gerado na tabela `commercials_and_events`.

- [ ] **Fase 3: Módulo de Bilhetagem e Validação de Quota**
  - [ ] Criar middleware/serviço que valida saldo da loja antes da geração.
  - [ ] Gravar bilhete detalhado em `ai_spot_billing_logs` após sucesso da geração.
  - [ ] Criar rotina diária de renovação de ciclo de franquia.

- [ ] **Fase 4: Interface Visual (UI Web)**
  - [ ] Criar página do **Estúdio de Locuções** no painel administrativo e no player web da loja.
  - [ ] Assistente em 4 passos: Escolha do Tipo -> Texto (com botão de Copy IA) -> Locutor -> Trilha.
  - [ ] Widget de player para escutar as amostras locais (0 créditos) e o resultado final mixado.
  - [ ] Painel Master de auditoria com extrato de consumo e saldo da conta ElevenLabs.
