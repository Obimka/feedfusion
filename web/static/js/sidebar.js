// sidebar.js - Gestion de la sidebar

function toggleSidebar() {
    const appContainer = document.querySelector('.app-container');
    const btn = document.querySelector('.sidebar-toggle i');

    appContainer.classList.toggle('sidebar-collapsed');

    // Change icon: fa-bars (closed) <-> chevron-left (open)
    if (appContainer.classList.contains('sidebar-collapsed')) {
        btn.className = 'fas fa-bars';
    } else {
        btn.className = 'fas fa-chevron-left';
    }

    // Save state to localStorage
    const isCollapsed = appContainer.classList.contains('sidebar-collapsed');
    localStorage.setItem('sidebarCollapsed', isCollapsed);
}

// Restore sidebar state on page load
function initSidebar() {
    const appContainer = document.querySelector('.app-container');
    const btn = document.querySelector('.sidebar-toggle i');
    const savedSidebarState = localStorage.getItem('sidebarCollapsed');
    
    if (savedSidebarState === 'true') {
        appContainer.classList.add('sidebar-collapsed');
        btn.className = 'fas fa-bars';
    } else {
        btn.className = 'fas fa-chevron-left';
    }
}

// Auto-initialize when DOM is loaded
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initSidebar);
} else {
    initSidebar();
}
