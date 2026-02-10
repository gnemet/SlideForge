// Antigravity Global Utilities
window.Antigravity = {
    Config: {
        Theme: localStorage.getItem('theme') || 'light',
        Language: document.documentElement.lang || 'en'
    },

    copyToClipboard: function (text) {
        if (!text) return;
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(text).then(() => {
                this.showNotification('Copied to clipboard');
            });
        }
    },

    showNotification: function (msg, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `toast-notification toast-${type}`;
        toast.innerHTML = `<i class='fas fa-info-circle'></i> ${msg}`;
        document.body.appendChild(toast);

        setTimeout(() => {
            toast.classList.add('fade-out');
            setTimeout(() => toast.remove(), 500);
        }, 3000);
    }
};

$(document).ready(function () {
    initTheme();
    initGlobalEvents();
    initUserMenu();
    initTerminalObserver();
    initSidebar();
});

function initTheme() {
    const themeToggle = document.getElementById('theme-toggle');
    if (!themeToggle) return;

    const themeIcon = themeToggle.querySelector('i');
    const currentTheme = localStorage.getItem('theme') || 'light';

    document.documentElement.setAttribute('data-theme', currentTheme);
    updateThemeIcon(themeIcon, currentTheme);

    themeToggle.addEventListener('click', () => {
        const current = document.documentElement.getAttribute('data-theme') || 'light';
        const newTheme = current === 'light' ? 'dark' : 'light';

        document.documentElement.setAttribute('data-theme', newTheme);
        localStorage.setItem('theme', newTheme);
        updateThemeIcon(themeIcon, newTheme);
    });
}

function updateThemeIcon(icon, theme) {
    if (icon) {
        icon.className = theme === 'light' ? 'fas fa-moon' : 'fas fa-sun';
    }
}

function toggleLanguage() {
    const btn = document.querySelector('[data-action="switch-lang"]');
    if (!btn) return;

    const langsData = btn.getAttribute('data-langs');
    if (!langsData) return;

    const langs = JSON.parse(langsData);
    const current = btn.getAttribute('data-current') || 'en';

    let idx = langs.indexOf(current);
    let nextIdx = (idx + 1) % langs.length;
    const next = langs[nextIdx];

    window.location.href = `/set-language?lang=${next}`;
}

function initGlobalEvents() {
    document.addEventListener('click', (e) => {
        const btn = e.target.closest('[data-action]');
        if (!btn) return;

        const action = btn.dataset.action;
        if (action === 'switch-lang') {
            e.preventDefault();
            toggleLanguage();
        }
    });
}

function initUserMenu() {
    const toggle = document.getElementById('user-menu-toggle');
    const dropdown = document.getElementById('user-dropdown');
    if (!toggle || !dropdown) return;

    toggle.addEventListener('click', (e) => {
        e.stopPropagation();
        dropdown.classList.toggle('hidden');
    });

    document.addEventListener('click', (e) => {
        if (!dropdown.contains(e.target) && !toggle.contains(e.target)) {
            dropdown.classList.add('hidden');
        }
    });
}

function initTerminalObserver() {
    const output = document.getElementById('bg-process-output');
    const statusContainer = document.getElementById('reprocess-status');
    if (!statusContainer && !output) return;

    let wasProcessing = false;

    const evtSource = new EventSource("/events/status");
    evtSource.onmessage = function (event) {
        const data = JSON.parse(event.data);

        if (output && data.last_log) {
            output.textContent = data.last_log;
            output.style.color = 'var(--accent-color)';
            setTimeout(() => {
                output.style.color = 'var(--text-color)';
            }, 500);
        }

        // Trigger refresh if processing just finished
        if (wasProcessing && !data.is_processing) {
            console.log("Processing finished, refreshing files...");
            document.body.dispatchEvent(new Event('refreshFiles'));
            Antigravity.showNotification('Processing completed, list updated', 'success');
        }
        wasProcessing = data.is_processing;

        // Update Reprocess Status badge globally if on dashboard
        if (statusContainer) {
            if (data.is_processing) {
                let statusText = "Processing...";
                if (data.current_file) {
                    statusText = `Processing: ${data.current_file} (${data.total_queued} left)`;
                }

                let durationHtml = "";
                if (data.start_time) {
                    const elapsed = Math.floor(Date.now() / 1000 - data.start_time);
                    const minutes = Math.floor(elapsed / 60);
                    const seconds = elapsed % 60;
                    durationHtml = ` <span style='opacity: 0.8; font-family: monospace;'>[${minutes}:${seconds.toString().padStart(2, '0')}]</span>`;
                }

                statusContainer.innerHTML = `
                    <span class="badge bg-warning animate-pulse" style="margin: 0;">${statusText}${durationHtml}</span>
                    <i class="fas fa-cog fa-spin text-warning"></i>
                `;
                const count = parseInt(document.getElementById('total-pptx-count')?.innerText || '0');
                const btnClass = count > 10 ? 'btn-danger' : 'btn-muted';

                // If it was processing and now stopped, return to ready state
                statusContainer.innerHTML = `
                    <div class="stat-value" style="font-size: 1.1rem; color: var(--success-color);">
                        <i class="fas fa-check-circle"></i> Ready
                    </div>
                    <button onclick="confirmReprocess(${count})" class="btn ${btnClass} btn-sm"
                        title="Reprocess all PPTX">
                        <i class="fas fa-sync"></i>
                    </button>
                `;
            }
        }
    };
}

function confirmReprocess(count) {
    let msg = "Are you sure you want to reprocess all files?";
    if (count > 10) {
        msg = `⚠️ DANGER: You have ${count} files. \n\nReprocessing will:\n1. Clear all current slide metadata and AI analysis.\n2. Move files from the Template folder back to Stage.\n3. Re-run complete AI analysis for EVERY file (may incur costs and take time).\n\nDo you want to proceed?`;
    } else {
        msg = "Reprocess all files? This will reset metadata and re-run AI analysis.";
    }

    if (confirm(msg)) {
        htmx.ajax('POST', '/reprocess', { target: '#reprocess-status' });
    }
}

function initSidebar() {
    // Restore state
    const isCollapsed = localStorage.getItem('sidebar-collapsed') === 'true';
    const sidebar = document.querySelector('.sidebar');
    if (sidebar && isCollapsed) {
        sidebar.classList.add('collapsed');
    }

    // Toggle logic
    $(document).on('click', '#sidebar-toggle', function (e) {
        e.preventDefault();
        const sidebar = document.querySelector('.sidebar');
        if (!sidebar) return;

        sidebar.classList.toggle('collapsed');
        const collapsed = sidebar.classList.contains('collapsed');
        localStorage.setItem('sidebar-collapsed', collapsed);
    });
}
