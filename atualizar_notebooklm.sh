#!/usr/bin/env bash
set -e

echo "====================================================================="
echo "🧠 IMC - Backup de Memória & Gerador de Documentação (NotebookLM)"
echo "====================================================================="
echo ""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "1/2. Copiando planos, históricos e walkthroughs das conversas..."
python3 ./tools/backup_brain.py

echo ""
echo "2/2. Consolidando regras de negócio e gerando docs/NOTEBOOK_LM_REGRAS_DE_NEGOCIO_IMC.md..."
python3 ./tools/generate_notebooklm_docs.py

echo ""
echo "====================================================================="
echo "✅ SUCESSO! Documentação do NotebookLM atualizada com sucesso!"
echo "📄 Arquivo gerado em: docs/NOTEBOOK_LM_REGRAS_DE_NEGOCIO_IMC.md"
echo "📄 Base consolidada em: docs/NOTEBOOKLM_IMC_KNOWLEDGE_BASE.md"
echo "====================================================================="
echo ""
