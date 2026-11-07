# Walkthrough - Implementações e Ajustes no IMC-Server

Este documento resume as correções e a nova funcionalidade implementada no backend centralizer (`imc-server`).

---

## 1. Nova Funcionalidade: Esteira de Masterização de Áudio e Selo SONIC

Implementamos a esteira universal de masterização de áudio para a biblioteca de músicas e de comerciais, introduzindo o selo de qualidade **SONIC** (Selo de Otimização e Nivelamento Comercial) e bloqueio de sincronização para mídias maiores de 5 minutos úteis.

### Alterações Realizadas

#### [Componente: Banco de Dados & Modelos]
* #### [MODIFY] [postgres.go](file:///p:/imc-server/internal/database/postgres.go)
  * Adicionadas migrações automáticas para inserir a coluna `is_sonic` (`BOOLEAN DEFAULT FALSE`) nas tabelas `music_library` e `commercials_and_events`.
* #### [MODIFY] [music_track.go](file:///p:/imc-server/internal/models/music_track.go) e [media..go](file:///p:/imc-server/internal/models/media..go)
  * Incluído o campo `IsSonic bool` com tags JSON e de banco nas structs `MusicTrack` e `CommercialsAndEvents`.

#### [Componente: Repositórios]
* #### [MODIFY] [music_track_repository.go](file:///p:/imc-server/internal/repository/music_track_repository.go) e [commercial_repository.go](file:///p:/imc-server/internal/repository/commercial_repository.go)
  * Atualizados comandos SQL (`INSERT`, `SELECT`, `UPDATE`) para carregar e salvar a flag `is_sonic`.

#### [Componente: Processador de Metadados (Core Engine)]
* #### [MODIFY] [metadata.go](file:///p:/imc-server/internal/utils/metadata.go)
  * **Fase 1 (IRR Check):** Análise rápida preliminar de ruído. Se o Índice de Risco de Ruído (IRR) passar de 50%, o upload é rejeitado com erro imediato.
  * **Fase 4 (Physical Masterization):** Transcodificação física utilizando FFmpeg com filtros de broadcast:
    `ffmpeg -y -i <in> -af "highpass=f=80,equalizer=f=3000:width_type=h:width=200:g=3,loudnorm=I=-14:LRA=7:TP=-1.5" -ac 1 -ar 44100 -b:a 128k <out>`
  * **Selo SONIC:** Certificação `IsSonic = true` concedida apenas se o IRR for aceitável e a duração útil (Cue Out - Cue In) for menor ou igual a 5 minutos (300 segundos).
  * **Fallback robusto:** Caso o arquivo não exista (como em testes unitários/ambientes sem ffmpeg), o sistema retorna os valores padrões com segurança.

#### [Componente: Sincronização & API]
* #### [MODIFY] [store_handler.go](file:///p:/imc-server/internal/handlers/store_handler.go)
  * Atualizado o endpoint de sincronização das lojas (`/api/v1/sync`) para enviar apenas músicas e comerciais que possuem o selo **SONIC** ativo (`is_sonic = true`) e duração útil <= 300 segundos.

#### [Componente: Web Frontend]
* #### [MODIFY] [tracks_table_partial.html](file:///p:/imc-server/web/templates/tracks_table_partial.html) e [commercials.html](file:///p:/imc-server/web/templates/commercials.html)
  * Inserida regra visual no template HTML para marcar em **vermelho** (`style="color: #ff4d4d !important;"`) as linhas de músicas ou comerciais cujo tempo útil (Cue Out - Cue In) seja maior de 5 minutos, servindo como alerta visual para o administrador.
* #### [MODIFY] [login.html](file:///p:/imc-server/web/templates/login.html)
  * Adicionada a marcação de versão `Versão v1.5.0 - SONIC` abaixo do botão de login para identificação imediata da versão implantada.

---

## 2. Verificação de Funcionamento

* **Compilação:** O binário compila perfeitamente localmente (`server.exe`).
* **Testes unitários e de integração:** Todos os testes das suítes de repositório, utilitários e handlers passaram com sucesso (`go test ./...` finalizado com OK).
* **Conexão com Banco Remoto:** A string de conexão fallback no código local foi atualizada para se conectar dinamicamente ao banco de dados PostgreSQL no servidor de desenvolvimento Linux (`192.168.15.250`).
* **Execução:** O servidor centralizer foi iniciado e está rodando em segundo plano localmente na porta `:8080` com as migrações aplicadas automaticamente.

---

## 3. Cópia Flexível de Grades de Áudio/Vídeo (Anterior)

* #### [Componente: Web Frontend]
  * Adicionado dropdown dinâmico nas grades de áudio e vídeo permitindo a replicação fácil da programação entre qualquer dia da semana.
