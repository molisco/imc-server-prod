# Tarefas: Migração de Estilos por ID no Sincronismo

## 🌐 Componente: Servidor Central (imc-server)
- `[x]` Atualizar a struct `GridsPageData` e consulta SQL em `grid_handler.go` para usar `StyleInfo` (ID + Nome)
- `[x]` Atualizar o template `grids_audio.html` para guardar e associar os IDs nos dropdowns e no JSON final
- `[x]` Atualizar o template `grids_video.html` para fazer o mesmo com estilos de vídeo
- `[x]` Atualizar a lógica do `store_handler.go` na rota `/api/v1/sync` para suportar fallback UUID/Nome

## 📻 Componente: Player Local (imc-audio-local)
- `[x]` Atualizar `selectNextMusicTrack` em `scheduler.go` para aceitar UUIDs diretamente sem busca por nome
- `[x]` Ajustar a lógica de simulação em `GetDailyPlaylist` em `scheduler.go` para validar UUID e Nome
- `[x]` Validar a compilação e rodar testes locais do player
