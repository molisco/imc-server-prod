# Walkthrough - Inicialização do Projeto `imc-music-studio`

Criamos e estruturamos com sucesso o novo projeto independente **`imc-music-studio`** no diretório `/home/paulo-assis/imc-music-studio`, com controle de versão Git isolado, regras arquiteturais dedicadas e os módulos essenciais de análise física (DSP), taxonomia de curadoria, normalização a -14 LUFS e exportação de catálogos.

---

## 📁 Estrutura de Diretórios Criada

```
/home/paulo-assis/imc-music-studio/
├── .agents/
│   └── AGENTS.md                   # Regras de curadoria, taxonomia e padrões DSP
├── .gitignore                      # Ignora caches Python, áudios temporários e builds
├── README.md                       # Documentação e guia de uso da suíte
├── requirements.txt                # Dependências (librosa, scipy, soundfile, mutagen, etc.)
├── config.yaml                     # Configuração de metas (-14 LUFS, -35dB Cues, rotas)
├── cli.py                          # Interface CLI unificada (analyze, process, export)
├── atualizar_notebooklm.sh         # Script de sincronização de memória e regras
├── atualizar_notebooklm.bat        # Versão Windows do sincronizador
├── studio/
│   ├── core/
│   │   ├── taxonomy.py             # Enums: Macro-Gêneros, Subgêneros, Moods, Perfil Vocal, Fit
│   │   ├── audio_analyzer.py       # Extração DSP: BPM, R128 (LUFS/LRA/Peak), Cue In/Out
│   │   ├── vocal_analyzer.py       # Detecção de Perfil Vocal (Instrumental vs Cantado)
│   │   └── audio_normalizer.py     # Normalização para -14 LUFS e exportação MP3
│   ├── harvester/
│   │   ├── fma_harvester.py        # Coletor FMA com filtro anti-CC0 e prova visual 1:1
│   │   ├── fma_license_manager.py  # Gestão e auditoria de licenças
│   │   └── migrate_licenses_to_json.py # Migrador de certificados para JSON
│   ├── exporter/
│   │   ├── manifest.py             # Gerador de catalog_manifest.json com SHA-256
│   │   └── server_client.py        # Cliente REST para publicação no IMC-Server
│   ├── ai/                         # Módulo de sugestão de tags por IA
│   └── ui/                         # Interface visual do curador
├── docs/
│   ├── NOTEBOOKLM_IMC_KNOWLEDGE_BASE.md
│   ├── NOTEBOOK_LM_REGRAS_DE_NEGOCIO_IMC.md
│   ├── BUSINESS_RULES_AND_ARCHITECTURAL_DECISIONS.md
│   ├── TODO_PENDENCIAS_SISTEMA_IMC.md
│   └── brain/                      # Backup de memórias e conversas
└── tools/
    ├── backup_brain.py             # Backup cross-platform do Brain
    └── generate_notebooklm_docs.py # Compilador de documentação
```

---

## 🧪 Validação dos Módulos

* **Repositório Git Inicializado:** Branch `main` com commit raiz `33b5b55` (`feat(initial): initialize imc-music-studio curatorship suite and DSP engine`).
* **CLI Operacional:** Testado com sucesso `python3 cli.py --help`:
  ```
  usage: cli.py [-h] {analyze,process,export} ...
  IMC Music Studio - Ferramenta de Curadoria e Ingestão Musical
  ```
* **Sincronizador NotebookLM:** Executado e validado via `./atualizar_notebooklm.sh`.

---

## 🚀 Como Abrir o Projeto no Antigravity

1. No menu superior do Antigravity, clique em **File $\rightarrow$ Open Folder...** (ou pressione `Ctrl+O` / `Ctrl+K Ctrl+O`).
2. Selecione a pasta `/home/paulo-assis/imc-music-studio`.
3. Pronto! O Antigravity abrirá um Workspace 100% focado no **`imc-music-studio`**, com indexação limpa e chat dedicado à curadoria musical.
