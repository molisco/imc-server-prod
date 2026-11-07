# Checklist de Implementação - Logs e Visualizações do Dashboard

- [x] Desenvolver a visualização de rodízio de músicas na biblioteca
  - [x] Criar endpoint JSON retornando contagem de plays (48h) e último play das músicas
  - [x] Adicionar nova aba "Rotação de Músicas" no painel web local
  - [x] Renderizar tabela de rodízio de músicas no frontend
- [x] Desenvolver a busca de último play de comerciais no histórico absoluto
  - [x] Modificar query de comerciais para buscar último play na tabela `play_history`
  - [x] Atualizar coluna "Último Play" na tabela de comerciais do painel
- [x] Implementar o Log Centralizado e Cronológico
  - [x] Configurar gravação de logs Go no arquivo `client_player.log`
  - [x] Criar endpoint `/api/centralized-logs` para ler, mesclar e ordenar cronologicamente logs do Go e do Liquidsoap
  - [x] Atualizar tela de logs no painel web com colorização e prefixos `[PLAYER]` / `[ENGINE]`
  - [x] Adicionar aba dedicada "Logs do Sistema" para visualização limpa e de tamanho expandido (600px) dos logs centralizados
- [x] Compilar, executar e testar o player com os logs unificados em execução
  - [x] Executar build do player local (`go build`)
  - [x] Iniciar executável `imc-audio-local.exe` em segundo plano
  - [x] Validar respostas da API do painel e logs centralizados
- [x] Customizar a tabela de Execução Real (Hoje) com colunas de áudio e compactar layout
  - [x] Reduzir tamanho de fonte para `0.8rem` e padding de células
  - [x] Buscar dados de Cue In/Out, Loudness (LUFS/LRA) e risco de ruído da biblioteca/comerciais no DB
  - [x] Formatar Cue In/Out em minutos e milissegundos (`MM:SS.mmm`)
  - [x] Exibir crachás coloridos de notas e porcentagem de ruído
  - [x] Compilar, reiniciar processo local e sincronizar com repositório Git (Push origin main)

