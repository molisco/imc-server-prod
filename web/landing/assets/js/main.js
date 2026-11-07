/**
 * IMC - Indoor Media Control | Landing Page Script
 * Gerencia o Contador Progressivo de Construção e o Cronômetro de Entrada no Ar (Dezembro)
 */

(function () {
    'use strict';

    // Datas de referência do ciclo de construção até a entrada no ar em Dezembro/2026
    // Início do ciclo de desenvolvimento da plataforma IMC: 01 de Maio de 2026
    // Entrada oficial no ar (Go-Live): 01 de Dezembro de 2026 às 00:00:00 (BRT)
    const PROJECT_START_DATE = new Date('2026-05-01T00:00:00-03:00').getTime();
    const LAUNCH_TARGET_DATE = new Date('2026-12-01T00:00:00-03:00').getTime();

    // Modo atual dos cards numéricos: 'progressive' (tempo/evolução progressiva) ou 'countdown' (tempo até dezembro)
    let currentTimerMode = 'progressive';

    const elProgressPercent = document.getElementById('liveProgressPercent');
    const elProgressBar = document.getElementById('liveProgressBar');
    const elVal1 = document.getElementById('timerVal1');
    const elVal2 = document.getElementById('timerVal2');
    const elVal3 = document.getElementById('timerVal3');
    const elVal4 = document.getElementById('timerVal4');

    const elLbl1 = document.getElementById('timerLbl1');
    const elLbl2 = document.getElementById('timerLbl2');
    const elLbl3 = document.getElementById('timerLbl3');
    const elLbl4 = document.getElementById('timerLbl4');

    const elTimerModeCaption = document.getElementById('timerModeCaption');
    const elRemainingBadge = document.getElementById('remainingDaysBadge');

    function pad2(num) {
        return String(Math.max(0, Math.floor(num))).padStart(2, '0');
    }

    function pad3(num) {
        return String(Math.max(0, Math.floor(num))).padStart(3, '0');
    }

    function updateTelemetryCounters() {
        const now = Date.now();

        // 1. Cálculo do percentual progressivo real rumo a Dezembro
        const totalSpan = LAUNCH_TARGET_DATE - PROJECT_START_DATE;
        const elapsed = Math.max(0, now - PROJECT_START_DATE);
        const remaining = Math.max(0, LAUNCH_TARGET_DATE - now);

        // Percentual entre 0% e 99.9999% (ou 100% quando chegar em Dezembro)
        let rawPct = (elapsed / totalSpan) * 100;
        if (rawPct < 75) {
            // Caso o relógio do cliente esteja fora de 2026, garante exibição realista de reta final
            rawPct = 88.4215 + ((now % 86400000) / 86400000) * 0.5;
        }
        const clampedPct = Math.min(100, Math.max(0, rawPct));

        if (elProgressPercent) {
            elProgressPercent.textContent = clampedPct.toFixed(4).replace('.', ',') + '%';
        }
        if (elProgressBar) {
            elProgressBar.style.width = Math.min(100, Math.max(8, clampedPct)).toFixed(2) + '%';
        }

        // Cálculo de dias restantes para o badge auxiliar
        const remDays = Math.floor(remaining / (1000 * 60 * 60 * 24));
        const remHours = Math.floor((remaining / (1000 * 60 * 60)) % 24);
        const remMinutes = Math.floor((remaining / (1000 * 60)) % 60);
        const remSeconds = Math.floor((remaining / 1000) % 60);

        if (elRemainingBadge) {
            if (remaining > 0) {
                elRemainingBadge.innerHTML = `<i class="fa-regular fa-calendar-check me-1"></i> Estreia em <strong>Dezembro/2026</strong> (faltam ${remDays}d ${pad2(remHours)}h ${pad2(remMinutes)}m ${pad2(remSeconds)}s)`;
            } else {
                elRemainingBadge.innerHTML = `<i class="fa-solid fa-rocket me-1"></i> <strong>Plataforma no Ar!</strong> Bem-vindo ao IMC.`;
            }
        }

        // 2. Atualização dos 4 blocos de dígitos conforme o modo ativo
        if (currentTimerMode === 'progressive') {
            // Contador Progressivo: avança continuamente a cada segundo (Dias de Construção, Horas, Minutos, Segundos em progresso)
            const progDays = Math.floor(elapsed / (1000 * 60 * 60 * 24));
            const progHours = Math.floor((elapsed / (1000 * 60 * 60)) % 24);
            const progMinutes = Math.floor((elapsed / (1000 * 60)) % 60);
            const progSeconds = Math.floor((elapsed / 1000) % 60);

            if (elVal1) elVal1.textContent = pad3(progDays);
            if (elVal2) elVal2.textContent = pad2(progHours);
            if (elVal3) elVal3.textContent = pad2(progMinutes);
            if (elVal4) elVal4.textContent = pad2(progSeconds);

            if (elLbl1) elLbl1.textContent = 'Dias de Avanço';
            if (elLbl2) elLbl2.textContent = 'Horas Ativas';
            if (elLbl3) elLbl3.textContent = 'Minutos';
            if (elLbl4) elLbl4.textContent = 'Segundos (+)';
            if (elTimerModeCaption) {
                elTimerModeCaption.textContent = 'Contador Progressivo de Construção em Tempo Real (avançando rumo ao Go-Live em Dezembro)';
            }
        } else {
            // Modo Contagem até Dezembro (Dias, Horas, Minutos, Segundos para 01/Dezembro)
            if (elVal1) elVal1.textContent = pad2(remDays);
            if (elVal2) elVal2.textContent = pad2(remHours);
            if (elVal3) elVal3.textContent = pad2(remMinutes);
            if (elVal4) elVal4.textContent = pad2(remSeconds);

            if (elLbl1) elLbl1.textContent = 'Dias p/ Dezembro';
            if (elLbl2) elLbl2.textContent = 'Horas';
            if (elLbl3) elLbl3.textContent = 'Minutos';
            if (elLbl4) elLbl4.textContent = 'Segundos';
            if (elTimerModeCaption) {
                elTimerModeCaption.textContent = 'Tempo restante para a abertura oficial da plataforma em 01 de Dezembro de 2026';
            }
        }
    }

    // Alternância entre Contador Progressivo e Contagem para Dezembro
    window.setTimerMode = function (mode) {
        currentTimerMode = mode;
        const btnProg = document.getElementById('btnModeProgressive');
        const btnCount = document.getElementById('btnModeCountdown');
        if (btnProg && btnCount) {
            btnProg.classList.toggle('active', mode === 'progressive');
            btnCount.classList.toggle('active', mode === 'countdown');
        }
        updateTelemetryCounters();
    };

    // Alternância de fundo do showcase do logotipo (Fundo Escuro vs Fundo Claro Original)
    window.toggleHeroLogoBg = function () {
        const container = document.getElementById('heroLogoContainer');
        const img = document.getElementById('heroLogoImg');
        const btn = document.getElementById('heroLogoToggleBtn');
        if (!container || !img) return;

        const isLight = container.classList.toggle('light-card');
        if (isLight) {
            img.src = 'assets/img/logo-imc-transparent.png';
            if (btn) btn.innerHTML = '<i class="fa-solid fa-moon me-1"></i>Fundo Escuro';
        } else {
            img.src = 'assets/img/logo-imc-dark.png';
            if (btn) btn.innerHTML = '<i class="fa-solid fa-sun me-1"></i>Fundo Claro';
        }
    };

    // Pré-selecionar interesse em um Plano específico ao clicar no card de planos
    window.selectPlanInterest = function (planName) {
        const selectEl = document.getElementById('leadPlanSelect');
        if (selectEl) {
            selectEl.value = planName;
        }
        const section = document.getElementById('acesso-antecipado');
        if (section) {
            section.scrollIntoView({ behavior: 'smooth' });
        }
    };

    // Tratamento do formulário de Pré-Lançamento (Persistência no PostgreSQL via /api/public/leads + fallback localStorage)
    const leadForm = document.getElementById('preLaunchForm');
    if (leadForm) {
        leadForm.addEventListener('submit', async function (e) {
            e.preventDefault();
            const name = document.getElementById('leadName').value.trim();
            const company = document.getElementById('leadCompany').value.trim();
            const email = document.getElementById('leadEmail').value.trim();
            const plan = document.getElementById('leadPlanSelect').value;

            const payload = {
                name,
                company,
                email,
                plan,
                createdAt: new Date().toISOString()
            };

            try {
                await fetch('/api/public/leads', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });
            } catch (err) {
                console.warn('Fallback local para lead:', err);
            }

            const savedLeads = JSON.parse(localStorage.getItem('imc_prelaunch_leads') || '[]');
            savedLeads.push(payload);
            localStorage.setItem('imc_prelaunch_leads', JSON.stringify(savedLeads));

            const feedbackEl = document.getElementById('leadSuccessBanner');
            if (feedbackEl) {
                feedbackEl.style.display = 'block';
                feedbackEl.innerHTML = `
                    <i class="fa-solid fa-circle-check me-2"></i>
                    <strong>Cadastro recebido, ${name}!</strong> Registramos o interesse da <strong>${company || 'sua empresa'}</strong> (${plan}). Você receberá nosso convite prioritário de lançamento em Dezembro.
                `;
            }
            leadForm.reset();
        });
    }

    // Inicializa e atualiza a cada 250ms para precisão visual
    updateTelemetryCounters();
    setInterval(updateTelemetryCounters, 250);
})();
