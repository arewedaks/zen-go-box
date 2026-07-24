document.addEventListener('DOMContentLoaded', () => {
    const statusBadge = document.getElementById('status-badge');
    const statusText = document.getElementById('status-text');
    const powerSwitch = document.getElementById('power-switch');
    const coreName = document.getElementById('core-name');
    const corePid = document.getElementById('core-pid');
    const btnClash = document.getElementById('btn-clash');
    const terminal = document.getElementById('terminal');
    const btnScroll = document.getElementById('btn-scroll');

    let autoScroll = true;
    let isProcessing = false;
    let currentConfig = null;

    // Settings Modal Elements
    const btnSettings = document.getElementById('btn-settings');
    const settingsModal = document.getElementById('settings-modal');
    const btnCloseSettings = document.getElementById('btn-close-settings');
    const btnSaveSettings = document.getElementById('btn-save-settings');

    // Toggle Auto Scroll
    btnScroll.addEventListener('click', () => {
        autoScroll = !autoScroll;
        btnScroll.classList.toggle('active', autoScroll);
        if (autoScroll) scrollToBottom();
    });

    terminal.addEventListener('scroll', () => {
        const isAtBottom = terminal.scrollHeight - terminal.scrollTop <= terminal.clientHeight + 10;
        if (!isAtBottom && autoScroll) {
            autoScroll = false;
            btnScroll.classList.remove('active');
        }
    });

    function scrollToBottom() {
        terminal.scrollTop = terminal.scrollHeight;
    }

    function formatLogLine(line) {
        if (!line) return '';
        let className = 'log-line';
        if (line.includes('[INFO]')) className += ' log-info';
        else if (line.includes('[WARN]')) className += ' log-warn';
        else if (line.includes('[ERROR]') || line.includes('[FATAL]')) className += ' log-error';

        // Extract time if exists (assuming HH:MM:SS format at start)
        const timeMatch = line.match(/^(\d{2}:\d{2}:\d{2})/);
        if (timeMatch) {
            line = line.replace(timeMatch[0], `<span class="log-time">${timeMatch[0]}</span>`);
        }
        
        return `<div class="${className}">${line}</div>`;
    }

    async function fetchLogs() {
        try {
            const res = await fetch('/api/logs');
            const data = await res.json();
            
            if (data.logs && data.logs.length > 0) {
                terminal.innerHTML = data.logs.map(formatLogLine).join('');
                if (autoScroll) scrollToBottom();
            }
        } catch (e) {
            console.error('Failed to fetch logs', e);
        }
    }

    async function fetchStatus() {
        try {
            const res = await fetch('/api/status');
            const data = await res.json();
            
            if (data.needs_setup) {
                document.getElementById('setup-wizard').style.display = 'block';
                document.getElementById('main-container').style.display = 'none';
                return;
            } else {
                document.getElementById('setup-wizard').style.display = 'none';
                document.getElementById('main-container').style.display = 'block';
            }
            
            // Update Status Badge
            statusBadge.className = `status-badge ${data.running ? 'active' : 'offline'}`;
            statusText.textContent = data.running ? 'Active' : 'Offline';
            
            // Update Core Info
            coreName.textContent = data.core ? data.core.toUpperCase() : 'Unknown Core';
            corePid.textContent = data.running && data.pid > 0 ? `PID: ${data.pid}` : 'PID: ---';
            
            // Clash Dashboard Button
            btnClash.style.display = (data.core === 'clash' && data.running) ? 'inline-block' : 'none';

            // Update Switch (Only if not processing a click)
            if (!isProcessing) {
                powerSwitch.checked = data.running;
                powerSwitch.disabled = false;
            }
        } catch (e) {
            statusBadge.className = 'status-badge offline';
            statusText.textContent = 'Disconnected';
            powerSwitch.disabled = true;
        }
    }

    // Handle Switch Click
    powerSwitch.addEventListener('change', async (e) => {
        isProcessing = true;
        powerSwitch.disabled = true;
        
        const action = e.target.checked ? 'start' : 'stop';
        try {
            await fetch(`/api/${action}`, { method: 'POST' });
            // Immediate fetch to reflect status
            await fetchStatus();
            await fetchLogs();
        } catch (err) {
            alert('Failed to execute command: ' + err.message);
            e.target.checked = !e.target.checked; // Revert
        } finally {
            isProcessing = false;
            powerSwitch.disabled = false;
        }
    });

    // Polling intervals
    fetchStatus();
    fetchLogs();
    setInterval(fetchStatus, 3000);
    setInterval(fetchLogs, 2000);

    // Settings Modal Logic
    async function openSettings() {
        try {
            const res = await fetch('/api/config');
            currentConfig = await res.json();
            
            // Populate form
            document.getElementById('setting-core').value = currentConfig.Core.BinName || 'clash';
            document.getElementById('setting-wifi').checked = currentConfig.Wifi.Enabled;
            document.getElementById('setting-cron').checked = currentConfig.Schedule.Enabled;
            document.getElementById('setting-rules').checked = currentConfig.Subscription.InjectRules;
            
            settingsModal.style.display = 'flex';
        } catch (e) {
            alert('Failed to load settings: ' + e.message);
        }
    }

    async function saveSettings() {
        if (!currentConfig) return;
        
        btnSaveSettings.disabled = true;
        btnSaveSettings.textContent = 'Saving...';
        
        // Update config object
        currentConfig.Core.BinName = document.getElementById('setting-core').value;
        currentConfig.Wifi.Enabled = document.getElementById('setting-wifi').checked;
        currentConfig.Schedule.Enabled = document.getElementById('setting-cron').checked;
        currentConfig.Subscription.InjectRules = document.getElementById('setting-rules').checked;
        
        try {
            const res = await fetch('/api/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(currentConfig)
            });
            
            if (res.ok) {
                alert('Settings saved! Restarting proxy service to apply changes...');
                await fetch('/api/stop', { method: 'POST' });
                await new Promise(r => setTimeout(r, 1000));
                await fetch('/api/start', { method: 'POST' });
                
                settingsModal.style.display = 'none';
                fetchStatus();
            } else {
                const errText = await res.text();
                alert('Failed to save settings: ' + errText);
            }
        } catch (e) {
            alert('Error saving settings: ' + e.message);
        } finally {
            btnSaveSettings.disabled = false;
            btnSaveSettings.textContent = 'Save & Restart Daemon';
        }
    }

    btnSettings.addEventListener('click', openSettings);
    btnCloseSettings.addEventListener('click', () => { settingsModal.style.display = 'none'; });
    btnSaveSettings.addEventListener('click', saveSettings);
    
    // Close modal on outside click
    settingsModal.addEventListener('click', (e) => {
        if (e.target === settingsModal) {
            settingsModal.style.display = 'none';
        }
    });

    // Setup Wizard Logic
    const btnStartSetup = document.getElementById('btn-start-setup');
    if (btnStartSetup) {
        btnStartSetup.addEventListener('click', async () => {
            const core = document.getElementById('setup-core').value;
            const btn = document.getElementById('btn-start-setup');
            const logsPanel = document.getElementById('setup-logs-panel');
            const setupTerminal = document.getElementById('setup-terminal');
            const setupStatusText = document.getElementById('setup-status-text');

            btn.disabled = true;
            btn.innerHTML = 'Installing...';
            logsPanel.style.display = 'block';
            setupTerminal.innerText = "> Preparing environment...\n";

            try {
                await fetch('/api/setup', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ core: core })
                });

                const pollInterval = setInterval(async () => {
                    try {
                        const res = await fetch('/api/setup_log');
                        const text = await res.text();
                        if (text && text.trim() !== "") {
                            setupTerminal.innerText = text;
                            setupTerminal.scrollTop = setupTerminal.scrollHeight;
                        }
                        
                        if (text.includes("Setup complete!")) {
                            clearInterval(pollInterval);
                            setupStatusText.innerText = "Setup complete! Restarting daemon...";
                            
                            // Restart daemon so it loads the new config and drops Setup Mode
                            await fetch('/api/restart', { method: 'POST' });
                            
                            setTimeout(() => {
                                window.location.reload();
                            }, 3000);
                        }
                    } catch (e) {
                        console.error(e);
                    }
                }, 1000);

            } catch (e) {
                alert('Setup failed: ' + e.message);
                btn.disabled = false;
                btn.innerHTML = '🚀 Start Installation';
            }
        });
    }
});
