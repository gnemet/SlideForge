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
    if (!output) return;

    const evtSource = new EventSource("/events/logs");
    evtSource.onmessage = function (event) {
        output.textContent = event.data;
        // Visual feedback
        output.style.color = 'var(--accent-color)';
        setTimeout(() => {
            output.style.color = 'var(--text-color)';
        }, 500);
    };
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
