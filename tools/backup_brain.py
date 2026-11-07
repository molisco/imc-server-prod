#!/usr/bin/env python3
"""
Backup Brain Tool
Copia arquivos de planos de implementação, walkthroughs, tasks e transcrições
de conversas do Antigravity (~/.gemini/antigravity/brain) para docs/brain/ no repositório.
Compatível com Linux, macOS e Windows sem depender de Go.
"""

import os
import shutil
from pathlib import Path

def main():
    home_dir = Path.home()
    brain_root_dir = home_dir / ".gemini" / "antigravity" / "brain"
    
    script_dir = Path(__file__).resolve().parent
    repo_root = script_dir.parent
    dest_root = repo_root / "docs" / "brain"
    
    print(f"🔍 Diretório raiz do Brain: {brain_root_dir}")
    print(f"📂 Diretório de destino: {dest_root}")
    
    if not brain_root_dir.exists():
        print(f"❌ Diretório do Brain não encontrado: {brain_root_dir}")
        return

    files_to_copy = {
        "implementation_plan.md": "implementation_plan.md",
        "walkthrough.md": "walkthrough.md",
        "task.md": "task.md",
        ".system_generated/logs/transcript.jsonl": "transcript.jsonl",
    }
    
    total_copied = 0
    total_convs = 0
    
    for entry in brain_root_dir.iterdir():
        if not entry.is_dir() or entry.name == "tempmediaStorage":
            continue
            
        conv_id = entry.name
        conv_src_dir = entry
        conv_dest_dir = dest_root / conv_id
        
        # Verifica se a pasta possui algum dos arquivos alvo
        has_any = False
        for src_rel in files_to_copy:
            if (conv_src_dir / src_rel).exists():
                has_any = True
                break
                
        if not has_any:
            continue
            
        conv_dest_dir.mkdir(parents=True, exist_ok=True)
        print(f"📦 Fazendo backup da conversa: {conv_id}")
        
        copied_count = 0
        for src_rel, dest_name in files_to_copy.items():
            src_path = conv_src_dir / src_rel
            dest_path = conv_dest_dir / dest_name
            
            if src_path.exists():
                try:
                    shutil.copy2(src_path, dest_path)
                    print(f"  ✓ Copiado: {src_rel} -> {dest_name}")
                    copied_count += 1
                    total_copied += 1
                except Exception as e:
                    print(f"  ❌ Erro ao copiar {src_rel}: {e}")
                    
        if copied_count > 0:
            total_convs += 1
            
    print(f"\n🎉 Backup concluído! Copiados {total_copied} arquivos em {total_convs} conversas.")

if __name__ == "__main__":
    main()
