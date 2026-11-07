#!/usr/bin/env python3
"""
Gerador de Fonte Única da Verdade para NotebookLM
Compila todas as regras de negócio, arquitetura, definições do sistema e históricos
de funcionalidades do IMC (imc-server e imc-audio-local) em um arquivo Markdown consolidado.
"""

import os
import glob

def main():
    root_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    docs_dir = os.path.join(root_dir, "docs")
    output_file = os.path.join(docs_dir, "NOTEBOOK_LM_REGRAS_DE_NEGOCIO_IMC.md")

    sections = []

    # Cabeçalho Principal
    sections.append("# MANUAL MESTRE DE REGRAS DE NEGÓCIO E ARQUITETURA - SISTEMA IMC (IMC-SERVER & IMC-AUDIO-LOCAL)\n")
    sections.append("> Este documento é a Fonte Única da Verdade contendo toda a inteligência, regras de negócio, arquitetura de playout e especificações funcionais da plataforma IMC (In-Store Music Controller).\n")

    # 1. Regras Gerais de Arquitetura (.agents/AGENTS.md)
    agents_path = os.path.join(root_dir, ".agents", "AGENTS.md")
    if os.path.exists(agents_path):
        with open(agents_path, "r", encoding="utf-8") as f:
            sections.append("## 1. Diretrizes de Arquitetura e Regras Principais do Sistema\n")
            sections.append(f.read().strip() + "\n")

    # 2. Regras Hotmix (docs/HOTMIX_BUSINESS_RULES.md)
    hotmix_path = os.path.join(docs_dir, "HOTMIX_BUSINESS_RULES.md")
    if os.path.exists(hotmix_path):
        with open(hotmix_path, "r", encoding="utf-8") as f:
            sections.append("## 2. Regras de Negócio de Integração Hotmix Streaming\n")
            sections.append(f.read().strip() + "\n")

    # 3. Módulos e Funcionalidades (docs/brain/*/walkthrough.md e implementation_plan.md)
    sections.append("## 3. Especificações Técnicas e Funcionalidades do Sistema\n")
    brain_dir = os.path.join(docs_dir, "brain")
    if os.path.exists(brain_dir):
        conv_folders = glob.glob(os.path.join(brain_dir, "*"))
        for folder in sorted(conv_folders):
            if not os.path.isdir(folder):
                continue
            conv_id = os.path.basename(folder)
            walkthrough_path = os.path.join(folder, "walkthrough.md")
            plan_path = os.path.join(folder, "implementation_plan.md")
            
            content = ""
            if os.path.exists(walkthrough_path):
                with open(walkthrough_path, "r", encoding="utf-8") as f:
                    content += f.read().strip() + "\n\n"
            elif os.path.exists(plan_path):
                with open(plan_path, "r", encoding="utf-8") as f:
                    content += f.read().strip() + "\n\n"

            if content:
                sections.append(f"### Módulo / Atualização ({conv_id})\n")
                sections.append(content)

    # 4. Textos Complementares de Negócio
    sections.append("## 4. Requisitos de Negócio e Wireframes Históricos\n")
    for negocio_file in sorted(glob.glob(os.path.join(docs_dir, "negocio_part*.txt"))):
        base = os.path.basename(negocio_file)
        with open(negocio_file, "r", encoding="utf-8", errors="ignore") as f:
            text = f.read().strip()
            # Limitar texto longo para manter foco na inteligência
            if len(text) > 30000:
                text = text[:30000] + "\n...[conteúdo resumido]..."
            sections.append(f"#### Documento de Negócio: {base}\n")
            sections.append(text + "\n")

    # 5. Lista Mestre de Tarefas e Pendências (TODO / Roadmap)
    todo_path = os.path.join(docs_dir, "TODO_PENDENCIAS_SISTEMA_IMC.md")
    if os.path.exists(todo_path):
        with open(todo_path, "r", encoding="utf-8") as f:
            sections.append("## 5. Roteiro de Desenvolvimento, Tarefas Concluídas e Pendências (TODO)\n")
            sections.append(f.read().strip() + "\n")

    full_document = "\n\n---\n\n".join(sections)

    with open(output_file, "w", encoding="utf-8") as f:
        f.write(full_document)

    generate_player_manual(root_dir, docs_dir)
    generate_server_manual(root_dir, docs_dir)
    generate_help_hub(root_dir, docs_dir)

def generate_player_manual(root_dir, docs_dir):
    html_content = """<!DOCTYPE html>
<html lang="pt-br">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Manual do Operador - Web Player IMC</title>
    <!-- Bootstrap 5 CSS & FontAwesome Icons -->
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/css/bootstrap.min.css" rel="stylesheet">
    <link href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css" rel="stylesheet">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@300;400;500;600;700;800&family=JetBrains+Mono:wght@400;500;700&display=swap" rel="stylesheet">

    <style>
        :root {
            --primary: #4364f7;
            --primary-dark: #1e3a8a;
            --accent-cyan: #06b6d4;
            --accent-green: #10b981;
            --accent-amber: #f59e0b;
            --accent-rose: #f43f5e;
            --bg-dark: #0b0f19;
            --bg-card: #131b2e;
            --bg-card-alt: #1a243d;
            --text-main: #f8fafc;
            --text-muted: #94a3b8;
            --border-color: rgba(255, 255, 255, 0.1);
        }

        * {
            box-sizing: border-box;
        }

        body {
            font-family: 'Plus Jakarta Sans', sans-serif;
            background-color: var(--bg-dark);
            color: var(--text-main);
            line-height: 1.6;
            margin: 0;
            padding: 0;
        }

        /* Top Action Bar */
        .manual-top-bar {
            background: rgba(11, 15, 25, 0.95);
            backdrop-filter: blur(12px);
            border-bottom: 1px solid var(--border-color);
            position: sticky;
            top: 0;
            z-index: 1000;
            padding: 0.75rem 2rem;
        }

        .btn-pdf {
            background: linear-gradient(135deg, #e11d48, #be123c);
            color: #ffffff;
            font-weight: 700;
            border: none;
            border-radius: 8px;
            padding: 0.55rem 1.25rem;
            display: inline-flex;
            align-items: center;
            gap: 8px;
            box-shadow: 0 4px 12px rgba(225, 29, 72, 0.35);
            transition: all 0.2s ease;
            text-decoration: none;
            cursor: pointer;
        }

        .btn-pdf:hover {
            background: linear-gradient(135deg, #f43f5e, #e11d48);
            transform: translateY(-2px);
            box-shadow: 0 6px 16px rgba(225, 29, 72, 0.5);
            color: #ffffff;
        }

        .container-manual {
            max-width: 1140px;
            margin: 2rem auto;
            padding: 0 1.5rem;
        }

        /* Hero Header */
        .manual-hero {
            background: linear-gradient(135deg, rgba(67, 100, 247, 0.15) 0%, rgba(6, 182, 212, 0.08) 100%);
            border: 1px solid rgba(67, 100, 247, 0.3);
            border-radius: 20px;
            padding: 2.5rem;
            margin-bottom: 2.5rem;
            position: relative;
            overflow: hidden;
        }

        .manual-hero::before {
            content: '';
            position: absolute;
            top: -50%;
            right: -20%;
            width: 400px;
            height: 400px;
            background: radial-gradient(circle, rgba(6, 182, 212, 0.15), transparent 70%);
            pointer-events: none;
        }

        .manual-badge {
            background: rgba(6, 182, 212, 0.2);
            color: var(--accent-cyan);
            border: 1px solid rgba(6, 182, 212, 0.4);
            padding: 0.35rem 0.85rem;
            border-radius: 30px;
            font-size: 0.8rem;
            font-weight: 700;
            letter-spacing: 1px;
            text-transform: uppercase;
            display: inline-block;
            margin-bottom: 1rem;
        }

        /* Table of Contents */
        .toc-card {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-radius: 16px;
            padding: 1.5rem 2rem;
            margin-bottom: 3rem;
        }

        .toc-list {
            list-style: none;
            padding: 0;
            margin: 0;
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
            gap: 0.75rem;
        }

        .toc-list li a {
            color: var(--text-muted);
            text-decoration: none;
            display: flex;
            align-items: center;
            gap: 10px;
            padding: 0.5rem 0.75rem;
            border-radius: 8px;
            transition: all 0.2s ease;
            font-weight: 500;
        }

        .toc-list li a:hover {
            color: var(--text-main);
            background: rgba(255, 255, 255, 0.05);
            transform: translateX(4px);
        }

        .toc-number {
            background: rgba(67, 100, 247, 0.2);
            color: #60a5fa;
            width: 24px;
            height: 24px;
            border-radius: 6px;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            font-size: 0.75rem;
            font-weight: 700;
        }

        /* Content Sections */
        .manual-section {
            margin-bottom: 3.5rem;
        }

        .section-header {
            display: flex;
            align-items: center;
            gap: 12px;
            margin-bottom: 1.25rem;
            padding-bottom: 0.75rem;
            border-bottom: 2px solid rgba(67, 100, 247, 0.3);
        }

        .section-icon {
            width: 42px;
            height: 42px;
            border-radius: 10px;
            background: linear-gradient(135deg, var(--primary), var(--accent-cyan));
            color: #ffffff;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.2rem;
        }

        .section-title {
            font-size: 1.6rem;
            font-weight: 800;
            margin: 0;
            color: #ffffff;
        }

        /* UI Mockups */
        .ui-mockup {
            background: #0f1422;
            border: 1px solid rgba(255, 255, 255, 0.15);
            border-radius: 14px;
            padding: 1.5rem;
            margin: 1.5rem 0;
            box-shadow: 0 15px 35px rgba(0, 0, 0, 0.6);
            position: relative;
        }

        .mockup-header {
            font-size: 0.75rem;
            text-transform: uppercase;
            letter-spacing: 1.5px;
            color: var(--text-muted);
            margin-bottom: 1rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            border-bottom: 1px solid rgba(255, 255, 255, 0.08);
            padding-bottom: 0.5rem;
        }

        /* Callout Boxes */
        .callout {
            border-radius: 12px;
            padding: 1.25rem 1.5rem;
            margin: 1.25rem 0;
            border-left: 4px solid;
        }

        .callout-info {
            background: rgba(6, 182, 212, 0.08);
            border-color: var(--accent-cyan);
            color: #e0f2fe;
        }

        .callout-tip {
            background: rgba(16, 185, 129, 0.08);
            border-color: var(--accent-green);
            color: #d1fae5;
        }

        .callout-warning {
            background: rgba(245, 158, 11, 0.08);
            border-color: var(--accent-amber);
            color: #fef3c7;
        }

        .callout-title {
            font-weight: 700;
            margin-bottom: 0.35rem;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        /* Feature Cards Grid */
        .feature-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 1.25rem;
            margin: 1.5rem 0;
        }

        .feature-card {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-radius: 14px;
            padding: 1.5rem;
            transition: all 0.2s ease;
        }

        .feature-card:hover {
            border-color: rgba(67, 100, 247, 0.4);
            transform: translateY(-2px);
        }

        .feature-card h5 {
            color: #ffffff;
            font-weight: 700;
            display: flex;
            align-items: center;
            gap: 10px;
            margin-bottom: 0.75rem;
        }

        /* Tables */
        .table-manual {
            width: 100%;
            border-collapse: separate;
            border-spacing: 0;
            border: 1px solid var(--border-color);
            border-radius: 12px;
            overflow: hidden;
            margin: 1.5rem 0;
        }

        .table-manual th {
            background: var(--bg-card-alt);
            color: #ffffff;
            padding: 0.85rem 1.25rem;
            font-weight: 700;
            font-size: 0.85rem;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            border-bottom: 1px solid var(--border-color);
        }

        .table-manual td {
            background: var(--bg-card);
            padding: 0.85rem 1.25rem;
            font-size: 0.9rem;
            border-bottom: 1px solid var(--border-color);
        }

        .table-manual tr:last-child td {
            border-bottom: none;
        }

        /* Mockup Component Samples */
        .mock-player-controls {
            background: #161a29;
            border-radius: 12px;
            padding: 1.25rem;
            display: flex;
            align-items: center;
            justify-content: space-between;
            flex-wrap: wrap;
            gap: 1rem;
        }

        .mock-play-btn {
            width: 50px;
            height: 50px;
            border-radius: 50%;
            background: var(--accent-green);
            color: #ffffff;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.25rem;
            box-shadow: 0 0 15px rgba(16, 185, 129, 0.5);
        }

        .mock-badge-live {
            background: rgba(16, 185, 129, 0.15);
            color: #34d399;
            border: 1px solid rgba(16, 185, 129, 0.3);
            border-radius: 20px;
            padding: 0.3rem 0.75rem;
            font-size: 0.75rem;
            font-weight: 700;
            display: inline-flex;
            align-items: center;
            gap: 6px;
        }

        .mock-badge-live::before {
            content: '';
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background-color: #10b981;
            box-shadow: 0 0 8px #10b981;
        }

        /* Footer */
        .manual-footer {
            border-top: 1px solid var(--border-color);
            padding: 2.5rem 0;
            margin-top: 4rem;
            text-align: center;
            color: var(--text-muted);
            font-size: 0.85rem;
        }

        /* PRINT STYLES FOR PDF EXPORT */
        @media print {
            body {
                background: #ffffff !important;
                color: #0f172a !important;
                font-size: 11pt;
            }

            .manual-top-bar, .btn-pdf, .btn-print-hide {
                display: none !important;
            }

            .container-manual {
                max-width: 100% !important;
                margin: 0 !important;
                padding: 0 !important;
            }

            .manual-hero {
                background: #f8fafc !important;
                border: 1px solid #cbd5e1 !important;
                color: #0f172a !important;
                page-break-after: avoid;
            }

            .manual-section {
                page-break-inside: avoid;
                margin-bottom: 2rem !important;
            }

            .section-title {
                color: #0f172a !important;
            }

            .toc-card, .feature-card, .ui-mockup {
                background: #f8fafc !important;
                border: 1px solid #cbd5e1 !important;
                color: #0f172a !important;
                box-shadow: none !important;
            }

            .table-manual th {
                background: #e2e8f0 !important;
                color: #0f172a !important;
                border-color: #cbd5e1 !important;
            }

            .table-manual td {
                background: #ffffff !important;
                color: #334155 !important;
                border-color: #cbd5e1 !important;
            }

            .callout {
                background: #f1f5f9 !important;
                color: #0f172a !important;
                border-left-width: 4px !important;
            }

            .callout-info { border-color: #0284c7 !important; }
            .callout-tip { border-color: #16a34a !important; }
            .callout-warning { border-color: #d97706 !important; }

            .mock-player-controls {
                background: #f1f5f9 !important;
                border: 1px solid #cbd5e1 !important;
            }

            h1, h2, h3, h4, h5 {
                color: #0f172a !important;
            }
        }
    </style>
</head>
<body>

    <!-- Barra de Ações Superior -->
    <header class="manual-top-bar d-flex justify-content-between align-items-center">
        <div class="d-flex align-items-center gap-3">
            <span class="badge bg-primary px-3 py-2 fw-bold" style="letter-spacing: 1px;">IMC-SERVER</span>
            <span class="fw-bold text-white fs-6">Manual do Operador — Web Player de Loja</span>
        </div>
        <div class="d-flex align-items-center gap-2">
            <button class="btn-pdf" onclick="window.print()">
                <i class="fa-solid fa-file-pdf fs-5"></i> Exportar para PDF / Imprimir
            </button>
        </div>
    </header>

    <div class="container-manual">

        <!-- Hero de Abertura -->
        <div class="manual-hero">
            <span class="manual-badge">Documentação Oficial & Guia Prático</span>
            <h1 class="fw-extrabold text-white mb-2" style="font-size: 2.2rem;">Manual de Operação do Web Player</h1>
            <p class="text-muted fs-5 mb-0" style="max-width: 800px;">
                Guia passo a passo para gerentes, atendentes e operadores de loja utilizarem o reprodutor de áudio corporativo, executarem chamadas de estacionamento, acompanharem a transmissão ao vivo e garantirem a melhor experiência sonora no ponto de venda.
            </p>
        </div>

        <!-- Índice Rápido -->
        <div class="toc-card">
            <h5 class="fw-bold text-white mb-3 d-flex align-items-center gap-2">
                <i class="fa-solid fa-list-check text-primary"></i> Sumário do Manual
            </h5>
            <ul class="toc-list">
                <li><a href="#sec-1"><span class="toc-number">1</span> Visão Geral e Acesso Seguro</a></li>
                <li><a href="#sec-2"><span class="toc-number">2</span> Controles de Reprodução e Volume</a></li>
                <li><a href="#sec-3"><span class="toc-number">3</span> Modos de Áudio (Streaming vs Acervo)</a></li>
                <li><a href="#sec-4"><span class="toc-number">4</span> Chamadas de Veículos & Avisos</a></li>
                <li><a href="#sec-5"><span class="toc-number">5</span> Estrutura de Blocos & Histórico</a></li>
                <li><a href="#sec-6"><span class="toc-number">6</span> Playout 24/7 & Reconexão Automática</a></li>
                <li><a href="#sec-7"><span class="toc-number">7</span> Dúvidas Frequentes & Diagnóstico</a></li>
            </ul>
        </div>

        <!-- SEÇÃO 1: VISÃO GERAL -->
        <section id="sec-1" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-door-open"></i></div>
                <h2 class="section-title">1. Visão Geral e Acesso Seguro</h2>
            </div>
            <p>
                O <strong>Web Player IMC</strong> é a estação de trabalho sonora da sua loja. Ele roda diretamente no navegador de internet (Google Chrome, Microsoft Edge, Firefox) sem necessidade de instalar aplicativos pesados, transmitindo músicas 24 horas por dia, vinhetas institucionais, ofertas comerciais e comunicados instantâneos de estacionamento.
            </p>

            <div class="feature-grid">
                <div class="feature-card">
                    <h5><i class="fa-solid fa-user-shield text-info"></i> Isolamento por Loja (RBAC)</h5>
                    <p class="text-muted small mb-0">
                        Cada operador possui acesso restrito exclusivamente ao player da sua filial (ex: <code>/player/001</code>). Não há risco de alterar acidentalmente a programação de outras lojas ou configurações administrativas da matriz.
                    </p>
                </div>
                <div class="feature-card">
                    <h5><i class="fa-solid fa-bolt text-warning"></i> Playout Contínuo</h5>
                    <p class="text-muted small mb-0">
                        O player foi projetado para nunca parar. Se houver oscilação de rede ou troca de faixa, o sistema de reconexão automática restaura a reprodução em menos de 3 segundos sem intervenção manual.
                    </p>
                </div>
            </div>

            <div class="callout callout-info">
                <div class="callout-title"><i class="fa-solid fa-circle-info"></i> Como Acessar no Início do Expediente:</div>
                Abra o navegador no computador da loja, acesse o link fornecido pela equipe de TI (ex: <code>http://servidor-imc:8080/player/sua-loja</code>), efetue o login com seu usuário de operador e clique no botão verde <strong>"Iniciar Transmissão"</strong>.
            </div>
        </section>

        <!-- SEÇÃO 2: CONTROLES DO PLAYER -->
        <section id="sec-2" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-sliders"></i></div>
                <h2 class="section-title">2. Controles de Reprodução e Volume</h2>
            </div>
            <p>
                A interface principal exibe em destaque o status da transmissão, a arte da rádio, o título da mídia no ar e os controles essenciais de áudio.
            </p>

            <!-- Mockup Visual dos Controles -->
            <div class="ui-mockup">
                <div class="mockup-header">
                    <span><i class="fa-solid fa-desktop me-1"></i> Representação Visual da Tela Principal</span>
                    <span class="mock-badge-live">NO AR (AO VIVO)</span>
                </div>
                <div class="mock-player-controls">
                    <div class="d-flex align-items-center gap-3">
                        <div class="mock-play-btn"><i class="fa-solid fa-play"></i></div>
                        <div>
                            <div class="fw-bold text-white fs-5">Hotmix Instore - Shopping Vibes</div>
                            <div class="text-muted small">Shopping Cidade Sorocaba • <strong>Streaming Ao Vivo</strong></div>
                        </div>
                    </div>
                    <div class="d-flex align-items-center gap-3">
                        <span class="badge bg-dark border border-secondary text-info px-3 py-2">
                            <i class="fa-solid fa-hourglass-half me-1"></i> --:--
                        </span>
                        <div class="d-flex align-items-center gap-2 bg-dark p-2 rounded border border-secondary">
                            <i class="fa-solid fa-volume-high text-muted"></i>
                            <input type="range" class="form-range" style="width: 100px;" value="80">
                        </div>
                    </div>
                </div>
            </div>

            <table class="table-manual">
                <thead>
                    <tr>
                        <th style="width: 25%;">Elemento</th>
                        <th style="width: 25%;">Identificação Visual</th>
                        <th>Função e Comportamento</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td><strong>Botão Play / Pause</strong></td>
                        <td><span class="badge bg-success"><i class="fa-solid fa-play me-1"></i> Tocar / Pausar</span></td>
                        <td>Inicia ou pausa a reprodução local do áudio da loja. Ao dar play, ativa o mecanismo contínuo de áudio.</td>
                    </tr>
                    <tr>
                        <td><strong>Controle de Volume</strong></td>
                        <td><i class="fa-solid fa-volume-high text-info me-1"></i> Slider deslizante (0 a 100%)</td>
                        <td>Ajusta o nível de som do computador da loja. O volume é salvo automaticamente na memória do navegador.</td>
                    </tr>
                    <tr>
                        <td><strong>Tempo Restante</strong></td>
                        <td><i class="fa-solid fa-hourglass-half text-warning me-1"></i> Contagem regressiva</td>
                        <td>Em modo Streaming exibe <code>--:--</code> (fluxo contínuo). Em modo MP3 exibe os minutos/segundos restantes para a próxima faixa.</td>
                    </tr>
                    <tr>
                        <td><strong>Botão Pular Música</strong></td>
                        <td><span class="badge bg-secondary"><i class="fa-solid fa-forward-step me-1"></i> Pular</span></td>
                        <td>Fica visível <strong>apenas</strong> quando a rádio está tocando faixas do acervo MP3 local. Em transmissões de Streaming ao vivo, ele é automaticamente ocultado para proteger a sincronia da rádio.</td>
                    </tr>
                </tbody>
            </table>
        </section>

        <!-- SEÇÃO 3: MODOS DE TRANSMISSÃO -->
        <section id="sec-3" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-radio"></i></div>
                <h2 class="section-title">3. Modos de Áudio (Streaming vs Acervo Local)</h2>
            </div>
            <p>
                O IMC opera de forma inteligente alternando automaticamente entre dois modos de exibição conforme a grade definida pela matriz:
            </p>

            <div class="row g-3">
                <div class="col-md-6">
                    <div class="feature-card h-100" style="border-top: 4px solid var(--accent-cyan);">
                        <h5 class="text-cyan"><i class="fa-solid fa-tower-broadcast text-info"></i> Modo Streaming Online (Rádio Web)</h5>
                        <ul class="text-muted small ps-3 mb-0">
                            <li>Transmissão ao vivo de canais especializados (ex: *Shopping Vibes, Workout, Best-of, Lounge*).</li>
                            <li>Tempo restante formatado com tracinhos (<code>--:--</code>), pois o fluxo de streaming é contínuo e ininterrupto.</li>
                            <li>Botão "Pular Música" oculto automaticamente para preservar a integridade da transmissão.</li>
                            <li>Comerciais e avisos entram pontualmente no minuto exato programado sem cortar a harmonia.</li>
                        </ul>
                    </div>
                </div>
                <div class="col-md-6">
                    <div class="feature-card h-100" style="border-top: 4px solid var(--accent-green);">
                        <h5 class="text-success"><i class="fa-solid fa-music text-success"></i> Modo Acervo Local (Biblioteca MP3)</h5>
                        <ul class="text-muted small ps-3 mb-0">
                            <li>Execução de arquivos musicais gravados com o Selo SONIC de alta fidelidade.</li>
                            <li>Transição suave com *Crossfade* de 3 segundos entre o final de uma música e o início da próxima.</li>
                            <li>Botão "Pular Música" habilitado para a equipe da loja avançar de faixa se necessário.</li>
                            <li>Funciona 100% offline mesmo em casos de perda total de conexão com a internet.</li>
                        </ul>
                    </div>
                </div>
            </div>
        </section>

        <!-- SEÇÃO 4: AVISOS E VEÍCULOS -->
        <section id="sec-4" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-bullhorn"></i></div>
                <h2 class="section-title">4. Chamadas de Veículos & Avisos Instantâneos</h2>
            </div>
            <p>
                Na aba <strong>"Avisos Instantâneos & Veículos"</strong>, o operador pode emitir comunicados urgentes para o salão de vendas ou estacionamento com apenas 2 cliques.
            </p>

            <div class="ui-mockup">
                <div class="mockup-header">
                    <span><i class="fa-solid fa-car me-1"></i> Módulo de Chamada de Veículos</span>
                    <span class="badge bg-warning text-dark fw-bold">ATENUAÇÃO AUTOMÁTICA (DUCKING)</span>
                </div>
                <div class="row g-3">
                    <div class="col-md-4">
                        <label class="form-label text-muted small">Placa do Veículo:</label>
                        <div class="input-group">
                            <span class="input-group-text bg-dark border-secondary text-warning"><i class="fa-solid fa-car-side"></i></span>
                            <input type="text" class="form-control bg-dark border-secondary text-white fw-bold text-uppercase" value="ABC-1234" readonly>
                        </div>
                    </div>
                    <div class="col-md-5">
                        <label class="form-label text-muted small">Motivo da Chamada:</label>
                        <select class="form-select bg-dark border-secondary text-white" disabled>
                            <option>Faróis Acesos no Estacionamento</option>
                        </select>
                    </div>
                    <div class="col-md-3 d-flex align-items-end">
                        <button class="btn btn-primary w-100 fw-bold" disabled>
                            <i class="fa-solid fa-paper-plane me-1"></i> Disparar Aviso
                        </button>
                    </div>
                </div>
            </div>

            <div class="callout callout-tip">
                <div class="callout-title"><i class="fa-solid fa-wand-magic-sparkles"></i> Tecnologia de Ducking Inteligente:</div>
                Quando você clica em disparar um aviso, o sistema IMC não corta o som abruptamente. Ele toca a vinheta de atenção (*Din-Don*), <strong>abaixa suavemente a música para 3% do volume</strong>, executa a voz do comunicado com clareza cristalina e, ao término, devolve o volume da música ao nível normal de forma suave.
            </div>

            <h5 class="fw-bold text-white mt-4 mb-2">Principais Motivos de Chamada Disponíveis:</h5>
            <div class="row g-2 text-muted small">
                <div class="col-md-6">• <strong>Faróis Acesos:</strong> Solicita ao proprietário para desligar as luzes do veículo.</div>
                <div class="col-md-6">• <strong>Bloqueando Saída / Garagem:</strong> Alerta para desobstrução de vaga ou rampa.</div>
                <div class="col-md-6">• <strong>Vidros Abertos:</strong> Alerta para fechamento de janelas em dias de chuva.</div>
                <div class="col-md-6">• <strong>Alarme Disparado:</strong> Convoca o condutor até o veículo no estacionamento.</div>
            </div>
        </section>

        <!-- SEÇÃO 5: HISTÓRICO E ESTRUTURA -->
        <section id="sec-5" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-table-list"></i></div>
                <h2 class="section-title">5. Estrutura de Blocos & Histórico de Execuções</h2>
            </div>
            <p>
                Na aba <strong>"Estrutura de Blocos & Histórico"</strong>, o operador tem total transparência do que está sendo veiculado na loja, servindo como ferramenta de auditoria e conferência.
            </p>

            <table class="table-manual">
                <thead>
                    <tr>
                        <th>Status</th>
                        <th>Significado no Player</th>
                        <th>Ação do Sistema</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td><span class="badge bg-success bg-opacity-25 text-success border border-success">executada</span></td>
                        <td>Mídia finalizada com sucesso.</td>
                        <td>Auditada e gravada no relatório de comprovação (Proof-of-Play).</td>
                    </tr>
                    <tr>
                        <td><span class="badge bg-primary bg-opacity-25 text-primary border border-primary">na_ponta</span></td>
                        <td>Faixa em reprodução neste exato segundo.</td>
                        <td>Alimenta o visor principal do Web Player e os medidores de progresso.</td>
                    </tr>
                    <tr>
                        <td><span class="badge bg-warning bg-opacity-25 text-warning border border-warning">na_fila</span></td>
                        <td>Próxima faixa reservada na memória.</td>
                        <td>Pré-carregada para garantir transição instantânea sem pausas.</td>
                    </tr>
                    <tr>
                        <td><span class="badge bg-secondary bg-opacity-25 text-muted border border-secondary">pulada</span></td>
                        <td>Faixa pulada pelo operador ou ajustada por horário.</td>
                        <td>Registrada para auditoria de rodízio musical.</td>
                    </tr>
                </tbody>
            </table>
        </section>

        <!-- SEÇÃO 6: WATCHDOG & 24/7 -->
        <section id="sec-6" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-shield-halved"></i></div>
                <h2 class="section-title">6. Playout 24/7 & Reconexão Automática</h2>
            </div>
            <p>
                O Web Player IMC possui um <strong>Watchdog Ativo de 3 Segundos</strong> integrado diretamente na engine de reprodução:
            </p>

            <div class="feature-grid">
                <div class="feature-card">
                    <h5><i class="fa-solid fa-heart-pulse text-rose"></i> Monitoramento de Batimento Cardíaco (Heartbeat)</h5>
                    <p class="text-muted small mb-0">
                        O navegador verifica a cada 3 segundos se o ponteiro de áudio está progredindo. Caso ocorra um congelamento de buffer devido a instabilidade externa de internet, o watchdog força uma reconexão instantânea em segundo plano.
                    </p>
                </div>
                <div class="feature-card">
                    <h5><i class="fa-solid fa-rotate text-info"></i> Preservação da Intenção de Tocar (User Intent)</h5>
                    <p class="text-muted small mb-0">
                        Uma vez que o operador clicou em *Play*, o player entende que a loja deve permanecer tocando. Atualizações de catálogo ou trocas de stream nunca encerram o áudio da sua loja.
                    </p>
                </div>
            </div>
        </section>

        <!-- SEÇÃO 7: DÚVIDAS E SOLUÇÕES -->
        <section id="sec-7" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-circle-question"></i></div>
                <h2 class="section-title">7. Dúvidas Frequentes & Solução Rápida</h2>
            </div>

            <div class="accordion" id="faqAccordion">
                <div class="feature-card mb-3">
                    <h5 class="text-white"><i class="fa-solid fa-volume-xmark text-warning"></i> O player está tocando mas não sai som nas caixas da loja. O que fazer?</h5>
                    <p class="text-muted small mb-0">
                        1. Verifique se a barra de volume do Web Player não está zerada ou no mudo.<br>
                        2. Verifique o volume do próprio Windows/computador da loja.<br>
                        3. Certifique-se de que o cabo P2/RCA do amplificador está plugado na saída de som correta.
                    </p>
                </div>

                <div class="feature-card mb-3">
                    <h5 class="text-white"><i class="fa-solid fa-wifi text-info"></i> A internet da loja caiu. A música vai parar?</h5>
                    <p class="text-muted small mb-0">
                        Não! Se a rádio estiver programada com acervo local, ela continuará tocando normalmente offline. Caso estivesse em streaming, o sistema aguarda a conexão ser restabelecida e reconecta automaticamente assim que o sinal voltar.
                    </p>
                </div>

                <div class="feature-card">
                    <h5 class="text-white"><i class="fa-solid fa-ban text-danger"></i> Por que o botão "Pular Música" sumiu da tela?</h5>
                    <p class="text-muted small mb-0">
                        Isso ocorre quando a grade do dia/horário está transmitindo uma emissora de <strong>Streaming Ao Vivo</strong> (ex: *Hotmix*). Em transmissões de rádio web externa ao vivo não é possível pular faixas, sendo um comportamento normal e esperado do sistema.
                    </p>
                </div>
            </div>
        </section>

        <!-- Rodapé do Manual -->
        <footer class="manual-footer">
            <div class="fw-bold text-white mb-1">SISTEMA IMC — IN-STORE MUSIC CONTROLLER</div>
            <div>Versão da Plataforma: <strong>v1.5.0 (SONIC Edition)</strong> • Documento Homologado para Operação de Loja</div>
            <div class="mt-2 text-muted" style="font-size: 0.75rem;">Para suporte técnico avançado ou dúvidas sobre a grade musical, contate a Central de Operações da Matriz.</div>
        </footer>

    </div>

    <!-- Bootstrap 5 JS -->
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>"""

    manual_docs_path = os.path.join(docs_dir, "manual_player.html")
    with open(manual_docs_path, "w", encoding="utf-8") as f:
        f.write(html_content)

    templates_dir = os.path.join(root_dir, "web", "templates")
    if os.path.exists(templates_dir):
        manual_tmpl_path = os.path.join(templates_dir, "manual_player.html")
        with open(manual_tmpl_path, "w", encoding="utf-8") as f:
            f.write(html_content)

    print(f"[SUCCESS] Manual HTML do Player gerado em: {manual_docs_path}")

def generate_server_manual(root_dir, docs_dir):
    html_content = """<!DOCTYPE html>
<html lang="pt-br">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Manual do Administrador e Gestor - IMC-Server</title>
    <!-- Bootstrap 5 CSS & FontAwesome Icons -->
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/css/bootstrap.min.css" rel="stylesheet">
    <link href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css" rel="stylesheet">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@300;400;500;600;700;800&family=JetBrains+Mono:wght@400;500;700&display=swap" rel="stylesheet">

    <style>
        :root {
            --primary: #4364f7;
            --primary-dark: #1e3a8a;
            --accent-cyan: #06b6d4;
            --accent-green: #10b981;
            --accent-purple: #8b5cf6;
            --accent-amber: #f59e0b;
            --accent-rose: #f43f5e;
            --bg-dark: #0b0f19;
            --bg-card: #131b2e;
            --bg-card-alt: #1a243d;
            --text-main: #f8fafc;
            --text-muted: #94a3b8;
            --border-color: rgba(255, 255, 255, 0.1);
        }

        * {
            box-sizing: border-box;
        }

        body {
            font-family: 'Plus Jakarta Sans', sans-serif;
            background-color: var(--bg-dark);
            color: var(--text-main);
            line-height: 1.6;
            margin: 0;
            padding: 0;
        }

        /* Top Action Bar */
        .manual-top-bar {
            background: rgba(11, 15, 25, 0.95);
            backdrop-filter: blur(12px);
            border-bottom: 1px solid var(--border-color);
            position: sticky;
            top: 0;
            z-index: 1000;
            padding: 0.75rem 2rem;
        }

        .btn-pdf {
            background: linear-gradient(135deg, #e11d48, #be123c);
            color: #ffffff;
            font-weight: 700;
            border: none;
            border-radius: 8px;
            padding: 0.55rem 1.25rem;
            display: inline-flex;
            align-items: center;
            gap: 8px;
            box-shadow: 0 4px 12px rgba(225, 29, 72, 0.35);
            transition: all 0.2s ease;
            text-decoration: none;
            cursor: pointer;
        }

        .btn-pdf:hover {
            background: linear-gradient(135deg, #f43f5e, #e11d48);
            transform: translateY(-2px);
            box-shadow: 0 6px 16px rgba(225, 29, 72, 0.5);
            color: #ffffff;
        }

        .container-manual {
            max-width: 1140px;
            margin: 2rem auto;
            padding: 0 1.5rem;
        }

        /* Hero Header */
        .manual-hero {
            background: linear-gradient(135deg, rgba(67, 100, 247, 0.18) 0%, rgba(139, 92, 246, 0.12) 100%);
            border: 1px solid rgba(67, 100, 247, 0.3);
            border-radius: 20px;
            padding: 2.5rem;
            margin-bottom: 2.5rem;
            position: relative;
            overflow: hidden;
        }

        .manual-hero::before {
            content: '';
            position: absolute;
            top: -50%;
            right: -20%;
            width: 400px;
            height: 400px;
            background: radial-gradient(circle, rgba(139, 92, 246, 0.15), transparent 70%);
            pointer-events: none;
        }

        .manual-badge {
            background: rgba(139, 92, 246, 0.2);
            color: #a78bfa;
            border: 1px solid rgba(139, 92, 246, 0.4);
            padding: 0.35rem 0.85rem;
            border-radius: 30px;
            font-size: 0.8rem;
            font-weight: 700;
            letter-spacing: 1px;
            text-transform: uppercase;
            display: inline-block;
            margin-bottom: 1rem;
        }

        /* Table of Contents */
        .toc-card {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-radius: 16px;
            padding: 1.5rem 2rem;
            margin-bottom: 3rem;
        }

        .toc-list {
            list-style: none;
            padding: 0;
            margin: 0;
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
            gap: 0.75rem;
        }

        .toc-list li a {
            color: var(--text-muted);
            text-decoration: none;
            display: flex;
            align-items: center;
            gap: 10px;
            padding: 0.5rem 0.75rem;
            border-radius: 8px;
            transition: all 0.2s ease;
            font-weight: 500;
        }

        .toc-list li a:hover {
            color: var(--text-main);
            background: rgba(255, 255, 255, 0.05);
            transform: translateX(4px);
        }

        .toc-number {
            background: rgba(67, 100, 247, 0.2);
            color: #60a5fa;
            width: 24px;
            height: 24px;
            border-radius: 6px;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            font-size: 0.75rem;
            font-weight: 700;
        }

        /* Content Sections */
        .manual-section {
            margin-bottom: 3.5rem;
        }

        .section-header {
            display: flex;
            align-items: center;
            gap: 12px;
            margin-bottom: 1.25rem;
            padding-bottom: 0.75rem;
            border-bottom: 2px solid rgba(67, 100, 247, 0.3);
        }

        .section-icon {
            width: 42px;
            height: 42px;
            border-radius: 10px;
            background: linear-gradient(135deg, var(--primary), var(--accent-purple));
            color: #ffffff;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.2rem;
        }

        .section-title {
            font-size: 1.6rem;
            font-weight: 800;
            margin: 0;
            color: #ffffff;
        }

        /* UI Mockups */
        .ui-mockup {
            background: #0f1422;
            border: 1px solid rgba(255, 255, 255, 0.15);
            border-radius: 14px;
            padding: 1.5rem;
            margin: 1.5rem 0;
            box-shadow: 0 15px 35px rgba(0, 0, 0, 0.6);
        }

        .mockup-header {
            font-size: 0.75rem;
            text-transform: uppercase;
            letter-spacing: 1.5px;
            color: var(--text-muted);
            margin-bottom: 1rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            border-bottom: 1px solid rgba(255, 255, 255, 0.08);
            padding-bottom: 0.5rem;
        }

        /* Callout Boxes */
        .callout {
            border-radius: 12px;
            padding: 1.25rem 1.5rem;
            margin: 1.25rem 0;
            border-left: 4px solid;
        }

        .callout-info {
            background: rgba(6, 182, 212, 0.08);
            border-color: var(--accent-cyan);
            color: #e0f2fe;
        }

        .callout-tip {
            background: rgba(16, 185, 129, 0.08);
            border-color: var(--accent-green);
            color: #d1fae5;
        }

        .callout-warning {
            background: rgba(245, 158, 11, 0.08);
            border-color: var(--accent-amber);
            color: #fef3c7;
        }

        .callout-title {
            font-weight: 700;
            margin-bottom: 0.35rem;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        /* Feature Cards Grid */
        .feature-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 1.25rem;
            margin: 1.5rem 0;
        }

        .feature-card {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-radius: 14px;
            padding: 1.5rem;
            transition: all 0.2s ease;
        }

        .feature-card:hover {
            border-color: rgba(67, 100, 247, 0.4);
            transform: translateY(-2px);
        }

        .feature-card h5 {
            color: #ffffff;
            font-weight: 700;
            display: flex;
            align-items: center;
            gap: 10px;
            margin-bottom: 0.75rem;
        }

        /* Tables */
        .table-manual {
            width: 100%;
            border-collapse: separate;
            border-spacing: 0;
            border: 1px solid var(--border-color);
            border-radius: 12px;
            overflow: hidden;
            margin: 1.5rem 0;
        }

        .table-manual th {
            background: var(--bg-card-alt);
            color: #ffffff;
            padding: 0.85rem 1.25rem;
            font-weight: 700;
            font-size: 0.85rem;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            border-bottom: 1px solid var(--border-color);
        }

        .table-manual td {
            background: var(--bg-card);
            padding: 0.85rem 1.25rem;
            font-size: 0.9rem;
            border-bottom: 1px solid var(--border-color);
        }

        .table-manual tr:last-child td {
            border-bottom: none;
        }

        /* Footer */
        .manual-footer {
            border-top: 1px solid var(--border-color);
            padding: 2.5rem 0;
            margin-top: 4rem;
            text-align: center;
            color: var(--text-muted);
            font-size: 0.85rem;
        }

        /* PRINT STYLES FOR PDF EXPORT */
        @media print {
            body {
                background: #ffffff !important;
                color: #0f172a !important;
                font-size: 11pt;
            }

            .manual-top-bar, .btn-pdf, .btn-print-hide {
                display: none !important;
            }

            .container-manual {
                max-width: 100% !important;
                margin: 0 !important;
                padding: 0 !important;
            }

            .manual-hero {
                background: #f8fafc !important;
                border: 1px solid #cbd5e1 !important;
                color: #0f172a !important;
                page-break-after: avoid;
            }

            .manual-section {
                page-break-inside: avoid;
                margin-bottom: 2rem !important;
            }

            .section-title {
                color: #0f172a !important;
            }

            .toc-card, .feature-card, .ui-mockup {
                background: #f8fafc !important;
                border: 1px solid #cbd5e1 !important;
                color: #0f172a !important;
                box-shadow: none !important;
            }

            .table-manual th {
                background: #e2e8f0 !important;
                color: #0f172a !important;
                border-color: #cbd5e1 !important;
            }

            .table-manual td {
                background: #ffffff !important;
                color: #334155 !important;
                border-color: #cbd5e1 !important;
            }

            .callout {
                background: #f1f5f9 !important;
                color: #0f172a !important;
                border-left-width: 4px !important;
            }

            .callout-info { border-color: #0284c7 !important; }
            .callout-tip { border-color: #16a34a !important; }
            .callout-warning { border-color: #d97706 !important; }

            h1, h2, h3, h4, h5 {
                color: #0f172a !important;
            }
        }
    </style>
</head>
<body>

    <!-- Barra de Ações Superior -->
    <header class="manual-top-bar d-flex justify-content-between align-items-center">
        <div class="d-flex align-items-center gap-3">
            <span class="badge bg-primary px-3 py-2 fw-bold" style="letter-spacing: 1px;">IMC-SERVER</span>
            <span class="fw-bold text-white fs-6">Manual do Administrador & Gestor — IMC Central</span>
        </div>
        <div class="d-flex align-items-center gap-2">
            <button class="btn-pdf" onclick="window.print()">
                <i class="fa-solid fa-file-pdf fs-5"></i> Exportar para PDF / Imprimir
            </button>
        </div>
    </header>

    <div class="container-manual">

        <!-- Hero de Abertura -->
        <div class="manual-hero">
            <span class="manual-badge">Documentação Administrativa Master</span>
            <h1 class="fw-extrabold text-white mb-2" style="font-size: 2.2rem;">Manual do Administrador do IMC-Server</h1>
            <p class="text-muted fs-5 mb-0" style="max-width: 850px;">
                Guia completo de governança, parametrização de clientes, filiais, curadoria musical, montagem de grades de áudio 24 horas, veiculação de comerciais e auditoria de transmissão (Proof-of-Play).
            </p>
        </div>

        <!-- Índice Rápido -->
        <div class="toc-card">
            <h5 class="fw-bold text-white mb-3 d-flex align-items-center gap-2">
                <i class="fa-solid fa-list-check text-primary"></i> Módulos do Sistema Central
            </h5>
            <ul class="toc-list">
                <li><a href="#sec-1"><span class="toc-number">1</span> Arquitetura Central & Orquestração</a></li>
                <li><a href="#sec-2"><span class="toc-number">2</span> Clientes & Redes (/tenants)</a></li>
                <li><a href="#sec-3"><span class="toc-number">3</span> Filiais & Lojas (/stores)</a></li>
                <li><a href="#sec-4"><span class="toc-number">4</span> Usuários & Permissões (/users)</a></li>
                <li><a href="#sec-5"><span class="toc-number">5</span> Estilos Musicais & Selo SONIC (/music-styles)</a></li>
                <li><a href="#sec-6"><span class="toc-number">6</span> Grades de Áudio 24h (/grids/audio)</a></li>
                <li><a href="#sec-7"><span class="toc-number">7</span> Comerciais, Avisos & Chimes (/commercials)</a></li>
                <li><a href="#sec-8"><span class="toc-number">8</span> Streamings & Rádios Online (/streamings)</a></li>
                <li><a href="#sec-9"><span class="toc-number">9</span> Monitoramento & Relatórios (/play-history)</a></li>
                <li><a href="#sec-10"><span class="toc-number">10</span> Web Terminal & Infraestrutura (/ws/terminal)</a></li>
            </ul>
        </div>

        <!-- SEÇÃO 1: ARQUITETURA -->
        <section id="sec-1" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-network-wired"></i></div>
                <h2 class="section-title">1. Arquitetura Central & Orquestração</h2>
            </div>
            <p>
                O <strong>IMC-Server</strong> funciona como o cérebro centralizador de toda a rede de rádio indoor. Ele gerencia as diretrizes musicais, horários de funcionamento, veiculação de spots e distribui as configurações automaticamente para as instâncias de playout de cada loja.
            </p>

            <div class="feature-grid">
                <div class="feature-card">
                    <h5><i class="fa-solid fa-cubes text-info"></i> Isolamento por Contêiner Docker</h5>
                    <p class="text-muted small mb-0">
                        Cada filial roda uma instância independente de Liquidsoap e SQLite local. Se uma loja sofrer perda de internet, todas as demais continuam operando normalmente sem qualquer impacto cruzado.
                    </p>
                </div>
                <div class="feature-card">
                    <h5><i class="fa-solid fa-clock text-warning"></i> Sincronização por Timezone (TZ)</h5>
                    <p class="text-muted small mb-0">
                        O servidor respeita o fuso horário geográfico cadastrado em cada loja (ex: <code>America/Sao_Paulo</code>, <code>America/Manaus</code>), garantindo que os blocos comerciais entrem no horário correto de cada região.
                    </p>
                </div>
            </div>
        </section>

        <!-- SEÇÃO 2: TENANTS -->
        <section id="sec-2" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-building"></i></div>
                <h2 class="section-title">2. Clientes & Redes (Tenants)</h2>
            </div>
            <p>
                Acessível em <code>/tenants</code>, permite cadastrar as redes de lojas, marcas ou shoppings clientes atendidos pela sua central.
            </p>
            <table class="table-manual">
                <thead>
                    <tr>
                        <th>Campo</th>
                        <th>Tipo</th>
                        <th>Descrição</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td><strong>Razão Social / Nome Fantasia</strong></td>
                        <td>Texto</td>
                        <td>Nome oficial da marca ou empresa cliente.</td>
                    </tr>
                    <tr>
                        <td><strong>Status (Ativo / Inativo)</strong></td>
                        <td>Toggle</td>
                        <td>Controla se as lojas vinculadas à rede estão autorizadas a transmitir.</td>
                    </tr>
                </tbody>
            </table>
        </section>

        <!-- SEÇÃO 3: STORES -->
        <section id="sec-3" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-store"></i></div>
                <h2 class="section-title">3. Filiais & Lojas (Stores)</h2>
            </div>
            <p>
                Acessível em <code>/stores</code>, gerencia as filiais físicas. Ao criar uma loja, o IMC-Server gera automaticamente o link exclusivo do Web Player (<code>/player/{id}</code>) e inicializa a infraestrutura de transmissão da filial.
            </p>

            <div class="callout callout-info">
                <div class="callout-title"><i class="fa-solid fa-sliders"></i> Atribuição de Grade de Áudio:</div>
                No formulário da loja, o campo <strong>"Grade de Áudio Semanal"</strong> define qual planejamento de 24h a loja seguirá em cada dia da semana (Domingo a Sábado + Feriados).
            </div>
        </section>

        <!-- SEÇÃO 4: USERS -->
        <section id="sec-4" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-users-gear"></i></div>
                <h2 class="section-title">4. Usuários & Perfis de Acesso (RBAC)</h2>
            </div>
            <p>
                Acessível em <code>/users</code>, implementa o controle rigoroso de privilégios de acesso:
            </p>

            <table class="table-manual">
                <thead>
                    <tr>
                        <th>Nível de Permissão</th>
                        <th>ID</th>
                        <th>Escopo de Acesso Permitido</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td><span class="badge bg-danger">Administrador</span></td>
                        <td>1</td>
                        <td>Acesso total a todos os menus, criação de usuários, infraestrutura Docker e terminal.</td>
                    </tr>
                    <tr>
                        <td><span class="badge bg-primary">Programador / Gestor</span></td>
                        <td>2</td>
                        <td>Gestão de catálogo musical, criação de grades, comerciais e relatórios.</td>
                    </tr>
                    <tr>
                        <td><span class="badge bg-secondary">Operador de Loja</span></td>
                        <td>3</td>
                        <td>Acesso restrito exclusivamente ao Web Player da sua loja (<code>/player/{store_id}</code>).</td>
                    </tr>
                </tbody>
            </table>
        </section>

        <!-- SEÇÃO 5: MUSIC STYLES -->
        <section id="sec-5" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-headphones"></i></div>
                <h2 class="section-title">5. Estilos Musicais & Selo de Qualidade SONIC</h2>
            </div>
            <p>
                Acessível em <code>/music-styles</code>, organiza o acervo fonográfico da rádio:
            </p>

            <div class="feature-grid">
                <div class="feature-card">
                    <h5><i class="fa-solid fa-wand-magic-sparkles text-info"></i> Selo SONIC de Alta Fidelidade</h5>
                    <p class="text-muted small mb-0">
                        Cada faixa sincronizada passa pela esteira de normalização automática (-14 LUFS) e cálculo de pontos inteligentes de entrada e saída (CUE IN / CUE OUT), eliminando silêncios no início/fim das músicas.
                    </p>
                </div>
                <div class="feature-card">
                    <h5><i class="fa-solid fa-tower-broadcast text-cyan"></i> Suporte a Estilos STREAM</h5>
                    <p class="text-muted small mb-0">
                        Permite criar estilos que apontam para URLs de rádios web online (Icecast/Shoutcast), permitindo mesclar estilos locais e streaming na mesma grade semanal.
                    </p>
                </div>
            </div>
        </section>

        <!-- SEÇÃO 6: AUDIO GRIDS -->
        <section id="sec-6" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-calendar-days"></i></div>
                <h2 class="section-title">6. Grades de Áudio 24 Horas</h2>
            </div>
            <p>
                Acessível em <code>/grids/audio</code>, permite programar a linha do tempo sonora das 00:00 às 23:59:
            </p>
            <ul>
                <li><strong>Blocos Musicais (5m, 15m, 60m):</strong> Selecione um ou mais estilos para rotacionar no período.</li>
                <li><strong>Blocos Comerciais:</strong> Defina os intervalos de veiculação e limites de duração dos anúncios.</li>
                <li><strong>Replicação Rápida:</strong> Botões de preenchimento em lote para replicar o mesmo estilo musical ao longo de várias horas.</li>
            </ul>
        </section>

        <!-- SEÇÃO 7: COMMERCIALS -->
        <section id="sec-7" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-bullhorn"></i></div>
                <h2 class="section-title">7. Comerciais, Avisos & Chimes (Din-Don)</h2>
            </div>
            <p>
                Acessível em <code>/commercials</code>, <code>/voice-packs</code> e <code>/chimes</code>:
            </p>
            <table class="table-manual">
                <thead>
                    <tr>
                        <th>Tipo de Mídia</th>
                        <th>Comportamento no Playout</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td><strong>Spot Comercial / Oferta</strong></td>
                        <td>Aumenta automaticamente o volume do mixer em +20% para garantir clareza na locução.</td>
                    </tr>
                    <tr>
                        <td><strong>Avisos com Pacote de Voz</strong></td>
                        <td>Executa a chamada de veículos com voz neural realista e vinheta de abertura.</td>
                    </tr>
                    <tr>
                        <td><strong>Hora Certa & Temperatura</strong></td>
                        <td>Sintetiza e anuncia a hora e clima local nos blocos programados.</td>
                    </tr>
                </tbody>
            </table>
        </section>

        <!-- SEÇÃO 8: STREAMINGS -->
        <section id="sec-8" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-tower-broadcast"></i></div>
                <h2 class="section-title">8. Streamings & Rádios Online</h2>
            </div>
            <p>
                Acessível em <code>/streamings</code>, oferece um painel central de monitoramento em tempo real:
            </p>
            <ul>
                <li><strong>Teste de Conectividade Automático:</strong> Testa as URLs das emissoras e exibe a badge verde <code>OK (200)</code>.</li>
                <li><strong>Mapeamento de Lojas no Ar:</strong> Identifica quais filiais estão conectadas a cada estação no minuto atual.</li>
                <li><strong>Pré-escuta no Painel:</strong> Permite ao gestor ouvir a estação diretamente no navegador antes de colocá-la na grade.</li>
            </ul>
        </section>

        <!-- SEÇÃO 9: AUDITORIA -->
        <section id="sec-9" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-chart-pie"></i></div>
                <h2 class="section-title">9. Monitoramento & Relatórios (Proof-of-Play)</h2>
            </div>
            <p>
                Acessível em <code>/play-history</code> e <code>/radios/status</code>:
            </p>
            <div class="callout callout-tip">
                <div class="callout-title"><i class="fa-solid fa-file-invoice"></i> Comprovante Oficial de Veiculação:</div>
                Gere relatórios auditáveis com data, hora, segundo exato e status de cada comercial executado nas lojas para apresentação a anunciantes e patrocinadores.
            </div>
        </section>

        <!-- SEÇÃO 10: TERMINAL -->
        <section id="sec-10" class="manual-section">
            <div class="section-header">
                <div class="section-icon"><i class="fa-solid fa-terminal"></i></div>
                <h2 class="section-title">10. Web Terminal & Infraestrutura</h2>
            </div>
            <p>
                Acessível em <code>/ws/terminal</code>, permite aos administradores abrir sessões de console PTY direto pelo navegador para inspecionar contêineres Docker, consultar logs do sistema e verificar a saúde dos serviços de transmissão.
            </p>
        </section>

        <!-- Rodapé do Manual -->
        <footer class="manual-footer">
            <div class="fw-bold text-white mb-1">SISTEMA IMC — CENTRAL DE GESTÃO E ARQUITETURA</div>
            <div>Versão da Plataforma: <strong>v1.5.0 (SONIC Edition)</strong> • Documento Homologado para Administradores</div>
        </footer>

    </div>

    <!-- Bootstrap 5 JS -->
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>"""

    manual_docs_path = os.path.join(docs_dir, "manual_server.html")
    with open(manual_docs_path, "w", encoding="utf-8") as f:
        f.write(html_content)

    templates_dir = os.path.join(root_dir, "web", "templates")
    if os.path.exists(templates_dir):
        manual_tmpl_path = os.path.join(templates_dir, "manual_server.html")
        with open(manual_tmpl_path, "w", encoding="utf-8") as f:
            f.write(html_content)

    print(f"[SUCCESS] Manual HTML do Servidor gerado em: {manual_docs_path}")

def generate_help_hub(root_dir, docs_dir):
    html_content = """<!DOCTYPE html>
<html lang="pt-br">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Central de Ajuda & Manuais - IMC</title>
    <!-- Bootstrap 5 CSS & FontAwesome Icons -->
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/css/bootstrap.min.css" rel="stylesheet">
    <link href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css" rel="stylesheet">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@300;400;500;600;700;800&display=swap" rel="stylesheet">

    <style>
        :root {
            --primary: #4364f7;
            --accent-cyan: #06b6d4;
            --accent-green: #10b981;
            --accent-purple: #8b5cf6;
            --bg-dark: #0b0f19;
            --bg-card: #131b2e;
            --text-main: #f8fafc;
            --text-muted: #94a3b8;
            --border-color: rgba(255, 255, 255, 0.1);
        }

        body {
            font-family: 'Plus Jakarta Sans', sans-serif;
            background-color: var(--bg-dark);
            color: var(--text-main);
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            justify-content: space-between;
        }

        .hub-container {
            max-width: 1000px;
            margin: 4rem auto;
            padding: 0 1.5rem;
        }

        .hub-card {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            border-radius: 20px;
            padding: 2.5rem;
            transition: all 0.3s ease;
            display: flex;
            flex-direction: column;
            justify-content: space-between;
            height: 100%;
        }

        .hub-card:hover {
            transform: translateY(-6px);
            border-color: rgba(67, 100, 247, 0.5);
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.5);
        }

        .hub-icon {
            width: 60px;
            height: 60px;
            border-radius: 16px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.75rem;
            margin-bottom: 1.5rem;
        }

        .btn-hub {
            border-radius: 10px;
            font-weight: 700;
            padding: 0.75rem 1.25rem;
            display: inline-flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
            transition: all 0.2s ease;
            text-decoration: none;
        }
    </style>
</head>
<body>

    <div class="hub-container">
        <div class="text-center mb-5">
            <span class="badge bg-primary px-3 py-2 fw-bold text-uppercase mb-2" style="letter-spacing: 1.5px;">Documentação Oficial</span>
            <h1 class="fw-extrabold text-white">Central de Ajuda & Manuais do Sistema IMC</h1>
            <p class="text-muted fs-5">Selecione o manual desejado para consultar online ou exportar diretamente em PDF.</p>
        </div>

        <div class="row g-4">
            <!-- Card 1: Manual do Servidor -->
            <div class="col-md-6">
                <div class="hub-card">
                    <div>
                        <div class="hub-icon" style="background: rgba(139, 92, 246, 0.15); color: #a78bfa; border: 1px solid rgba(139, 92, 246, 0.3);">
                            <i class="fa-solid fa-server"></i>
                        </div>
                        <h3 class="fw-bold text-white mb-2">Manual do Administrador (IMC-Server)</h3>
                        <p class="text-muted small mb-4">
                            Guia completo de governança para administradores e gestores: cadastro de filiais, gestão de grades de áudio 24h, inserção de comerciais, monitoramento de streamings e auditoria de transmissão.
                        </p>
                    </div>
                    <div class="d-flex gap-2">
                        <a href="/manual/server" class="btn btn-primary btn-hub flex-grow-1">
                            <i class="fa-solid fa-book-open"></i> Abrir Manual do Servidor
                        </a>
                    </div>
                </div>
            </div>

            <!-- Card 2: Manual do Web Player -->
            <div class="col-md-6">
                <div class="hub-card">
                    <div>
                        <div class="hub-icon" style="background: rgba(6, 182, 212, 0.15); color: var(--accent-cyan); border: 1px solid rgba(6, 182, 212, 0.3);">
                            <i class="fa-solid fa-radio"></i>
                        </div>
                        <h3 class="fw-bold text-white mb-2">Manual do Operador (Web Player)</h3>
                        <p class="text-muted small mb-4">
                            Guia prático para gerentes e atendentes de loja: como iniciar a transmissão, controle de volume, chamadas de estacionamento por placa com ducking e solução rápida de dúvidas.
                        </p>
                    </div>
                    <div class="d-flex gap-2">
                        <a href="/manual/player" class="btn btn-info btn-hub flex-grow-1 text-dark">
                            <i class="fa-solid fa-headphones"></i> Abrir Manual do Player
                        </a>
                    </div>
                </div>
            </div>
        </div>

        <div class="text-center mt-5">
            <a href="/" class="btn btn-outline-secondary btn-sm px-4">
                <i class="fa-solid fa-arrow-left me-1"></i> Voltar ao Painel Central
            </a>
        </div>
    </div>

    <footer class="text-center py-4 border-top border-secondary text-muted small">
        SISTEMA IMC — IN-STORE MUSIC CONTROLLER • v1.5.0
    </footer>

</body>
</html>"""

    help_docs_path = os.path.join(docs_dir, "help.html")
    with open(help_docs_path, "w", encoding="utf-8") as f:
        f.write(html_content)

    templates_dir = os.path.join(root_dir, "web", "templates")
    if os.path.exists(templates_dir):
        help_tmpl_path = os.path.join(templates_dir, "help.html")
        with open(help_tmpl_path, "w", encoding="utf-8") as f:
            f.write(html_content)

    print(f"[SUCCESS] Hub de Ajuda gerado em: {help_docs_path}")

if __name__ == "__main__":
    main()



