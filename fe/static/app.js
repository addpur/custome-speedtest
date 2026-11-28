/**
 * Custome Speedtest (ct-speedtest)
 * High-Performance Smooth 60 FPS HTML5 Speed Test Engine
 * 
 * Features:
 * - 60 FPS Physics / LERP Animation for Gauge (Speedtest.net style smooth acceleration)
 * - Rolling Window (600ms) + Exponential Moving Average (EMA) to eliminate TCP packet burst jitter
 * - Smooth Bezier Bandwidth Chart without jagged teeth or flatline drops
 * - Real-time continuous Upload Progress via xhr.upload.onprogress (no chunky stalls)
 * - Zero-Memory-Leak Fetch stream reading for Download
 * - True Latency & RFC 1889 Jitter
 * - Light / Dark Theme toggle with persistent preference
 */

(function () {
    'use strict';

    // State & Configurations
    const state = {
        isRunning: false,
        phase: 'idle', // 'idle' | 'ping' | 'download' | 'upload' | 'completed'
        targetSpeed: 0,     // Network speed calculated via rolling window & EMA
        displayedSpeed: 0,  // Visually animated speed on gauge at 60 FPS
        peakSpeed: 0,
        chartData: [],
        dlChartData: [],
        ulChartData: [],
        dlPeakSpeed: 0,
        ulPeakSpeed: 0,
        maxChartPoints: 45,
        animFrameId: null,
        activeXHRs: [],
        downloadAbortController: null,
        results: {
            ping: 0,
            jitter: 0,
            download: 0,
            downloadBytes: 0,
            upload: 0,
            uploadBytes: 0
        },
        config: {
            duration: 10,
            parallel: 8,
            overhead: 1.0,
            autoStart: false
        }
    };

    // Dynamic Byte Unit Formatter (B, KB, MB, GB, TB, PB)
    function formatBytes(bytes) {
        if (!bytes || bytes <= 0) return '0 B';
        const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
        const k = 1024;
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        const idx = Math.min(Math.max(i, 0), units.length - 1);
        const val = bytes / Math.pow(k, idx);
        return `${val >= 100 || idx === 0 ? val.toFixed(0) : val.toFixed(1)} ${units[idx]}`;
    }

    // DOM Elements
    const elements = {
        statusBadge: document.getElementById('status-badge'),
        statusText: document.getElementById('status-text'),
        speedDisplay: document.getElementById('speed-display'),
        gaugePhaseName: document.getElementById('gauge-phase-name'),
        testProgressBar: document.getElementById('test-progress-bar'),
        valPing: document.getElementById('val-ping'),
        valJitter: document.getElementById('val-jitter'),
        valDownload: document.getElementById('val-download'),
        valDownloadBytes: document.getElementById('val-download-bytes'),
        valUpload: document.getElementById('val-upload'),
        valUploadBytes: document.getElementById('val-upload-bytes'),
        btnStart: document.getElementById('btn-start'),
        btnStress: document.getElementById('btn-stress'),
        gaugeCanvas: document.getElementById('gauge-canvas'),
        liveChart: document.getElementById('live-chart'),
        chartDlCanvas: document.getElementById('live-dl-chart'),
        chartUlCanvas: document.getElementById('live-ul-chart'),
        chartDlPeak: document.getElementById('chart-dl-peak'),
        chartUlPeak: document.getElementById('chart-ul-peak'),
        chartMaxIndicator: document.getElementById('chart-max-indicator'),
        infoStreams: document.getElementById('info-streams'),
        infoDuration: document.getElementById('info-duration'),
        // Modals & Controls
        btnThemeToggle: document.getElementById('btn-theme-toggle'),
        themeIconSun: document.getElementById('theme-icon-sun'),
        themeIconMoon: document.getElementById('theme-icon-moon'),
        btnSettingsToggle: document.getElementById('btn-settings-toggle'),
        btnHistoryToggle: document.getElementById('btn-history-toggle'),
        settingsModal: document.getElementById('settings-modal'),
        historyModal: document.getElementById('history-modal'),
        btnCloseSettings: document.getElementById('btn-close-settings'),
        btnCloseHistory: document.getElementById('btn-close-history'),
        btnSaveSettings: document.getElementById('btn-save-settings'),
        btnClearHistory: document.getElementById('btn-clear-history'),
        inputDuration: document.getElementById('input-duration'),
        inputParallel: document.getElementById('input-parallel'),
        inputOverhead: document.getElementById('input-overhead'),
        lblDurationVal: document.getElementById('lbl-duration-val'),
        lblParallelVal: document.getElementById('lbl-parallel-val'),
        lblOverheadVal: document.getElementById('lbl-overhead-val'),
        chkAutoStart: document.getElementById('chk-auto-start'),
        historyTbody: document.getElementById('history-tbody'),
        cardPing: document.getElementById('metric-ping-card'),
        cardDownload: document.getElementById('metric-download-card'),
        cardUpload: document.getElementById('metric-upload-card')
    };

    // Canvas Contexts
    const gaugeCtx = elements.gaugeCanvas ? elements.gaugeCanvas.getContext('2d') : null;
    const dlChartCtx = elements.chartDlCanvas ? elements.chartDlCanvas.getContext('2d') : null;
    const ulChartCtx = elements.chartUlCanvas ? elements.chartUlCanvas.getContext('2d') : null;
    const chartCtx = elements.liveChart ? elements.liveChart.getContext('2d') : null;

    function setupCanvas(canvas, ctx) {
        if (!canvas || !ctx) return { width: 340, height: 85 };
        const dpr = window.devicePixelRatio || 1;
        const rect = canvas.getBoundingClientRect();
        const w = rect.width > 0 ? rect.width : (canvas.width || 340);
        const h = rect.height > 0 ? rect.height : (canvas.height || 85);
        canvas.width = w * dpr;
        canvas.height = h * dpr;
        ctx.scale(dpr, dpr);
        return { width: w, height: h };
    }

    let gaugeDim = { width: 380, height: 230 };
    let dlChartDim = { width: 340, height: 85 };
    let ulChartDim = { width: 340, height: 85 };
    let chartDim = { width: 700, height: 90 };

    function resizeCanvases() {
        if (elements.gaugeCanvas && gaugeCtx) {
            gaugeDim = setupCanvas(elements.gaugeCanvas, gaugeCtx);
            drawGauge(state.displayedSpeed);
        }
        if (elements.chartDlCanvas && dlChartCtx) {
            dlChartDim = setupCanvas(elements.chartDlCanvas, dlChartCtx);
            drawDownloadChart();
        }
        if (elements.chartUlCanvas && ulChartCtx) {
            ulChartDim = setupCanvas(elements.chartUlCanvas, ulChartCtx);
            drawUploadChart();
        }
        if (elements.liveChart && chartCtx) {
            chartDim = setupCanvas(elements.liveChart, chartCtx);
            drawChart();
        }
    }

    // 60 FPS Gauge Animation Loop (LERP Easing)
    function startGaugeAnimation() {
        function frame() {
            if (state.isRunning || Math.abs(state.targetSpeed - state.displayedSpeed) > 0.1) {
                // Smooth LERP towards target speed (smooth acceleration like Speedtest.net)
                const easing = 0.085;
                state.displayedSpeed += (state.targetSpeed - state.displayedSpeed) * easing;

                drawGauge(state.displayedSpeed);
                elements.speedDisplay.textContent = state.displayedSpeed >= 100
                    ? state.displayedSpeed.toFixed(0)
                    : state.displayedSpeed.toFixed(1);

                state.animFrameId = requestAnimationFrame(frame);
            } else {
                state.animFrameId = null;
            }
        }
        if (!state.animFrameId) {
            state.animFrameId = requestAnimationFrame(frame);
        }
    }

    function stopGaugeAnimation(finalSpeed) {
        if (state.animFrameId) {
            cancelAnimationFrame(state.animFrameId);
            state.animFrameId = null;
        }
        state.targetSpeed = finalSpeed;
        state.displayedSpeed = finalSpeed;
        drawGauge(finalSpeed);
        elements.speedDisplay.textContent = finalSpeed >= 100 ? finalSpeed.toFixed(0) : finalSpeed.toFixed(1);
    }

    // Dynamic Speedometer Gauge Drawing
    function drawGauge(speed) {
        const { width, height } = gaugeDim;
        gaugeCtx.clearRect(0, 0, width, height);

        const centerX = width / 2;
        const centerY = height - 25;
        const radius = Math.min(centerX - 25, centerY - 15);
        const startAngle = Math.PI * 0.85;
        const endAngle = Math.PI * 2.15;
        const totalAngle = endAngle - startAngle;

        const isLight = document.body.classList.contains('light-mode');

        // Dynamic Scale: smoothly adapts as speed climbs
        let maxGauge = 100;
        if (speed > 2500) maxGauge = 5000;
        else if (speed > 1000) maxGauge = 2500;
        else if (speed > 500) maxGauge = 1000;
        else if (speed > 200) maxGauge = 500;
        else if (speed > 100) maxGauge = 250;
        else if (speed > 50) maxGauge = 100;
        else if (speed > 20) maxGauge = 50;
        else maxGauge = 25;

        // Background Track
        gaugeCtx.beginPath();
        gaugeCtx.arc(centerX, centerY, radius, startAngle, endAngle);
        gaugeCtx.strokeStyle = isLight ? 'rgba(0, 0, 0, 0.08)' : 'rgba(255, 255, 255, 0.08)';
        gaugeCtx.lineWidth = 14;
        gaugeCtx.lineCap = 'round';
        gaugeCtx.stroke();

        // Tick Marks
        const ticks = 10;
        for (let i = 0; i <= ticks; i++) {
            const angle = startAngle + (i / ticks) * totalAngle;
            const rInner = radius - 14;
            const rOuter = radius - 8;
            const x1 = centerX + Math.cos(angle) * rInner;
            const y1 = centerY + Math.sin(angle) * rInner;
            const x2 = centerX + Math.cos(angle) * rOuter;
            const y2 = centerY + Math.sin(angle) * rOuter;

            gaugeCtx.beginPath();
            gaugeCtx.moveTo(x1, y1);
            gaugeCtx.lineTo(x2, y2);
            gaugeCtx.strokeStyle = i % 2 === 0
                ? (isLight ? 'rgba(0, 0, 0, 0.35)' : 'rgba(255, 255, 255, 0.25)')
                : (isLight ? 'rgba(0, 0, 0, 0.15)' : 'rgba(255, 255, 255, 0.12)');
            gaugeCtx.lineWidth = i % 2 === 0 ? 2 : 1;
            gaugeCtx.stroke();
        }

        // Active Arc
        const fraction = Math.min(Math.max(speed / maxGauge, 0), 1);
        const currentAngle = startAngle + fraction * totalAngle;

        if (fraction > 0) {
            const grad = gaugeCtx.createLinearGradient(0, centerY, width, centerY);
            grad.addColorStop(0, '#06b6d4');
            grad.addColorStop(0.5, '#6366f1');
            grad.addColorStop(1, '#a855f7');

            gaugeCtx.beginPath();
            gaugeCtx.arc(centerX, centerY, radius, startAngle, currentAngle);
            gaugeCtx.strokeStyle = grad;
            gaugeCtx.lineWidth = 14;
            gaugeCtx.lineCap = 'round';
            gaugeCtx.shadowColor = '#06b6d4';
            gaugeCtx.shadowBlur = 15;
            gaugeCtx.stroke();
            gaugeCtx.shadowBlur = 0;
        }

        // Needle Glow Tip
        const tipX = centerX + Math.cos(currentAngle) * radius;
        const tipY = centerY + Math.sin(currentAngle) * radius;
        gaugeCtx.beginPath();
        gaugeCtx.arc(tipX, tipY, 7, 0, Math.PI * 2);
        gaugeCtx.fillStyle = '#ffffff';
        gaugeCtx.shadowColor = '#06b6d4';
        gaugeCtx.shadowBlur = 12;
        gaugeCtx.fill();
        gaugeCtx.shadowBlur = 0;
    }

    // Generic Smooth Bezier Bandwidth Chart Renderer
    function renderCurveChart(ctx, dim, data, strokeColor, gradColor) {
        if (!ctx || !dim) return;
        const { width, height } = dim;
        ctx.clearRect(0, 0, width, height);

        if (data.length < 2) return;

        const maxVal = Math.max(...data, 10) * 1.15; // 15% headroom
        const stepX = width / (state.maxChartPoints - 1);
        const startIdx = state.maxChartPoints - data.length;

        // Area Gradient Fill under Curve
        ctx.beginPath();
        ctx.moveTo(startIdx * stepX, height);

        for (let i = 0; i < data.length; i++) {
            const x = (startIdx + i) * stepX;
            const y = height - (data[i] / maxVal) * (height - 12) - 6;

            if (i === 0) {
                ctx.lineTo(x, y);
            } else {
                const prevX = (startIdx + i - 1) * stepX;
                const prevY = height - (data[i - 1] / maxVal) * (height - 12) - 6;
                const cpX1 = prevX + (x - prevX) / 2;
                const cpY1 = prevY;
                const cpX2 = prevX + (x - prevX) / 2;
                const cpY2 = y;
                ctx.bezierCurveTo(cpX1, cpY1, cpX2, cpY2, x, y);
            }
        }

        const lastX = (startIdx + data.length - 1) * stepX;
        ctx.lineTo(lastX, height);
        ctx.closePath();

        const grad = ctx.createLinearGradient(0, 0, 0, height);
        grad.addColorStop(0, gradColor);
        grad.addColorStop(1, 'rgba(0, 0, 0, 0)');
        ctx.fillStyle = grad;
        ctx.fill();

        // Stroke Outline Line with Glow
        ctx.beginPath();
        for (let i = 0; i < data.length; i++) {
            const x = (startIdx + i) * stepX;
            const y = height - (data[i] / maxVal) * (height - 12) - 6;

            if (i === 0) {
                ctx.moveTo(x, y);
            } else {
                const prevX = (startIdx + i - 1) * stepX;
                const prevY = height - (data[i - 1] / maxVal) * (height - 12) - 6;
                const cpX1 = prevX + (x - prevX) / 2;
                const cpY1 = prevY;
                const cpX2 = prevX + (x - prevX) / 2;
                const cpY2 = y;
                ctx.bezierCurveTo(cpX1, cpY1, cpX2, cpY2, x, y);
            }
        }
        ctx.strokeStyle = strokeColor;
        ctx.lineWidth = 3;
        ctx.shadowColor = strokeColor;
        ctx.shadowBlur = 10;
        ctx.stroke();
        ctx.shadowBlur = 0;
    }

    function drawDownloadChart() {
        if (dlChartCtx) {
            renderCurveChart(dlChartCtx, dlChartDim, state.dlChartData, '#06b6d4', 'rgba(6, 182, 212, 0.40)');
        }
    }

    function drawUploadChart() {
        if (ulChartCtx) {
            renderCurveChart(ulChartCtx, ulChartDim, state.ulChartData, '#a855f7', 'rgba(139, 92, 246, 0.40)');
        }
    }

    function drawChart() {
        drawDownloadChart();
        drawUploadChart();
        if (chartCtx) {
            renderCurveChart(chartCtx, chartDim, state.chartData, state.phase === 'upload' ? '#a855f7' : '#06b6d4', state.phase === 'upload' ? 'rgba(139, 92, 246, 0.40)' : 'rgba(6, 182, 212, 0.40)');
        }
    }

    function pushDownloadPoint(val) {
        state.dlChartData.push(val);
        if (state.dlChartData.length > state.maxChartPoints) {
            state.dlChartData.shift();
        }
        if (val > state.dlPeakSpeed) {
            state.dlPeakSpeed = val;
            if (elements.chartDlPeak) elements.chartDlPeak.textContent = `Peak: ${val.toFixed(1)} Mbps`;
        }
        pushChartPoint(val);
        drawDownloadChart();
    }

    function pushUploadPoint(val) {
        state.ulChartData.push(val);
        if (state.ulChartData.length > state.maxChartPoints) {
            state.ulChartData.shift();
        }
        if (val > state.ulPeakSpeed) {
            state.ulPeakSpeed = val;
            if (elements.chartUlPeak) elements.chartUlPeak.textContent = `Peak: ${val.toFixed(1)} Mbps`;
        }
        pushChartPoint(val);
        drawUploadChart();
    }

    function pushChartPoint(val) {
        state.chartData.push(val);
        if (state.chartData.length > state.maxChartPoints) {
            state.chartData.shift();
        }
        if (val > state.peakSpeed) {
            state.peakSpeed = val;
            if (elements.chartMaxIndicator) elements.chartMaxIndicator.textContent = `Peak: ${val.toFixed(1)} Mbps`;
        }
        if (chartCtx) drawChart();
    }

    // Set UI Phase
    function setPhase(phase) {
        state.phase = phase;
        elements.cardPing.classList.remove('active');
        elements.cardDownload.classList.remove('active');
        elements.cardUpload.classList.remove('active');

        elements.statusBadge.className = 'status-badge ' + phase;

        switch (phase) {
            case 'ping':
                elements.statusText.textContent = 'Measuring Latency & Jitter...';
                elements.gaugePhaseName.textContent = 'PING';
                elements.cardPing.classList.add('active');
                break;
            case 'download':
                elements.statusText.textContent = 'Testing Download Speed...';
                elements.gaugePhaseName.textContent = 'DOWNLOAD';
                elements.cardDownload.classList.add('active');
                break;
            case 'upload':
                elements.statusText.textContent = 'Testing Upload Speed...';
                elements.gaugePhaseName.textContent = 'UPLOAD';
                elements.cardUpload.classList.add('active');
                break;
            case 'completed':
                elements.statusText.textContent = 'Speed Test Completed';
                elements.gaugePhaseName.textContent = 'COMPLETE';
                elements.testProgressBar.style.width = '100%';
                break;
            default:
                elements.statusText.textContent = 'Ready to Test';
                elements.gaugePhaseName.textContent = 'SPEED';
                elements.testProgressBar.style.width = '0%';
        }
    }

    // 1. Latency & Jitter Measurement
    async function measurePing(count = 10) {
        setPhase('ping');
        const latencies = [];

        for (let i = 0; i < count; i++) {
            if (!state.isRunning) break;
            const start = performance.now();
            try {
                const res = await fetch(`/api/v1/ping?t=${Date.now()}_${i}`, {
                    cache: 'no-store',
                    headers: { 'Cache-Control': 'no-cache', 'Pragma': 'no-cache' }
                });
                if (res.ok) {
                    const rtt = performance.now() - start;
                    latencies.push(rtt);
                    elements.valPing.textContent = rtt.toFixed(0);
                    elements.speedDisplay.textContent = rtt.toFixed(0);
                    elements.testProgressBar.style.width = `${((i + 1) / count) * 100}%`;
                }
            } catch (err) { }
            await new Promise(r => setTimeout(r, 60));
        }

        if (latencies.length === 0) return { ping: 0, jitter: 0 };

        latencies.sort((a, b) => a - b);
        const minPing = latencies[0];

        let jitterSum = 0;
        for (let i = 1; i < latencies.length; i++) {
            jitterSum += Math.abs(latencies[i] - latencies[i - 1]);
        }
        const jitter = latencies.length > 1 ? jitterSum / (latencies.length - 1) : 0;

        elements.valPing.textContent = minPing.toFixed(0);
        elements.valJitter.textContent = `${jitter.toFixed(1)} ms`;

        return { ping: minPing, jitter };
    }

    // 2. High-Performance Smooth Download Test
    // Pipelined 32MB chunks with Content-Length + xhr.onprogress for zero-GC native socket streaming
    // Real-time inertia-smoothed EMA filter for Speedtest.net style ramp-up without jagged drops
    function measureDownload(durationSec, parallelStreams, overhead) {
        return new Promise((resolve) => {
            setPhase('download');
            state.chartData = [];
            state.peakSpeed = 0;
            state.targetSpeed = 0;
            state.displayedSpeed = 0;
            elements.chartMaxIndicator.textContent = 'Peak: 0.0 Mbps';

            const startTime = performance.now();
            let isComplete = false;

            const streamBytes = new Array(parallelStreams).fill(0);
            const byteSamples = []; // Array of { time, bytes }
            let smoothSpeed = 0;

            startGaugeAnimation();

            // Periodic sampling timer (every 50ms)
            let chartTimer = 0;
            const sampleInterval = setInterval(() => {
                if (!state.isRunning || isComplete) return;

                const now = performance.now();
                const elapsedSec = (now - startTime) / 1000;
                const currentTotal = streamBytes.reduce((a, b) => a + b, 0);

                byteSamples.push({ time: now, bytes: currentTotal });
                // 1000ms sliding window for steady network flow
                while (byteSamples.length > 2 && now - byteSamples[0].time > 1000) {
                    byteSamples.shift();
                }

                const oldest = byteSamples[0];
                const dt = (now - oldest.time) / 1000;
                const db = currentTotal - oldest.bytes;

                if (dt >= 0.2) {
                    const instantMbps = ((db * 8) / (dt * 1_000_000)) * overhead;
                    // Speedtest.net Smooth Physics:
                    // Rapid responsive ramp-up upward, high inertia downward (no zero-valleys on jitter)
                    if (instantMbps > smoothSpeed) {
                        smoothSpeed = smoothSpeed === 0 ? instantMbps : (smoothSpeed + (instantMbps - smoothSpeed) * 0.25);
                    } else {
                        smoothSpeed = Math.max(smoothSpeed * 0.97, instantMbps);
                    }
                    state.targetSpeed = smoothSpeed;
                }

                // Push chart point every 150ms for buttery-smooth curve
                chartTimer += 50;
                if (chartTimer >= 150) {
                    chartTimer = 0;
                    if (state.targetSpeed > 0) {
                        pushDownloadPoint(state.targetSpeed);
                    }
                }

                // Update Progress & dynamically formatted counter (KB, MB, GB, TB, PB)
                const progress = Math.min((elapsedSec / durationSec) * 100, 100);
                elements.testProgressBar.style.width = `${progress}%`;
                elements.valDownloadBytes.textContent = formatBytes(currentTotal);

                if (elapsedSec >= durationSec) {
                    finishDownload();
                }
            }, 50);

            function finishDownload() {
                if (isComplete) return;
                isComplete = true;
                clearInterval(sampleInterval);

                state.activeXHRs.forEach(xhr => {
                    try { xhr.abort(); } catch (_) { }
                });
                state.activeXHRs = [];

                const totalElapsedSec = (performance.now() - startTime) / 1000;
                const finalBytes = streamBytes.reduce((a, b) => a + b, 0);
                const finalMbps = totalElapsedSec > 0 ? ((finalBytes * 8) / (totalElapsedSec * 1_000_000)) * overhead : 0;

                stopGaugeAnimation(finalMbps);
                elements.valDownload.textContent = finalMbps.toFixed(1);
                elements.valDownloadBytes.textContent = formatBytes(finalBytes);
                state.results.download = finalMbps;
                state.results.downloadBytes = finalBytes;

                resolve(finalMbps);
            }

            // Launch parallel streaming readers with Content-Length for full wire speed
            for (let i = 0; i < parallelStreams; i++) {
                startStreamDownload(i);
            }

            function startStreamDownload(streamIndex) {
                if (isComplete || !state.isRunning) return;

                const xhr = new XMLHttpRequest();
                state.activeXHRs.push(xhr);

                // Use 32MB chunks from server RAM with Content-Length for maximum throughput
                xhr.open('GET', `/api/v1/download?size=33554432&stream=${streamIndex}&t=${Date.now()}_${Math.random()}`, true);
                xhr.responseType = 'blob';

                let lastLoaded = 0;
                let nextDispatched = false;

                xhr.onprogress = (e) => {
                    if (isComplete || !state.isRunning) return;
                    const loaded = e.loaded || 0;
                    const delta = loaded - lastLoaded;
                    if (delta > 0) {
                        streamBytes[streamIndex] += delta;
                        lastLoaded = loaded;
                    }

                    // Pipelined overlap: launch next chunk at 85% to maintain 100% link saturation
                    if (loaded >= 33554432 * 0.85 && !nextDispatched) {
                        nextDispatched = true;
                        startStreamDownload(streamIndex);
                    }
                };

                xhr.onload = xhr.onerror = () => {
                    const idx = state.activeXHRs.indexOf(xhr);
                    if (idx !== -1) state.activeXHRs.splice(idx, 1);

                    if (!nextDispatched && !isComplete && state.isRunning) {
                        startStreamDownload(streamIndex);
                    }
                };

                xhr.send();
            }
        });
    }

    // 3. High-Performance Smooth Upload Test
    // Pipelined continuous 8MB upload chunks with xhr.upload.onprogress
    // Overlapping chunk dispatch eliminates round-trip latency gaps on remote networks
    function measureUpload(durationSec, parallelStreams, overhead) {
        return new Promise((resolve) => {
            setPhase('upload');
            state.chartData = [];
            state.peakSpeed = 0;
            state.targetSpeed = 0;
            state.displayedSpeed = 0;
            elements.chartMaxIndicator.textContent = 'Peak: 0.0 Mbps';

            const startTime = performance.now();
            let isComplete = false;

            const streamBytes = new Array(parallelStreams).fill(0);
            const byteSamples = []; // Array of { time, bytes }
            let smoothSpeed = 0;

            // Pre-allocate 8MB chunk buffer in RAM (large buffer drastically reduces HTTP handshake overhead)
            const chunkSize = 8 * 1024 * 1024;
            const payload = new Uint8Array(chunkSize);
            for (let i = 0; i < chunkSize; i++) payload[i] = (i * 31) & 0xff;
            const blob = new Blob([payload], { type: 'application/octet-stream' });

            startGaugeAnimation();

            let chartTimer = 0;
            const sampleInterval = setInterval(() => {
                if (!state.isRunning || isComplete) return;

                const now = performance.now();
                const elapsedSec = (now - startTime) / 1000;
                const currentTotal = streamBytes.reduce((a, b) => a + b, 0);

                byteSamples.push({ time: now, bytes: currentTotal });
                while (byteSamples.length > 2 && now - byteSamples[0].time > 1000) {
                    byteSamples.shift();
                }

                const oldest = byteSamples[0];
                const dt = (now - oldest.time) / 1000;
                const db = currentTotal - oldest.bytes;

                if (dt >= 0.2) {
                    const instantMbps = ((db * 8) / (dt * 1_000_000)) * overhead;
                    if (instantMbps > smoothSpeed) {
                        smoothSpeed = smoothSpeed === 0 ? instantMbps : (smoothSpeed + (instantMbps - smoothSpeed) * 0.25);
                    } else {
                        smoothSpeed = Math.max(smoothSpeed * 0.97, instantMbps);
                    }
                    state.targetSpeed = smoothSpeed;
                }

                chartTimer += 50;
                if (chartTimer >= 150) {
                    chartTimer = 0;
                    if (state.targetSpeed > 0) {
                        pushUploadPoint(state.targetSpeed);
                    }
                }

                const progress = Math.min((elapsedSec / durationSec) * 100, 100);
                elements.testProgressBar.style.width = `${progress}%`;
                elements.valUploadBytes.textContent = formatBytes(currentTotal);

                if (elapsedSec >= durationSec) {
                    finishUpload();
                }
            }, 50);

            function finishUpload() {
                if (isComplete) return;
                isComplete = true;
                clearInterval(sampleInterval);

                state.activeXHRs.forEach(xhr => {
                    try { xhr.abort(); } catch (_) { }
                });
                state.activeXHRs = [];

                const totalElapsedSec = (performance.now() - startTime) / 1000;
                const finalBytes = streamBytes.reduce((a, b) => a + b, 0);
                const finalMbps = totalElapsedSec > 0 ? ((finalBytes * 8) / (totalElapsedSec * 1_000_000)) * overhead : 0;

                stopGaugeAnimation(finalMbps);
                elements.valUpload.textContent = finalMbps.toFixed(1);
                elements.valUploadBytes.textContent = formatBytes(finalBytes);
                state.results.upload = finalMbps;
                state.results.uploadBytes = finalBytes;

                resolve(finalMbps);
            }

            for (let i = 0; i < parallelStreams; i++) {
                sendChunk(i);
            }

            function sendChunk(streamIndex) {
                if (isComplete || !state.isRunning) return;

                const xhr = new XMLHttpRequest();
                state.activeXHRs.push(xhr);
                xhr.open('POST', `/api/v1/upload?stream=${streamIndex}&t=${Date.now()}_${Math.random()}`, true);

                let lastLoaded = 0;
                let nextDispatched = false;

                // Continuous byte progression tracking
                xhr.upload.onprogress = (e) => {
                    if (isComplete || !state.isRunning) return;
                    const loaded = e.loaded || 0;
                    const delta = loaded - lastLoaded;
                    if (delta > 0) {
                        streamBytes[streamIndex] += delta;
                        lastLoaded = loaded;
                    }

                    // Pipelined Overlap: when chunk is 85% uploaded to socket buffer,
                    // launch the next chunk immediately so there is ZERO idle latency gap!
                    if (loaded >= chunkSize * 0.85 && !nextDispatched) {
                        nextDispatched = true;
                        sendChunk(streamIndex);
                    }
                };

                xhr.onload = xhr.onerror = () => {
                    const idx = state.activeXHRs.indexOf(xhr);
                    if (idx !== -1) state.activeXHRs.splice(idx, 1);

                    if (!nextDispatched && !isComplete && state.isRunning) {
                        sendChunk(streamIndex);
                    }
                };

                xhr.send(blob);
            }
        });
    }

    // Main Test Coordinator
    async function startSpeedTest(isStress = false) {
        if (state.isRunning) {
            abortSpeedTest();
            return;
        }

        state.isRunning = true;
        elements.btnStart.classList.add('btn-running');
        elements.btnStart.innerHTML = `
            <svg class="btn-svg" viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="6" width="12" height="12"></rect></svg>
            <span>STOP TEST</span>
        `;
        elements.btnStress.disabled = true;

        elements.valPing.textContent = '--';
        elements.valJitter.textContent = '-- ms';
        elements.valDownload.textContent = '--';
        elements.valDownloadBytes.textContent = '--';
        elements.valUpload.textContent = '--';
        elements.valUploadBytes.textContent = '--';
        elements.speedDisplay.textContent = '0.0';
        drawGauge(0);

        state.dlChartData = [];
        state.ulChartData = [];
        state.dlPeakSpeed = 0;
        state.ulPeakSpeed = 0;
        if (elements.chartDlPeak) elements.chartDlPeak.textContent = 'Peak: 0.0 Mbps';
        if (elements.chartUlPeak) elements.chartUlPeak.textContent = 'Peak: 0.0 Mbps';
        drawDownloadChart();
        drawUploadChart();

        const duration = isStress ? 20 : state.config.duration;
        const parallel = isStress ? 16 : state.config.parallel;
        const overhead = state.config.overhead;

        try {
            // Step 1: Latency & Jitter
            const pingRes = await measurePing(isStress ? 15 : 10);
            state.results.ping = pingRes.ping;
            state.results.jitter = pingRes.jitter;

            if (!state.isRunning) return;
            await new Promise(r => setTimeout(r, 300));

            // Step 2: Smooth Download Test
            await measureDownload(duration, parallel, overhead);

            if (!state.isRunning) return;
            await new Promise(r => setTimeout(r, 400));

            // Step 3: Smooth Upload Test
            await measureUpload(duration, parallel, overhead);

            // Step 4: Completion
            setPhase('completed');
            stopGaugeAnimation(state.results.download);

            saveResult({
                type: isStress ? 'stress' : 'standard',
                ping: state.results.ping,
                jitter: state.results.jitter,
                download: state.results.download,
                upload: state.results.upload,
                timestamp: new Date().toISOString()
            });

        } catch (err) {
            console.error('Speed test error:', err);
        } finally {
            cleanup();
        }
    }

    function abortSpeedTest() {
        state.isRunning = false;
        if (state.downloadAbortController) {
            try { state.downloadAbortController.abort(); } catch (_) { }
        }
        state.activeXHRs.forEach(xhr => {
            try { xhr.abort(); } catch (_) { }
        });
        state.activeXHRs = [];
        setPhase('idle');
        stopGaugeAnimation(0);
        cleanup();
    }

    function cleanup() {
        state.isRunning = false;
        elements.btnStart.classList.remove('btn-running');
        elements.btnStart.innerHTML = `
            <svg class="btn-svg" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>
            <span>START TEST</span>
        `;
        elements.btnStress.disabled = false;
    }

    // Results Persistence & History
    async function saveResult(item) {
        try {
            await fetch('/api/v1/results', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    type: item.type,
                    server: 'Custome Speedtest Engine',
                    mbps: item.download,
                    ping_ms: item.ping,
                    jitter_ms: item.jitter,
                    bytes: state.results.downloadBytes + state.results.uploadBytes,
                    parallel: state.config.parallel
                })
            });
        } catch (_) { }

        try {
            const history = JSON.parse(localStorage.getItem('ct_speedtest_history') || '[]');
            history.unshift(item);
            if (history.length > 30) history.pop();
            localStorage.setItem('ct_speedtest_history', JSON.stringify(history));
            renderHistory();
        } catch (_) { }
    }

    function renderHistory() {
        const history = JSON.parse(localStorage.getItem('ct_speedtest_history') || '[]');
        if (history.length === 0) {
            elements.historyTbody.innerHTML = '<tr><td colspan="5" class="empty-state">No tests recorded yet.</td></tr>';
            return;
        }

        elements.historyTbody.innerHTML = history.map(item => {
            const date = new Date(item.timestamp).toLocaleString();
            return `
                <tr>
                    <td>${date}</td>
                    <td>${item.ping.toFixed(0)} ms</td>
                    <td>${item.jitter.toFixed(1)} ms</td>
                    <td style="color: var(--primary); font-weight: 600;">${item.download.toFixed(1)} Mbps</td>
                    <td style="color: var(--accent); font-weight: 600;">${item.upload.toFixed(1)} Mbps</td>
                </tr>
            `;
        }).join('');
    }

    // Theme Management
    function initTheme() {
        const savedTheme = localStorage.getItem('ct_speedtest_theme') || 'dark';
        applyTheme(savedTheme);
    }

    function applyTheme(theme) {
        const isLight = theme === 'light';
        if (isLight) {
            document.body.classList.add('light-mode');
            if (elements.themeIconSun) elements.themeIconSun.classList.remove('hidden');
            if (elements.themeIconMoon) elements.themeIconMoon.classList.add('hidden');
        } else {
            document.body.classList.remove('light-mode');
            if (elements.themeIconSun) elements.themeIconSun.classList.add('hidden');
            if (elements.themeIconMoon) elements.themeIconMoon.classList.remove('hidden');
        }
        localStorage.setItem('ct_speedtest_theme', theme);
        drawGauge(state.displayedSpeed);
        drawChart();
    }

    function toggleTheme() {
        const isLight = document.body.classList.contains('light-mode');
        applyTheme(isLight ? 'dark' : 'light');
    }

    // Settings Management
    function loadSettings() {
        const saved = localStorage.getItem('ct_speedtest_config');
        if (saved) {
            try {
                Object.assign(state.config, JSON.parse(saved));
            } catch (_) { }
        }

        elements.inputDuration.value = state.config.duration;
        elements.lblDurationVal.textContent = `${state.config.duration}s`;
        elements.inputParallel.value = state.config.parallel;
        elements.lblParallelVal.textContent = state.config.parallel;
        elements.inputOverhead.value = state.config.overhead;
        elements.lblOverheadVal.textContent = `${parseFloat(state.config.overhead).toFixed(2)}x`;
        elements.chkAutoStart.checked = state.config.autoStart;

        elements.infoDuration.textContent = `${state.config.duration}s / phase`;
        elements.infoStreams.textContent = `${state.config.parallel} Parallel`;
    }

    function saveSettings() {
        state.config.duration = parseInt(elements.inputDuration.value, 10);
        state.config.parallel = parseInt(elements.inputParallel.value, 10);
        state.config.overhead = parseFloat(elements.inputOverhead.value);
        state.config.autoStart = elements.chkAutoStart.checked;

        localStorage.setItem('ct_speedtest_config', JSON.stringify(state.config));
        loadSettings();
        elements.settingsModal.classList.add('hidden');
    }

    // Event Listeners
    function setupEvents() {
        window.addEventListener('resize', resizeCanvases);

        if (elements.btnThemeToggle) {
            elements.btnThemeToggle.addEventListener('click', toggleTheme);
        }

        elements.btnStart.addEventListener('click', () => startSpeedTest(false));
        elements.btnStress.addEventListener('click', () => startSpeedTest(true));

        elements.inputDuration.addEventListener('input', (e) => {
            elements.lblDurationVal.textContent = `${e.target.value}s`;
        });
        elements.inputParallel.addEventListener('input', (e) => {
            elements.lblParallelVal.textContent = e.target.value;
        });
        elements.inputOverhead.addEventListener('input', (e) => {
            elements.lblOverheadVal.textContent = `${parseFloat(e.target.value).toFixed(2)}x`;
        });

        elements.btnSettingsToggle.addEventListener('click', () => {
            elements.settingsModal.classList.remove('hidden');
        });
        elements.btnCloseSettings.addEventListener('click', () => {
            elements.settingsModal.classList.add('hidden');
        });
        elements.btnSaveSettings.addEventListener('click', saveSettings);

        elements.btnHistoryToggle.addEventListener('click', () => {
            renderHistory();
            elements.historyModal.classList.remove('hidden');
        });
        elements.btnCloseHistory.addEventListener('click', () => {
            elements.historyModal.classList.add('hidden');
        });
        elements.btnClearHistory.addEventListener('click', () => {
            localStorage.removeItem('ct_speedtest_history');
            renderHistory();
        });

        [elements.settingsModal, elements.historyModal].forEach(modal => {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) modal.classList.add('hidden');
            });
        });
    }

    function init() {
        initTheme();
        setupEvents();
        loadSettings();
        renderHistory();
        resizeCanvases();

        fetch('/api/v1/config')
            .then(res => res.json())
            .then(cfg => {
                if (cfg.buffer_size) {
                    const mb = (cfg.buffer_size / (1024 * 1024)).toFixed(0);
                    document.getElementById('info-buffer').textContent = `${mb} MB in RAM`;
                }
            })
            .catch(() => { });

        if (state.config.autoStart) {
            setTimeout(() => startSpeedTest(false), 500);
        }
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
