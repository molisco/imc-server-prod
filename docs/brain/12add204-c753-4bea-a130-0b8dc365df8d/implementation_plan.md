# Plano de Implementação: Sincronismo por ID de Estilos no Servidor (imc-server)

Este plano detalha a migração da associação de estilos musicais e de vídeo nas grades horárias do servidor para utilizar **IDs únicos (UUIDs)** em vez de **nomes textuais**. Isso garante que alterações de nome nos estilos não quebrem o sincronismo das lojas locais.

---

## 📌 Visão Geral das Alterações

1. **Camada de Visão (Templates HTML - `grids_audio.html` e `grids_video.html`):**
   * Modificar o array javascript `stylesList` para ser composto por objetos `{ id, name }` em vez de strings brutas.
   * Modificar os controles de exibição (badges e caixas de seleção) para gravar e associar o ID do estilo no JSON final de configurações (`custom_styles_json`).
   * Adicionar lógica de fallback para exibir corretamente o nome de estilos antigos (retrocompatibilidade com nomes textuais legados).

2. **Controlador do Portal (`grid_handler.go`):**
   * Criar a estrutura `StyleInfo` para conter `ID` e `Name`.
   * Atualizar a consulta de estilos para selecionar `id` e `name` (tabelas `music_styles` e `video_styles`).
   * Passar a lista estruturada no campo `Styles` para a renderização do template.

3. **API de Sincronismo (`store_handler.go`):**
   * Atualizar o parseador do campo `custom_styles_json` na rota `/api/v1/sync`.
   * Verificar se cada item associado é um UUID válido:
     * **Se for UUID válido:** Adiciona diretamente na lista de IDs programados (`programmedStyleIDs`), eliminando a necessidade de consulta por nome.
     * **Se for texto legado (Nome):** Adiciona à lista de nomes para busca reversa no banco, garantindo que grades antigas continuem funcionando sem quebra.

---

## 🛠️ Modificações Propostas

### 🌐 Componente: Servidor Central (P:\home\neoroute\imc-server)

#### [MODIFY] [grid_handler.go](file:///P:/home/neoroute/imc-server/internal/handlers/grid_handler.go)
* Alterar o tipo do campo `Styles` na struct `GridsPageData` de `[]string` para `[]StyleInfo`.
* Atualizar a função `renderGridsPage` para buscar `id` e `name` do banco de dados e popular a lista.

#### [MODIFY] [grids_audio.html](file:///P:/home/neoroute/imc-server/web/templates/grids_audio.html) e [grids_video.html](file:///P:/home/neoroute/imc-server/web/templates/grids_video.html)
* Ajustar a renderização do `stylesList` no bloco `<script>`.
* Atualizar o mapeamento de badges de visualização de estilos associados aos blocos para buscar o nome a partir do ID.
* Modificar a construção dos dropdowns de seleção para usar `style.id` nos checkboxes.

#### [MODIFY] [store_handler.go](file:///P:/home/neoroute/imc-server/internal/handlers/store_handler.go)
* Alterar a leitura de `custom_styles_json` no endpoint `/api/v1/sync` para suportar tanto UUIDs quanto nomes legados via parse de UUID.

---

### 📻 Componente: Player Local (c:\imc-audio-local)

#### [MODIFY] [scheduler.go](file:///c:/imc-audio-local/scheduler.go)
* **Função `selectNextMusicTrack` (Resolução de Estilos Ativos):**
  * Atualizar o bloco que busca os IDs dos estilos ativos ([L945-L963](file:///c:/imc-audio-local/scheduler.go#L945-L963)) para aceitar tanto UUIDs quanto nomes de estilo legados. Se um item no JSON for um UUID válido, usá-lo diretamente, ignorando a busca reversa por nome (o que evita a quebra do sincronismo ao renomear estilos).
* **Função `GetDailyPlaylist` (Simulação de Escala):**
  * Corrigir a comparação de estilos na simulação ([L2300-L2305](file:///c:/imc-audio-local/scheduler.go#L2300-L2305)) para buscar tanto pelo ID do estilo (`StyleID` da faixa) quanto pelo nome do estilo (`StyleName` da faixa), garantindo a fidelidade da simulação tanto para chaves novas (IDs) quanto legadas (nomes).

---

## 🧪 Plano de Verificação

### Testes de Sincronismo
1. Salvar uma grade no painel utilizando o novo formato baseado em IDs.
2. Alterar o nome de um estilo na tabela `music_styles` (ex: de `"0 - MPB"` para `"Z - MPB Renovado"`).
3. Realizar a requisição `/api/v1/sync` e garantir que o estilo e suas faixas continuem sendo retornados corretamente, provando que a associação por ID funciona.
4. Testar o sincronismo com uma grade antiga (que use nomes textuais no JSON) e garantir que a retrocompatibilidade continue funcionando.
