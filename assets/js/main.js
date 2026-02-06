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
    if (window.i18n) {
        window.i18n.setLanguage(lang);
    } else {
        // Fallback if i18n not loaded
        document.cookie = `lang=${lang}; path=/; max-age=31536000`;
        window.location.reload();
    }
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
function confirmDialog(message, onConfirm, onCancel) {
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
    delete: function (emailId, folder) {
        const message = window.i18n ? window.i18n.t('confirm_delete_email', 'このメールを削除してもよろしいですか？') : 'このメールを削除してもよろしいですか？';
        confirmDialog(message, () => {
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
                        const msg = window.i18n ? window.i18n.t('message_deleted', 'メールを削除しました') : 'メールを削除しました';
                        toastManager.show(msg, 'success');
                        // Remove email from list
                        document.querySelector(`[data-email-id="${emailId}"]`)?.remove();
                    } else {
                        const msg = window.i18n ? window.i18n.t('error_delete_failed', '削除に失敗しました') : '削除に失敗しました';
                        toastManager.show(msg, 'error');
                    }
                })
                .catch(error => {
                    console.error('Delete error:', error);
                    const msg = window.i18n ? window.i18n.t('message_error', 'エラーが発生しました') : 'エラーが発生しました';
                    toastManager.show(msg, 'error');
                });
        });
    },

    reply: function (emailId, folder) {
        this.fetchAndOpenCompose(`/api/reply/${emailId}`, folder);
    },

    replyAll: function (emailId, folder) {
        this.fetchAndOpenCompose(`/api/reply-all/${emailId}`, folder);
    },

    forward: function (emailId, folder) {
        this.fetchAndOpenCompose(`/api/forward/${emailId}`, folder);
    },

    move: function (emailId, folder) {
        // Dispatch event to open Move Modal
        window.dispatchEvent(new CustomEvent('open-move-modal', {
            detail: { emailId, currentFolder: folder }
        }));
    },

    fetchAndOpenCompose: function (url, folder) {
        fetch(url, {
            headers: {
                'Authorization': `Bearer ${this.getToken()}`,
                'X-CSRF-Token': this.getCSRFToken(),
                'X-Folder': folder
            }
        })
            .then(res => res.json())
            .then(data => {
                if (data.success) {
                    window.dispatchEvent(new CustomEvent('open-compose-with-data', { detail: data.data }));
                } else {
                    toastManager.show(data.error || 'Failed to load data', 'error');
                }
            })
            .catch(err => {
                console.error(err);
                toastManager.show('Network error', 'error');
            });
    },

    getToken: function () {
        // Get token from page data or cookie or localStorage
        return localStorage.getItem('token') || document.body.dataset.token || '';
    },

    getCSRFToken: function () {
        return document.querySelector('meta[name="csrf-token"]')?.getAttribute('content') || '';
    }
};

// Notification Manager
class NotificationManager {
    constructor() {
        this.connected = false;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.init();
    }

    init() {
        if (!window.EventSource) {
            console.warn('SSE not supported');
            return;
        }

        this.connect();
    }

    connect() {
        // Close existing connection if any
        if (this.evtSource) {
            this.evtSource.close();
        }

        this.evtSource = new EventSource('/events');

        this.evtSource.onopen = () => {
            console.log('SSE Connected');
            this.connected = true;
            this.reconnectAttempts = 0;
        };

        this.evtSource.onmessage = (e) => {
            try {
                const data = JSON.parse(e.data);
                this.handleNotification(data);
            } catch (err) {
                console.error('Error parsing notification:', err);
            }
        };

        this.evtSource.onerror = (e) => {
            // EventSource error often happens on disconnect or network issue
            // Check readyState: 0=CONNECTING, 1=OPEN, 2=CLOSED
            if (this.evtSource.readyState === 2) {
                console.log('SSE connection closed');
            } else {
                console.error('SSE Error:', e);
            }

            this.connected = false;
            this.evtSource.close();

            // Reconnect with exponential backoff
            if (this.reconnectAttempts < this.maxReconnectAttempts) {
                const timeout = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
                this.reconnectAttempts++;
                console.log(`Reconnecting in ${timeout}ms...`);
                setTimeout(() => this.connect(), timeout);
            }
        };
    }

    handleNotification(notification) {
        console.log('Received notification:', notification);

        // Use i18n if available
        const t = (key, defaultText) => window.i18n ? window.i18n.t(key, defaultText) : defaultText;

        switch (notification.type) {
            case 'new_email':
                // Show toast
                const from = notification.data?.from || 'Unknown';
                const subject = notification.data?.subject || 'No Subject';
                toastManager.show(`${t('new_email', 'New Email')}: ${from} - ${subject}`, 'info', 5000);

                // Optional: Trigger HTMX refresh for Inbox if needed
                // For now just toast is enough as per requirements
                break;

            case 'deleted':
                const deletedId = notification.data?.email_id;
                if (deletedId) {
                    // Remove from list if present in DOM
                    const el = document.querySelector(`[data-email-id="${deletedId}"]`);
                    // Or if thread view
                    const threadEl = document.querySelector(`[data-thread-id="${deletedId}"]`); // Assuming thread ID matches or we handle it

                    if (el) {
                        el.remove();
                        // Also show toast if it wasn't us who deleted it (hard to distinguish here without more data, but acceptable)
                        // toastManager.show(t('notification_email_deleted_remote', 'Email deleted elsewhere'), 'info');
                    }
                }
                break;

            case 'status_change':
                const emailId = notification.data?.email_id;
                const status = notification.data?.status; // "read" or "unread"
                if (emailId && status) {
                    // Update email list item style
                    // Note: .email-item might have .unread class
                    const el = document.querySelector(`[data-email-id="${emailId}"]`);
                    if (el) {
                        // Check structure of email item in templates/partials/email-list.html
                        // It usually has class "email-item unread"
                        const itemDiv = el.querySelector('.email-item') || el; // Depending on where data-email-id is placed

                        // Actually in email-list.html: <div class="hover:bg-gray-50..." ...> <div class="px-4 py-3"> ... <span ... fontWeight ...>
                        // Wait, in email-list.html:
                        // <div ... hx-get... @click...> <div class="px-4 py-3"> <div ...> <span class="font-medium ...">
                        // It doesn't use "unread" class explicitly in the partial snippet I saw earlier (Step 36 lines 186+).
                        // Ah, Step 36 lines 186-202: It uses standard tailwind classes.
                        // Wait, how is "unread" style applied? 
                        // I need to check `email-list.html` again. 
                        // Step 36 lines 186-202 seem to NOT have conditional bolding for unread?
                        // "span class='font-medium text-gray-900 truncate'" (Line 194)
                        // "h3 class='text-sm font-semibold text-gray-900 ...'" (Line 197)

                        // Check `templates/partials/email-list.html` to be sure. 
                        // Or `handlers/web/email.go` logic passing `Emails`.

                        if (status === 'read') {
                            itemDiv.classList.remove('font-bold', 'unread'); // Remove potential unread classes
                            itemDiv.classList.add('read');
                            // Find subject/sender to un-bold if needed
                            const subject = itemDiv.querySelector('.email-subject, h3');
                            if (subject) subject.classList.remove('font-bold', 'font-semibold');
                        } else {
                            itemDiv.classList.add('unread');
                            const subject = itemDiv.querySelector('.email-subject, h3');
                            if (subject) subject.classList.add('font-semibold');
                        }
                    }
                }
                break;
        }
    }
}

// Initialize on DOM ready
document.addEventListener('DOMContentLoaded', () => {
    // Initialize theme manager
    window.themeManager = new ThemeManager();

    // Initialize toast manager
    window.toastManager = new ToastManager();

    // Initialize notification manager
    window.notificationManager = new NotificationManager();

    // Initialize email viewer manager (mobile)
    if (window.innerWidth < 768) {
        window.emailViewerManager = new EmailViewerManager();
    }

    // HTMX event listeners
    document.body.addEventListener('htmx:configRequest', function (evt) {
        // Add CSRF Token
        const csrfToken = document.querySelector('meta[name="csrf-token"]')?.getAttribute('content');
        if (csrfToken) {
            evt.detail.headers['X-CSRF-Token'] = csrfToken;
        }
    });

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
