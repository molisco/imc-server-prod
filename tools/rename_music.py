import os
import re
import argparse
import unicodedata

# Regex to match leading numbers and common delimiters (e.g. "07 - ", "008 - ", "09. ", "01_ ", "02 ")
PREFIX_PATTERN = re.compile(r"^\d+(?:\s*[-_.]\s*|\s+)")

def remove_accents(text):
    """
    Removes accents from string (e.g. 'á' -> 'a', 'ç' -> 'c').
    """
    normalized = unicodedata.normalize('NFKD', text)
    return "".join(c for c in normalized if not unicodedata.combining(c))

def sanitize_chars(text):
    """
    Replaces special quotes/apostrophes and filters out non-standard characters.
    """
    replacements = {
        '´': "'",
        '`': "'",
        '‘': "'",
        '’': "'",
        '“': '"',
        '”': '"',
        '–': '-',  # en dash
        '—': '-',  # em dash
    }
    for orig, repl in replacements.items():
        text = text.replace(orig, repl)
    
    # Keep only letters, digits, spaces, hyphens, underscores, dots, parentheses, brackets, and single quotes
    cleaned = re.sub(r"[^\w\s\-\(\)\[\]\.\']", "", text)
    # Collapse multiple spaces
    cleaned = re.sub(r"\s+", " ", cleaned)
    return cleaned.strip()

def get_unique_name(base_name, ext, used_names_lower):
    """
    Generates a unique target filename using the [REPETIDOxx] suffix if the name is already taken.
    """
    # Try the original clean name first
    target_name = f"{base_name}{ext}"
    if target_name.lower() not in used_names_lower:
        return target_name

    # If already taken, look for [REPETIDO01], [REPETIDO02], etc.
    count = 1
    while True:
        candidate = f"{base_name} [REPETIDO{count:02d}]{ext}"
        if candidate.lower() not in used_names_lower:
            return candidate
        count += 1

def process_directory(directory, remove_accents_active=False, sanitize_active=False, clean_names_active=False):
    """
    Processes a single directory to find files that need renaming and plans their new names.
    """
    try:
        items = os.listdir(directory)
    except Exception as e:
        print(f"Erro ao listar o diretório {directory}: {e}")
        return []

    files = [f for f in items if os.path.isfile(os.path.join(directory, f))]
    
    candidates = []
    unchanged_files = []

    for f in files:
        name, ext = os.path.splitext(f)
        
        # 1. Strip prefix
        match = PREFIX_PATTERN.match(name)
        if match:
            temp_name = name[match.end():].strip()
            if not temp_name:
                temp_name = name
        else:
            temp_name = name
            
        # 2. Remove accents
        if remove_accents_active:
            temp_name = remove_accents(temp_name)
            
        # 3. Sanitize characters
        if sanitize_active:
            temp_name = sanitize_chars(temp_name)
            
        # 4. Clean names (replace spaces with underlines)
        if clean_names_active:
            temp_name = temp_name.replace(" ", "_")
            temp_name = re.sub(r'_+', '_', temp_name)
            
        target_name = f"{temp_name}{ext}"
        
        if target_name != f:
            candidates.append((f, temp_name, ext))
        else:
            unchanged_files.append(f)

    # Keep track of used names in lowercase for case-insensitive collision checks
    used_names_lower = {f.lower() for f in unchanged_files}
    
    rename_plans = []

    # Sort files to ensure deterministic ordering (e.g. 04 before 07)
    candidates.sort(key=lambda x: x[0])

    for orig_name, base_target, ext in candidates:
        target_name = get_unique_name(base_target, ext, used_names_lower)
        
        # Mark as used so subsequent duplicates get incremented suffixes
        used_names_lower.add(target_name.lower())
        
        rename_plans.append({
            "dir": directory,
            "original": orig_name,
            "target": target_name
        })

    return rename_plans

def main():
    parser = argparse.ArgumentParser(description="Script para limpar prefixos de músicas, remover acentos/caracteres especiais e resolver duplicatas.")
    parser.add_argument(
        "--dir", "-d",
        default="midias/musicas",
        help="Diretório base para processar (padrão: midias/musicas)"
    )
    parser.add_argument(
        "--run",
        action="store_true",
        help="Executa a renomeação real dos arquivos (por padrão, apenas simula)"
    )
    parser.add_argument(
        "--no-recursive",
        action="store_true",
        help="Processa apenas o diretório informado, sem entrar em subpastas"
    )
    parser.add_argument(
        "--remove-accents",
        action="store_true",
        help="Remove acentuação dos nomes dos arquivos (ex: Você -> Voce)"
    )
    parser.add_argument(
        "--sanitize",
        action="store_true",
        help="Sanitiza caracteres especiais (mantém apenas letras, números, espaços, hífens, parênteses e colchetes)"
    )
    parser.add_argument(
        "--clean-names",
        action="store_true",
        help="Substitui espaços por underline ('_') nos nomes dos arquivos de saída"
    )

    args = parser.parse_args()
    
    target_dir = args.dir
    # If path is relative, resolve it relative to the current working directory
    if not os.path.isabs(target_dir):
        target_dir = os.path.abspath(target_dir)

    if not os.path.exists(target_dir):
        print(f"Erro: O diretório '{target_dir}' não existe.")
        return

    print("=" * 80)
    print(f"Iniciando varredura no diretório: {target_dir}")
    print(f"Modo: {'EFETIVO (Renomeando arquivos)' if args.run else 'SIMULAÇÃO (Nenhuma alteração será feita)'}")
    print(f"Remover acentos: {'SIM' if args.remove_accents else 'NÃO'}")
    print(f"Sanitizar especiais: {'SIM' if args.sanitize else 'NÃO'}")
    print(f"Substituir espaços por '_': {'SIM' if args.clean_names else 'NÃO'}")
    print("=" * 80)

    all_plans = []
    
    if args.no_recursive:
        all_plans.extend(process_directory(target_dir, args.remove_accents, args.sanitize, args.clean_names))
    else:
        for root, dirs, _ in os.walk(target_dir):
            # Sort dirs in place to ensure deterministic walking order
            dirs.sort()
            plans = process_directory(root, args.remove_accents, args.sanitize, args.clean_names)
            if plans:
                all_plans.extend(plans)

    if not all_plans:
        print("Nenhum arquivo precisa ser renomeado.")
        return

    print(f"Total de arquivos para renomear: {len(all_plans)}\n")
    
    success_count = 0
    fail_count = 0

    for plan in all_plans:
        orig_path = os.path.join(plan["dir"], plan["original"])
        target_path = os.path.join(plan["dir"], plan["target"])
        
        # Display relative path if possible for cleaner output
        rel_dir = os.path.relpath(plan["dir"], target_dir)
        dir_prefix = "" if rel_dir == "." else f"[{rel_dir}] "

        print(f"{dir_prefix}{plan['original']}  ===>  {plan['target']}")

        if args.run:
            try:
                os.rename(orig_path, target_path)
                success_count += 1
            except Exception as e:
                print(f"   -> ERRO ao renomear: {e}")
                fail_count += 1

    print("\n" + "=" * 80)
    if args.run:
        print(f"Renomeação concluída! Sucesso: {success_count}, Falhas: {fail_count}")
    else:
        print("Fim da simulação. Para aplicar as alterações, execute o comando com a opção --run:")
        # Build example command with selected options
        opts = []
        if args.remove_accents:
            opts.append("--remove-accents")
        if args.sanitize:
            opts.append("--sanitize")
        if args.clean_names:
            opts.append("--clean-names")
        opts_str = " " + " ".join(opts) if opts else ""
        print(f"python tools/rename_music.py --dir \"{target_dir}\"{opts_str} --run")
    print("=" * 80)

if __name__ == "__main__":
    main()
