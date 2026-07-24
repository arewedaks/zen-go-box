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
});
