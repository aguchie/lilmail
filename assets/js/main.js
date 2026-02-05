// Theme Manager
class ThemeManager {
    constructor() {
        this.theme = localStorage.getItem('theme') || this.getSystemTheme();
        this.init();
    }
    
    init() {
        // Apply initial theme
        this.applyTheme(this.theme);
        
        // Listen for system theme changes
        window.matchMedia('(prefers-color-scheme: dark)')
            .addEventListener('change', (e) => {
                if (!localStorage.getItem('theme')) {
                    this.toggleTheme(e.matches ? 'dark' : 'light');
                }
            });
        
        // Setup theme toggle button if exists
        const toggleBtn = document.getElementById('theme-toggle');
        if (toggleBtn) {
            toggleBtn.addEventListener('click', () => this.toggleTheme());
        }
    }
    
    getSystemTheme() {
        return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }
    
    applyTheme(theme) {
        document.documentElement.setAttribute('data-theme', theme);
        this.theme = theme;
        
        // Update icon
        const icon = document.querySelector('#theme-toggle-icon');
        if (icon) {
            icon.textContent = theme === 'dark' ? '☀️' : '🌙';
        }
        
        // Update aria-label for accessibility
        const toggleBtn = document.getElementById('theme-toggle');
        if (toggleBtn) {
            toggleBtn.setAttribute('aria-label', 
                theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode');
        }
    }
    
    toggleTheme(newTheme = null) {
        const theme = newTheme || (this.theme === 'light' ? 'dark' : 'light');
        this.applyTheme(theme);
        localStorage.setItem('theme', theme);
        
        // Dispatch custom event
        window.dispatchEvent(new CustomEvent('themechange', { detail: { theme } }));
    }
    
    getCurrentTheme() {
        return this.theme;
    }
}

// Language Manager
function changeLanguage(lang) {
    // Save to cookie
    document.cookie = `lang=${lang}; path=/; max-age=31536000`;
    // Reload page to apply new language
    window.location.reload();
}

// Toast Notification System
class ToastManager {
    constructor() {
        this.container = this.createContainer();
    }
    
    createContainer() {
        let container = document.getElementById('toast-container');
        if (!container) {
            container = document.createElement('div');
            container.id = 'toast-container';
            container.style.cssText = 'position: fixed; bottom: 2rem; right: 2rem; z-index: 2000;';
            document.body.appendChild(container);
        }
        return container;
    }
    
    show(message, type = 'info', duration = 3000) {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.innerHTML = `
            <div style="display: flex; align-items: center; gap: 0.75rem;">
                <span style="font-size: 1.25rem;">${this.getIcon(type)}</span>
                <span>${message}</span>
            </div>
        `;
        
        this.container.appendChild(toast);
        
        // Auto remove after duration
        setTimeout(() => {
            toast.style.animation = 'slideOut 0.3s ease';
            setTimeout(() => toast.remove(), 300);
        }, duration);
    }
    
    getIcon(type) {
        const icons = {
            success: '✓',
            error: '✗',
            warning: '⚠',
            info: 'ℹ'
        };
        return icons[type] || icons.info;
    }
}

// Email Viewer Manager
class EmailViewerManager {
    constructor() {
        this.viewer = document.querySelector('.email-viewer-container');
        this.backButton = document.getElementById('email-back-btn');
        
        if (this.backButton) {
            this.backButton.addEventListener('click', () => this.close());
        }
    }
    
    open() {
        if (this.viewer) {
            this.viewer.classList.add('active');
            document.body.style.overflow = 'hidden'; // Prevent scroll on mobile
        }
    }
    
    close() {
        if (this.viewer) {
            this.viewer.classList.remove('active');
            document.body.style.overflow = '';
        }
    }
}

// Confirmation Dialog
function confirm Dialog(message, onConfirm, onCancel) {
    const overlay = document.createElement('div');
    overlay.style.cssText = `
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 3000;
    `;
    
    const dialog = document.createElement('div');
    dialog.style.cssText = `
        background: var(--bg-primary);
        border-radius: 8px;
        padding: 2rem;
        max-width: 400px;
        width: 90%;
        box-shadow: 0 4px 20px var(--shadow);
    `;
    
    dialog.innerHTML = `
        <p style="margin-bottom: 1.5rem; color: var(--text-primary);">${message}</p>
        <div style="display: flex; gap: 1rem; justify-content: flex-end;">
            <button class="btn btn-secondary" id="cancel-btn">Cancel</button>
            <button class="btn btn-primary" id="confirm-btn">Confirm</button>
        </div>
    `;
    
    overlay.appendChild(dialog);
    document.body.appendChild(overlay);
    
    dialog.querySelector('#confirm-btn').addEventListener('click', () => {
        overlay.remove();
        if (onConfirm) onConfirm();
    });
    
    dialog.querySelector('#cancel-btn').addEventListener('click', () => {
        overlay.remove();
        if (onCancel) onCancel();
    });
    
    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) {
            overlay.remove();
            if (onCancel) onCancel();
        }
    });
}

// Email Actions
const EmailActions = {
    delete: function(emailId, folder) {
        confirmDialog('このメールを削除してもよろしいですか？', () => {
            fetch(`/api/email/${emailId}`, {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${this.getToken()}`,
                    'X-Folder': folder
                }
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    toastManager.show('メールを削除しました', 'success');
                    // Remove email from list
                    document.querySelector(`[data-email-id="${emailId}"]`)?.remove();
                } else {
                    toastManager.show('削除に失敗しました', 'error');
                }
            })
            .catch(error => {
                console.error('Delete error:', error);
                toastManager.show('エラーが発生しました', 'error');
            });
        });
    },
    
    getToken: function() {
        // Get token from page data or cookie
        return document.body.dataset.token || '';
    }
};

// Initialize on DOM ready
document.addEventListener('DOMContentLoaded', () => {
    // Initialize theme manager
    window.themeManager = new ThemeManager();
    
    // Initialize toast manager
    window.toastManager = new ToastManager();
    
    // Initialize email viewer manager (mobile)
    if (window.innerWidth < 768) {
        window.emailViewerManager = new EmailViewerManager();
    }
    
    // HTMX event listeners
    document.body.addEventListener('htmx:afterSwap', (event) => {
        // Open email viewer on mobile
        if (event.detail.target.id === 'email-viewer' && window.innerWidth < 768) {
            if (window.emailViewerManager) {
                window.emailViewerManager.open();
            }
        }
    });
    
    // Handle window resize
    let resizeTimeout;
    window.addEventListener('resize', () => {
        clearTimeout(resizeTimeout);
        resizeTimeout = setTimeout(() => {
            if (window.innerWidth >= 768 && window.emailViewerManager) {
                window.emailViewerManager.close();
            }
        }, 250);
    });
});

// Add slideOut animation to CSS
const style = document.createElement('style');
style.textContent = `
    @keyframes slideOut {
        from {
            transform: translateX(0);
            opacity: 1;
        }
        to {
            transform: translateX(400px);
            opacity: 0;
        }
    }
`;
document.head.appendChild(style);
