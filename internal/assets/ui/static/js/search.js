let searchModes = [
    { value: 'fts', label: 'Full Text', title: 'Search using full text', icon: 'fa-magic', hasThreshold: false },
    { value: 'similarity', label: 'Similarity', title: 'Find slides with similar content', icon: 'fa-equals', hasThreshold: true },
    { value: 'word_similarity', label: 'Word Similarity', title: 'Find slides with similar words', icon: 'fa-font', hasThreshold: true }
];

let currentModeIdx = 0;

function initSearchComponent() {
    if (window.SEARCH_MODES_CONFIG) {
        searchModes = window.SEARCH_MODES_CONFIG;
    }
    // Restore state if available
    const modeInput = document.getElementById('search-mode');
    if (modeInput) {
        const currentModeValue = modeInput.value;
        const foundIdx = searchModes.findIndex(m => m.value === currentModeValue);
        if (foundIdx >= 0) {
            currentModeIdx = foundIdx;
        }
    }
    updateSearchUI();
}

function cycleSearchMode() {
    currentModeIdx = (currentModeIdx + 1) % searchModes.length;
    updateSearchUI();

    // Trigger htmx search if input has value
    const input = document.getElementById('search-input');
    if (input && input.value) {
        htmx.trigger(input, 'keyup');
    }
}

function updateSearchUI() {
    const mode = searchModes[currentModeIdx];

    // Update Hidden Input
    const modeInput = document.getElementById('search-mode');
    if (modeInput) modeInput.value = mode.value;

    // Update Label and Icon
    const labelSpan = document.getElementById('mode-label');
    const btnIcon = document.querySelector('#mode-loop-btn i');

    if (labelSpan) labelSpan.innerText = mode.label;
    if (btnIcon) btnIcon.className = 'fas ' + mode.icon;

    // Update Button Title
    const btn = document.getElementById('mode-loop-btn');
    if (btn) btn.title = mode.title || '';

    // Toggle Threshold Visibility
    const thresholdControl = document.getElementById('threshold-control');
    const separator = document.getElementById('mode-separator');

    if (mode.hasThreshold) {
        if (thresholdControl) thresholdControl.classList.remove('hidden');
        if (separator) separator.classList.remove('hidden');
    } else {
        if (thresholdControl) thresholdControl.classList.add('hidden');
        if (separator) separator.classList.add('hidden');
    }
}

// Global scope needed for inline event handlers, though best practice is to attach listeners
window.cycleSearchMode = cycleSearchMode;
window.saveSetting = function (key, value) {
    const formData = new FormData();
    formData.append('key', key);
    formData.append('value', value);
    fetch('/search-settings', {
        method: 'POST',
        body: formData
    }).then(r => r.text()).then(t => console.log('Setting saved:', t));
};

document.addEventListener('DOMContentLoaded', initSearchComponent);
