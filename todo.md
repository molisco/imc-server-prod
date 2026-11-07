# Lista de Tarefas - Projeto IMC

Este é o local centralizado de gerenciamento de tarefas para os projetos **IMC Server** (Servidor) e **IMC Local** (Player/Cliente).

---

## 📌 Links Rápidos
- **IMC Local**: [c:/imc-audio-local](file:///c:/imc-audio-local)
- **IMC Server**: [P:/imc-server](file:///P:/imc-server)

---

## 📋 Tarefas em Andamento / Planejadas

### 🖥️ IMC Server (Servidor - P:\imc-server)
- [x] **Ajustar o visual** do Dashboard e Sidebars com navegação categorizada e intuitiva *(Item 1)*
- [ ] Criar um **simulado no servidor** para que o operador consiga visualizar a linha do tempo/execução *(Item 5)*
- [x] Ajustar a questão de **usuários e permissões** no imc-server (RBAC, CRUD completo, perfis e isolamento multi-tenant) *(Item 6)*
- [ ] Ajustar e confirmar **layouts visuais por empresas** *(Item 7)*
- [ ] Inserir a **latitude e longitude** no cadastro da Loja e da Matriz
- [ ] Permitir configurar o **volume padrão de Comerciais e Eventos** no painel administrativo do servidor.
- [ ] No cadastro de **Estilos Musicais**, calcular e exibir a **média de duração das músicas** associadas.
- [ ] No cadastro de músicas, identificar e marcar automaticamente a tag de **Música Longa** para faixas que ultrapassem 5 minutos de duração efetiva (com base no CUE IN/OUT).
- [ ] Criar tela de **Suporte Central** no servidor para receber conexões dos players remotos e espelhar o painel de diagnósticos/logs em tempo real.
- [ ] Revisar toda a parte visual e de usabilidade do sistema **IMC Server** (para melhorar a experiência do operador).
- [ ] Adicionar um checkbox (ligar/desligar) na tela de edição de mídias (Músicas e Comerciais) no painel do servidor para permitir **forçar ou remover manualmente o selo SONIC**, independentemente da esteira automática.
- [ ] Ajustar o cadastro dos **eventos**
- [ ] Ajustar o cadastro da **temperatura**




### 📻 IMC Local (Player - C:\imc-audio-local)
- [ ] **Ajustar o visual** do Player *(Item 2)*
- [ ] Inserir as **funcionalidades do MP3** (Controles de Arquivos) *(Item 3)*
- [ ] Configurar o `imc-local` para rodar como um **serviço no Windows** *(Item 4)*
- [ ] Implementar o **"Din Don"** (Chime antes dos anúncios de voz) *(Item 18)*
- [ ] Configurar a **recepção de texto e síntese local de áudio** para reprodução imediata com fade-out/fade-in da música de fundo *(Item 22)*
- [ ] Definir a **política de geração de logs**, para execução de músicas e comerciais
- [ ] **Transição Inteligente**: Estudar/implementar a possibilidade de usar transições com fade-in e fade-out mais compridos exclusivamente entre músicas (e skips de sincronismo) para dar uma percepção clara de troca de faixa. Para comerciais, usar transição praticamente nula (sem fade/corte direto) para garantir que o áudio seja reproduzido por completo e de forma bem audível do início ao fim (definindo estratégias específicas de controle de transição por tipo de arquivo).
- [ ] Ler e aplicar a configuração de **volume customizado para Comerciais e Eventos** enviada pelo servidor.
- [ ] Criar botão de **Acionar Suporte** no painel local que abre uma conexão de túnel reversa (WebSockets/Túnel HTTP) para o servidor, permitindo que a TI visualize os logs e o status atual do player sem precisar abrir portas na loja.
- [ ] **Limpeza Tardia de Músicas (Cache Local)**: Evitar a exclusão física imediata do HD dos arquivos de música que saem do sincronismo da grade do servidor, mantendo-os em cache para reaproveitamento instantâneo caso sejam reassociados no futuro. Implementar um Garbage Collector semanal para apagar arquivos não utilizados há mais de 7 dias.
- [ ] Desenvolver a **Hora Certa**
- [ ] Desenvolver a **Temperatura**
- [ ] Desenvolver os **Eventos**
- [ ] Desenvolver **Avisos de Veículos**






### 🔒 Segurança & Controle de Acesso (Compartilhado)
- [ ] Implementar questões de **segurança geral**: controle de **Usuários**, autenticação/segurança de **Acessos no LiquidSoap** e controle de **Acessos ao Player Local** *(Item 20)*

### 🔄 Programação Musical & Regras (Compartilhado)
- [ ] Definir como configurável as **regras de horário musical** fora do expediente *(Item 8)*
- [ ] Definir a **ordem aleatória** das músicas e controle dentro da programação musical *(Item 9)*
- [ ] Definir a **ordem de execução** das faixas dentro de um Bloco *(Item 10)*
- [ ] Implementar la opção de **Eventos** *(Item 19)*
- [ ] Criar uma tag de **Música Longa** no cadastro de músicas, e permitir configurar na Grade se o bloco aceita ou não músicas longas (padrão: **não aceitar**).
- [ ] Ajustar o algoritmo de escolha de músicas no Player Local para considerar a **duração efetiva** (descontando CUE IN/OUT) ao encaixar músicas no bloco, em vez de olhar apenas a duração total do arquivo.



### 📢 Conteúdo, Avisos & Utilidades (Compartilhado)
- [ ] Criar opção para **avisos de Placas de Veículo** (Interface Celular e disparo Local) *(Item 11)*
- [ ] Implementar recurso de **Hora Certa** na programação *(Item 12)*
- [ ] Implementar recurso de **Temperatura** *(Item 13)*
- [ ] Possibilidade de incluir **Spots pequenos com notícias** (região, mundo, interesse geral) *(Item 14)*
- [ ] Criação de **Vinhetas ou Spots de Ofertas** automáticos (pegando falas, material de jornal, ou montagem pelo cliente) *(Item 17)*
- [ ] Implementar módulo gerador de **TTS (Text-to-Speech)** gratuito e integrado (via Edge-TTS, Piper ou Windows SAPI) *(Item 21)*

### 📱 Planejamento & Novos Aplicativos (Futuro)
- [ ] Definir detalhadamente os **produtos** que deverão ser desenvolvidos *(Item 15)*
- [ ] Desenvolver **Aplicativo (APP) Mobile** para funcionar como microfone da loja e inserção rápida de ofertas *(Item 16)*

---

## ✅ Concluído recentemente
- [x] Relocalização do Script Liquidsoap e Correção do Sincronismo da Linha do Tempo (Local)
