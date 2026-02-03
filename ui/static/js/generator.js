document.addEventListener('DOMContentLoaded', function () {
    initDragAndDrop();
    initSearch();
    updateSlideCounts();
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
                    // Check if this source item corresponds to the dropped item
                    // (Sortable cloned one item, we need to identify which one it was to avoid duplication)
                    const sourcePath = sourceItem.getAttribute('data-path');
                    const droppedPath = droppedItem.getAttribute('data-path');

                    if (sourcePath === droppedPath) {
                        // This is the item Sortable already added. Just process it.
                        processAddedItem(droppedItem);
                        refNode = droppedItem; // advancing reference
                    } else {
                        // Clone and insert
                        const clone = sourceItem.cloneNode(true);

                        // Insert after the last added item to maintain sequence
                        if (refNode.nextSibling) {
                            target.insertBefore(clone, refNode.nextSibling);
                        } else {
                            target.appendChild(clone);
                        }

                        processAddedItem(clone);
                        refNode = clone; // advance reference
                    }
                });
            } else {
                // Single item case
                processAddedItem(droppedItem);
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
            const slides = sourceContainer.querySelectorAll('.slide-item');
            slides.forEach(slide => {
                const clone = slide.cloneNode(true);
                target.appendChild(clone);
                processAddedItem(clone);
            });
            checkEmpty();
        }
    });
}

function processAddedItem(item) {
    const target = document.getElementById('collection-target');
    const empty = target.querySelector('.collection-empty');
    if (empty) empty.remove();

    item.style.cursor = 'grab';
    item.classList.remove('selected'); // Don't carry over selection to target

    const path = item.getAttribute('data-path');
    const info = item.querySelector('.slide-info').innerText;
    item.onclick = (e) => handleSlideClick(item, e);

    if (!item.querySelector('.remove-slide')) {
        const removeBtn = document.createElement('button');
        removeBtn.className = 'btn btn-muted btn-sm ml-auto remove-slide';
        removeBtn.innerHTML = '<i class="fas fa-trash"></i>';
        removeBtn.onclick = (e) => {
            e.stopPropagation();
            item.remove();
            checkEmpty();
        };
        item.appendChild(removeBtn);
    }
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
    });
}

function toggleNode(el, e) {
    if (e) e.stopPropagation();
    el.classList.toggle('expanded');
}

function handleSlideClick(item, e) {
    e.stopPropagation();

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

    // Normal Click: Exclusive Select (Standard File Explorer behavior)
    document.querySelectorAll('.slide-item.selected').forEach(el => {
        if (el !== item) el.classList.remove('selected');
    });
    item.classList.add('selected');
}

function showMeta(item) {
    const summary = item.getAttribute('data-summary') || "No summary available.";
    const raw = item.getAttribute('data-content') || "No raw content extracted.";

    document.getElementById('meta-summary').innerText = summary;
    document.getElementById('meta-raw').innerText = raw;

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
    }, 300);
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
    const count = document.querySelectorAll('.slide-item').length;
    // Note: this counts all in source, might want to be more specific
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
        showContextMenu(slideItem, e.pageX, e.pageY);
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

    // Adjust position to prevent overflow
    const winWidth = window.innerWidth;
    const winHeight = window.innerHeight;

    if (x + 200 > winWidth) x = winWidth - 210;
    if (y + 150 > winHeight) y = winHeight - 160;

    menu.style.left = x + 'px';
    menu.style.top = y + 'px';
    menu.classList.add('active');
}

function closeContextMenu() {
    const menu = document.getElementById('context-menu');
    menu.classList.remove('active');
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
        processAddedItem(cloned);
        checkEmpty();
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
