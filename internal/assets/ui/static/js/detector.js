// Global Handlers exposed for HTML template 'onclick' attributes
window.handleSlideClick = (el, event) => {
    if (!el) return;

    // Retrieve DOM elements inside handler to ensure they are available
    const mappingForm = document.getElementById('mapping-form-container');
    const emptyNotice = document.getElementById('no-slide-selected');
    const selectionInfo = document.getElementById('selection-info');
    const formFileId = document.getElementById('form-file-id');
    const formSlideNum = document.getElementById('form-slide-num');
    const largePreview = document.getElementById('large-preview-img');
    const slideTextarea = document.getElementById('slide-text-content');

    const thumbnail = el.dataset.path;
    const contentText = el.dataset.content;
    const slideNum = el.dataset.slideNum;

    // Find slide info from tree structure
    const node = el.closest('.tree-node');
    if (!node) return;

    const pptNode = node.querySelector('.ppt-node');
    if (!pptNode) return;

    const fileID = pptNode.dataset.id;
    const filename = pptNode.querySelector('span').innerText;

    // Update Form & UI
    const listContainer = document.getElementById('discovered-mappings');
    const slideLabel = (listContainer && listContainer.dataset.labelSlide) ? listContainer.dataset.labelSlide : 'Slide';

    if (formFileId) formFileId.value = fileID;
    if (formSlideNum) formSlideNum.value = slideNum;
    if (selectionInfo) selectionInfo.innerText = `${filename} - ${slideLabel} ${slideNum}`;
    if (largePreview && thumbnail) largePreview.src = thumbnail;
    if (slideTextarea) slideTextarea.value = contentText || '';

    // Show Form, Hide Notice
    if (mappingForm) mappingForm.style.display = 'block';
    if (emptyNotice) emptyNotice.style.display = 'none';

    // Update Discovered List for this slide
    if (window.updateDiscoveredList) window.updateDiscoveredList(fileID, slideNum);

    // Highlight selected
    document.querySelectorAll('.slide-item').forEach(i => i.classList.remove('selected'));
    el.classList.add('selected');
};

window.toggleFolder = (el, event) => {
    el.classList.toggle('expanded');
    const content = el.nextElementSibling;
    if (content) {
        content.style.display = content.style.display === 'none' ? 'block' : 'none';
    }
};

window.toggleNode = (el, event) => {
    if (el) el.classList.toggle('expanded');
};

document.addEventListener('DOMContentLoaded', () => {
    const itemsContainer = document.getElementById('discovery-items');
    const searchInput = document.getElementById('ppt-search');

    // Load data from JSON script tag
    const dataTag = document.getElementById('discovered-data');
    try {
        window.allDiscovered = dataTag ? JSON.parse(dataTag.textContent) : [];
    } catch (e) {
        console.error("Failed to parse discovered data:", e);
        window.allDiscovered = [];
    }

    // Initial marking of the tree
    markExistingDiscoveries();

    function markExistingDiscoveries() {
        if (!window.allDiscovered) return;

        // Group discovered by file/slide for efficient lookup
        const map = new Set();
        window.allDiscovered.forEach(d => {
            map.add(`${d.pptx_file_id}-${d.slide_number}`);
        });

        document.querySelectorAll('.slide-item').forEach(el => {
            const slideNum = el.dataset.slideNum;
            const node = el.closest('.tree-node');
            if (!node) return;

            const pptNode = node.querySelector('.ppt-node');
            if (!pptNode) return;

            const fileID = pptNode.dataset.id;

            if (slideNum && map.has(`${fileID}-${slideNum}`)) {
                el.classList.add('has-discovery');
            } else {
                el.classList.remove('has-discovery');
            }
        });
    }
    window.markExistingDiscoveries = markExistingDiscoveries;

    function updateDiscoveredList(fileID, slideNum) {
        if (!window.allDiscovered || !itemsContainer) return;

        const filtered = window.allDiscovered.filter(d =>
            d.pptx_file_id == fileID && d.slide_number == slideNum
        );

        const listContainer = document.getElementById('discovered-mappings');
        const noMappingsLabel = (listContainer && listContainer.dataset.labelNoMappings) ? listContainer.dataset.labelNoMappings : 'No mappings yet.';

        itemsContainer.innerHTML = '';
        if (filtered.length === 0) {
            itemsContainer.innerHTML = `<p class="text-muted" style="font-size: 0.8rem;">${noMappingsLabel}</p>`;
            return;
        }

        filtered.forEach(d => {
            const div = document.createElement('div');
            div.className = 'discovery-item';
            div.innerHTML = `
                <span class="text-accent">${d.placeholder_text}</span>
                <i class="fas fa-arrow-right mx-8 opacity-50"></i>
                <code class="text-primary">${d.metadata_key}</code>
            `;
            itemsContainer.appendChild(div);
        });
    }
    window.updateDiscoveredList = updateDiscoveredList;

    // Search/Filter logic
    if (searchInput) {
        searchInput.addEventListener('input', (e) => {
            const query = e.target.value.toLowerCase();
            document.querySelectorAll('.tree-node').forEach(node => {
                const text = node.dataset.filename.toLowerCase();
                node.style.display = text.includes(query) ? 'block' : 'none';
            });
        });
    }

    // Handle successful save
    document.body.addEventListener('discoverySaved', (evt) => {
        const formFileId = document.getElementById('form-file-id');
        const formSlideNum = document.getElementById('form-slide-num');
        if (!formFileId || !formSlideNum) return;

        const fid = formFileId.value;
        const snum = formSlideNum.value;
        const placeholderInput = document.querySelector('[name="placeholder"]');
        const keyInput = document.querySelector('[name="key"]');

        if (!placeholderInput || !keyInput) return;

        const placeholder = placeholderInput.value;
        const key = keyInput.value;

        // Optimistic update of local cache
        const newItem = {
            pptx_file_id: fid,
            slide_number: snum,
            placeholder_text: placeholder,
            metadata_key: key,
            discovered_at: new Date().toISOString()
        };

        const idx = window.allDiscovered.findIndex(d =>
            d.pptx_file_id == fid && d.slide_number == snum && d.placeholder_text == placeholder
        );
        if (idx > -1) {
            window.allDiscovered[idx] = newItem;
        } else {
            window.allDiscovered.push(newItem);
        }

        updateDiscoveredList(fid, snum);
        markExistingDiscoveries();

        placeholderInput.value = '';
        keyInput.value = '';
    });
});
