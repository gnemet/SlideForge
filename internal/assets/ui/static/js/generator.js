document.addEventListener('DOMContentLoaded', function () {
    initDragAndDrop();
    initSearch();
    updateSlideCounts();

    // Global Cleanup: Ensure "menu no buttons" rule is applied to any existing items
    document.querySelectorAll('#collection-target .slide-item button').forEach(btn => btn.remove());

    // Re-bind double-click for items already in collection (if any)
    document.querySelectorAll('#collection-target .slide-item').forEach(item => {
        item.ondblclick = (e) => {
            e.stopPropagation();
            item.remove();
            checkEmpty();
        };
    });
});

let currentDragItems = [];

function initDragAndDrop() {
    // Source side - individual slides
    const containers = document.querySelectorAll('.slides-container');
    containers.forEach(container => {
        new Sortable(container, {
            group: {
                name: 'slides',
                pull: 'clone',
                put: false
            },
            sort: false,
            animation: 150,
            onStart: function (evt) {
                // Prepare list of items to drag
                const item = evt.item;
                currentDragItems = [];

                if (item.classList.contains('selected')) {
                    // Multi-drag: grab all selected items from ANY container (global selection)
                    currentDragItems = Array.from(document.querySelectorAll('.slide-item.selected'));
                } else {
                    // Single drag
                    currentDragItems = [item];
                }

                // Optional: visual feedback
                currentDragItems.forEach(el => el.style.opacity = '0.5');
            },
            onEnd: function (evt) {
                // Restore opacity
                currentDragItems.forEach(el => el.style.opacity = '');
            }
        });
    });

    // Make PPT nodes draggable to add all slides at once
    const pptNodes = document.querySelectorAll('.ppt-node');
    pptNodes.forEach(node => {
        node.addEventListener('dragstart', (e) => {
            e.dataTransfer.setData('text/pptx-id', node.getAttribute('data-id'));
        });
    });

    // Target side
    const target = document.getElementById('collection-target');
    new Sortable(target, {
        group: 'slides',
        animation: 150,
        ghostClass: 'sortable-ghost',
        onAdd: function (evt) {
            const droppedItem = evt.item;

            // If dragging multiple items, add the others
            if (currentDragItems.length > 1) {
                // We reference the dropped item to insert others relative to it
                let refNode = droppedItem;

                // Sort drag items by their original DOM position to maintain order if possible?
                // The currentDragItems array is already in DOM order (from querySelectorAll)

                currentDragItems.forEach(sourceItem => {
                    const sourcePath = sourceItem.getAttribute('data-path');
                    const droppedPath = droppedItem.getAttribute('data-path');

                    if (sourcePath === droppedPath) {
                        processAddedItem(droppedItem, sourceItem);
                        refNode = droppedItem;
                    } else {
                        const clone = sourceItem.cloneNode(true);
                        if (refNode.nextSibling) {
                            target.insertBefore(clone, refNode.nextSibling);
                        } else {
                            target.appendChild(clone);
                        }
                        processAddedItem(clone, sourceItem);
                        refNode = clone;
                    }
                });
            } else {
                // Single item - identify its source
                const sourceItem = currentDragItems[0];
                processAddedItem(droppedItem, sourceItem);
            }
        }
    });

    // Handle PPTX drop manually
    target.addEventListener('dragover', (e) => e.preventDefault());
    target.addEventListener('drop', (e) => {
        const pptxId = e.dataTransfer.getData('text/pptx-id');
        if (pptxId) {
            e.preventDefault();
            const sourceContainer = document.getElementById('slides-' + pptxId);
            const sourceNode = sourceContainer.closest('.tree-node');
            const slides = sourceContainer.querySelectorAll('.slide-item');
            slides.forEach(slide => {
                const clone = slide.cloneNode(true);
                target.appendChild(clone);
                processAddedItem(clone, slide);
            });
            checkEmpty();
        }
    });
}

function processAddedItem(item, sourceItem) {
    const target = document.getElementById('collection-target');
    const empty = target.querySelector('.collection-empty');
    if (empty) empty.remove();

    item.style.cursor = 'grab';
    item.classList.remove('selected');

    // STRICT: Remove ALL buttons from the row. No action buttons allowed on rows!
    item.querySelectorAll('button').forEach(btn => btn.remove());

    // Resolve Compound Name: [Presentation] / [Slide Title]
    const slideInfoSpan = item.querySelector('.slide-info span');
    if (slideInfoSpan) {
        let pptxName = "";

        // Strategy 1: Use provided sourceItem
        if (sourceItem) {
            const container = sourceItem.closest('.slides-container');
            const pptNode = container ? container.previousElementSibling : null;
            if (pptNode && pptNode.classList.contains('ppt-node')) {
                pptxName = pptNode.querySelector('span').innerText.trim();
            }
        }

        // Strategy 2: If no sourceItem, try to resolve from current element tree
        if (!pptxName) {
            const container = item.closest('.slides-container');
            const pptNode = container ? container.previousElementSibling : null;
            if (pptNode && pptNode.classList.contains('ppt-node')) {
                pptxName = pptNode.querySelector('span').innerText.trim();
            }
        }

        if (pptxName) {
            const slideName = slideInfoSpan.innerText.trim();
            // Only prepend if not already in compound format
            if (!slideName.includes(' / ')) {
                slideInfoSpan.innerText = `${pptxName} / ${slideName}`;
            }
        }
    }

    // Interactions
    item.onclick = (e) => handleSlideClick(item, e);
    item.ondblclick = (e) => {
        e.stopPropagation();
        item.remove();
        checkEmpty();
    };
}

function initSearch() {
    const searchInput = document.getElementById('ppt-search');
    searchInput.addEventListener('input', function () {
        const term = this.value.toLowerCase();
        const nodes = document.querySelectorAll('.tree-node');

        nodes.forEach(node => {
            const filename = node.getAttribute('data-filename').toLowerCase();
            const tags = node.getAttribute('data-tags').toLowerCase();
            const matches = filename.includes(term) || tags.includes(term);
            node.style.display = matches ? 'block' : 'none';
        });
        updateSlideCounts();
    });
}

function toggleNode(el, e) {
    if (e) e.stopPropagation();
    el.classList.toggle('expanded');
}

function toggleFolder(el, e) {
    if (e) e.stopPropagation();
    const content = el.nextElementSibling;
    const chevron = el.querySelector('.chevron');
    el.classList.toggle('expanded');
    if (content.style.display === 'none') {
        content.style.display = 'block';
    } else {
        content.style.display = 'none';
    }
}

function handleSlideClick(item, e) {
    e.stopPropagation();

    // Alt + Shift + Click: Show JSON Data
    if (e.altKey && e.shiftKey) {
        showSlideJson(item);
        return;
    }

    // Alt + Click: Preview Slide
    if (e.altKey) {
        const path = item.getAttribute('data-path');
        const info = item.querySelector('.slide-info').innerText;
        previewSlide(path, info);
        return;
    }

    // Shift + Click: Show Metadata
    if (e.shiftKey) {
        showMeta(item);
        return;
    }

    // Ctrl + Click: Multi-select (Toggle)
    if (e.ctrlKey || e.metaKey) {
        item.classList.toggle('selected');
        return;
    }

    // Normal Click: Exclusive Select
    document.querySelectorAll('.slide-item.selected').forEach(el => {
        if (el !== item) el.classList.remove('selected');
    });
    item.classList.add('selected');

    // Handle Double Click for copy to collection (if in source)
    item.ondblclick = (e) => {
        if (item.parentElement.classList.contains('slides-container')) {
            e.stopPropagation();
            const cloned = item.cloneNode(true);
            document.getElementById('collection-target').appendChild(cloned);
            processAddedItem(cloned, item);
            checkEmpty();
        }
    };
}

function showSlideJson(item) {
    const data = {
        id: item.getAttribute('data-id'),
        path: item.getAttribute('data-path'),
        summary: item.getAttribute('data-summary'),
        content: item.getAttribute('data-content')
    };

    const modal = document.getElementById('meta-modal');
    const backdrop = document.getElementById('meta-backdrop');
    const rawBox = document.getElementById('meta-raw');

    document.getElementById('meta-summary').innerText = item.querySelector('.slide-info').innerText;

    rawBox.innerHTML = `<div class="json-viewer">${syntaxHighlightJSON(data)}</div>`;
    rawBox.style.maxHeight = 'none'; // Allow resizable container to control height

    modal.style.display = 'block';
    backdrop.style.display = 'block';

    setTimeout(() => {
        modal.classList.add('active');
        backdrop.style.opacity = '1';
    }, 10);
}

function showMeta(item) {
    const summary = item.getAttribute('data-summary') || "No summary available.";
    const raw = item.getAttribute('data-content') || "No raw content extracted.";

    const rawBox = document.getElementById('meta-raw');
    document.getElementById('meta-summary').innerText = summary;
    rawBox.innerText = raw;
    rawBox.style.maxHeight = '200px'; // Limit height for text metadata

    const modal = document.getElementById('meta-modal');
    const backdrop = document.getElementById('meta-backdrop');

    modal.style.display = 'block';
    backdrop.style.display = 'block';

    setTimeout(() => {
        modal.classList.add('active');
        backdrop.style.opacity = '1';
    }, 10);
}

function closeMeta() {
    const modal = document.getElementById('meta-modal');
    const backdrop = document.getElementById('meta-backdrop');

    modal.classList.remove('active');
    backdrop.style.opacity = '0';

    setTimeout(() => {
        modal.style.display = 'none';
        backdrop.style.display = 'none';
        // Reset resizable/draggable overrides
        modal.style.left = '';
        modal.style.top = '';
        modal.style.transform = '';
        modal.style.margin = '';
        modal.style.width = '500px';
        modal.style.height = '400px';
    }, 300);
}
// JSON Syntax Highlighting
function syntaxHighlightJSON(json) {
    if (typeof json !== 'string') {
        json = JSON.stringify(json, undefined, 2);
    }
    json = json.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    return json.replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g, function (match) {
        var cls = 'json-number';
        if (/^"/.test(match)) {
            if (/:$/.test(match)) {
                cls = 'json-key';
            } else {
                cls = 'json-string';
            }
        } else if (/true|false/.test(match)) {
            cls = 'json-boolean';
        } else if (/null/.test(match)) {
            cls = 'json-null';
        }
        return '<span class="' + cls + '">' + match + '</span>';
    });
}

// Modal Drag Logic
let activeModal = null;
let dragOffset = { x: 0, y: 0 };

function initModalDrag(e, modalId) {
    activeModal = document.getElementById(modalId);
    if (!activeModal) return;

    // Switch to absolute positioning if we start dragging
    const rect = activeModal.getBoundingClientRect();
    activeModal.style.transform = 'none';
    activeModal.style.left = rect.left + 'px';
    activeModal.style.top = rect.top + 'px';
    activeModal.style.margin = '0';

    dragOffset.x = e.clientX - rect.left;
    dragOffset.y = e.clientY - rect.top;

    document.addEventListener('mousemove', handleModalDrag);
    document.addEventListener('mouseup', stopModalDrag);
}

function handleModalDrag(e) {
    if (!activeModal) return;
    activeModal.style.left = (e.clientX - dragOffset.x) + 'px';
    activeModal.style.top = (e.clientY - dragOffset.y) + 'px';
}

function stopModalDrag() {
    document.removeEventListener('mousemove', handleModalDrag);
    document.removeEventListener('mouseup', stopModalDrag);
    activeModal = null;
}

function previewSlide(path, title) {
    const modal = document.getElementById('preview-modal');
    const backdrop = document.getElementById('preview-backdrop');
    const img = document.getElementById('preview-img');
    const titleEl = document.getElementById('preview-title');

    img.src = path;
    titleEl.innerText = title;

    modal.style.display = 'block';
    backdrop.style.display = 'block';

    // Trigger animation
    setTimeout(() => {
        modal.classList.add('active');
        backdrop.style.opacity = '1';
    }, 10);
}

function closePreview() {
    const modal = document.getElementById('preview-modal');
    const backdrop = document.getElementById('preview-backdrop');

    modal.classList.remove('active');
    backdrop.style.opacity = '0';

    setTimeout(() => {
        modal.style.display = 'none';
        backdrop.style.display = 'none';
        // Reset resizable/draggable overrides
        modal.style.left = '';
        modal.style.top = '';
        modal.style.transform = '';
        modal.style.margin = '';
    }, 300);
}

function clearCollection() {
    const target = document.getElementById('collection-target');
    target.innerHTML = '';
    checkEmpty();
}

function removeSelected() {
    const target = document.getElementById('collection-target');
    const selected = target.querySelectorAll('.slide-item.selected');
    selected.forEach(item => item.remove());
    checkEmpty();
}

function checkEmpty() {
    const target = document.getElementById('collection-target');
    if (target.children.length === 0) {
        target.innerHTML = `
            <div class="collection-empty">
                <i class="fas fa-hand-pointer fa-2x mb-16 opacity-50"></i>
                <p>Drag slides here to build your deck</p>
            </div>
        `;
    }
}

function updateSlideCounts() {
    const sourceTree = document.getElementById('source-tree');
    if (!sourceTree) return;

    // Count slides in segments that are currently visible (matching the filter)
    const visibleSlides = Array.from(sourceTree.querySelectorAll('.slide-item')).filter(s => {
        const node = s.closest('.tree-node');
        return node && node.style.display !== 'none';
    });

    // Count distinct PPTX files that are currently visible
    const visiblePPTs = Array.from(sourceTree.querySelectorAll('.tree-node')).filter(node => {
        return node.style.display !== 'none';
    });

    const countBadge = document.getElementById('source-count');
    if (countBadge) {
        const sSuffix = countBadge.getAttribute('data-suffix') || 'slides';
        const pSuffix = countBadge.getAttribute('data-pptx-suffix') || 'PPTX';
        countBadge.innerText = `${visibleSlides.length} ${sSuffix} (${visiblePPTs.length} ${pSuffix})`;
    }
}

function generateFromCollection() {
    const items = document.querySelectorAll('#collection-target .slide-item');
    const slideIds = Array.from(items).map(i => i.getAttribute('data-id'));

    if (slideIds.length === 0) {
        alert('Please collect at least one slide first!');
        return;
    }

    // Proactive feedback
    const btn = event.currentTarget;
    const originalHtml = btn.innerHTML;
    btn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Generating...';
    btn.disabled = true;

    // Simulate backend call
    setTimeout(() => {
        alert('Deck generation logic initiated for ' + slideIds.length + ' slides.');
        btn.innerHTML = originalHtml;
        btn.disabled = false;
    }, 1500);
}

// Context Menu Logic
let contextTargetItem = null;

document.addEventListener('contextmenu', function (e) {
    const slideItem = e.target.closest('.slide-item');
    if (slideItem) {
        e.preventDefault();
        showContextMenu(slideItem, e.clientX, e.clientY);
    } else {
        closeContextMenu();
    }
});

document.addEventListener('click', function (e) {
    if (!e.target.closest('#context-menu')) {
        closeContextMenu();
    }
});

function showContextMenu(item, x, y) {
    contextTargetItem = item;
    const menu = document.getElementById('context-menu');

    // Context-aware menu items
    const isSource = item.parentElement.classList.contains('slides-container');
    const addAction = menu.querySelector('[onclick="handleContextAction(\'add\')"]');
    const removeAction = menu.querySelector('[onclick="handleContextAction(\'remove\')"]');

    if (addAction) addAction.style.display = isSource ? 'flex' : 'none';
    if (removeAction) removeAction.style.display = isSource ? 'none' : 'flex';

    menu.style.position = 'fixed';

    // Adjust position to prevent overflow
    const winWidth = window.innerWidth;
    const winHeight = window.innerHeight;
    const menuWidth = 200;
    const menuHeight = menu.offsetHeight || 200;

    if (x + menuWidth > winWidth) x = winWidth - menuWidth - 10;
    if (y + menuHeight > winHeight) y = winHeight - menuHeight - 10;

    menu.style.left = x + 'px';
    menu.style.top = y + 'px';
    menu.classList.add('active');
}

function closeContextMenu() {
    const menu = document.getElementById('context-menu');
    if (menu) menu.classList.remove('active');
    contextTargetItem = null;
}

function handleContextAction(action) {
    if (!contextTargetItem) return;

    if (action === 'preview') {
        const path = contextTargetItem.getAttribute('data-path');
        const info = contextTargetItem.querySelector('.slide-info').innerText;
        previewSlide(path, info);
    } else if (action === 'meta') {
        showMeta(contextTargetItem);
    } else if (action === 'add') {
        const cloned = contextTargetItem.cloneNode(true);
        document.getElementById('collection-target').appendChild(cloned);
        processAddedItem(cloned, contextTargetItem);
        checkEmpty();
    } else if (action === 'json') {
        showSlideJson(contextTargetItem);
    } else if (action === 'remove') {
        if (contextTargetItem.parentElement.id === 'collection-target') {
            contextTargetItem.remove();
            checkEmpty();
        } else {
            alert('Cannot remove from source library.');
        }
    }

    closeContextMenu();
}
