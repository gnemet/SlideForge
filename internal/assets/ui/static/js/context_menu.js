document.addEventListener('DOMContentLoaded', function () {
    console.log("SlideForge Context Menu Initialized");

    let menu = document.getElementById('pptx-context-menu');
    if (!menu) {
        menu = document.createElement('div');
        menu.className = 'custom-context-menu';
        menu.id = 'pptx-context-menu';
        document.body.appendChild(menu);
    }

    let currentFileID = null;

    // Use capture phase to catch events early
    document.addEventListener('contextmenu', function (e) {
        const card = e.target.closest('.pptx-card');
        if (card) {
            console.log("Context menu triggered for card:", card.getAttribute('data-filename'));
            e.preventDefault();
            e.stopPropagation();

            currentFileID = card.getAttribute('data-id');
            const filename = card.getAttribute('data-filename');

            showMenu(e.clientX, e.clientY, currentFileID, filename);
        } else {
            hideMenu();
        }
    }, true);

    document.addEventListener('click', function (e) {
        if (menu && !menu.contains(e.target)) {
            hideMenu();
        }
    });

    function showMenu(clientX, clientY, fileID, filename) {
        menu.innerHTML = `
            <div class="menu-item" hx-post="/reprocess-file?fileID=${fileID}" hx-swap="none" hx-trigger="click">
                <i class="fas fa-sync-alt"></i> Reprocess
            </div>
            <div class="menu-item" hx-post="/copy-to-stage?fileID=${fileID}" hx-swap="none" hx-trigger="click">
                <i class="fas fa-file-export"></i> Copy to Stage
            </div>
            <div class="menu-separator"></div>
            <div class="menu-item" onclick="event.stopPropagation(); window.location='/editor/comments?fileID=${fileID}'">
                <i class="fas fa-edit"></i> Edit Placeholders
            </div>
            <div class="menu-separator"></div>
            <div class="menu-item danger" 
                 hx-post="/delete-file?fileID=${fileID}" 
                 hx-confirm="Are you sure you want to delete ${filename}?"
                 hx-swap="none" 
                 hx-trigger="click">
                <i class="fas fa-trash-alt"></i> Delete
            </div>
        `;

        menu.style.display = 'block';

        // Positioning logic
        const menuWidth = menu.offsetWidth || 180;
        const menuHeight = menu.offsetHeight || 200;
        const windowWidth = window.innerWidth;
        const windowHeight = window.innerHeight;

        let posX = clientX;
        let posY = clientY;

        if ((posX + menuWidth) > windowWidth) posX -= menuWidth;
        if ((posY + menuHeight) > windowHeight) posY -= menuHeight;

        menu.style.left = posX + 'px';
        menu.style.top = posY + 'px';

        if (window.htmx) {
            htmx.process(menu);
        }
    }

    function hideMenu() {
        if (menu) menu.style.display = 'none';
    }
});
