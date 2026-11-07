@echo off
chcp 65001 > nul
echo =====================================================================
echo 🧠 IMC - Backup de Memória ^& Gerador de Documentação (NotebookLM)
echo =====================================================================
echo.

cd /d "%~dp0"

echo 1/2. Copiando planos, históricos e walkthroughs das conversas...
python ./tools/backup_brain.py
if %ERRORLEVEL% neq 0 (
    echo ❌ Erro ao executar backup do brain.
    pause
    exit /b %ERRORLEVEL%
)

echo.
echo 2/2. Consolidando regras de negócio e gerando docs/NOTEBOOK_LM_REGRAS_DE_NEGOCIO_IMC.md...
python ./tools/generate_notebooklm_docs.py
if %ERRORLEVEL% neq 0 (
    echo ❌ Erro ao gerar documentação NotebookLM.
    pause
    exit /b %ERRORLEVEL%
)

echo.
echo =====================================================================
echo ✅ SUCESSO! Documentação do NotebookLM atualizada com sucesso!
echo 📄 Arquivo gerado em: docs\NOTEBOOK_LM_REGRAS_DE_NEGOCIO_IMC.md
echo =====================================================================
echo.
pause
