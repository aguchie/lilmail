// Notification System Client
class NotificationClient {
    constructor() {
        this.eventSource = null;
        this.isConnected = false;
        this.reconnectDelay = 1000;
        this.maxReconnectDelay = 30000;
    }

    connect() {
        if (this.isConnected) return;

        try {
            // Connect to SSE endpoint
            this.eventSource = new EventSource('/api/notifications/stream');

            this.eventSource.onopen = () => {
                console.log('Notification stream connected');
                this.isConnected = true;
                this.reconnectDelay = 1000;
            };

            this.eventSource.onmessage = (event) => {
                try {
                    const notification = JSON.parse(event.data);
                    this.handleNotification(notification);
                } catch (error) {
                    console.error('Failed to parse notification:', error);
                }
            };

            this.eventSource.onerror = (error) => {
                console.error('Notification stream error:', error);
                this.isConnected = false;
                this.reconnect();
            };
        } catch (error) {
            console.error('Failed to connect to notification stream:', error);
            this.reconnect();
        }
    }

    reconnect() {
        if (this.eventSource) {
            this.eventSource.close();
        }

        setTimeout(() => {
            console.log('Reconnecting to notification stream...');
            this.connect();

            // Increase delay for next reconnection (exponential backoff)
            this.reconnectDelay = Math.min(
                this.reconnectDelay * 2,
                this.maxReconnectDelay
            );
        }, this.reconnectDelay);
    }

    handleNotification(notification) {
        // Show toast notification
        if (window.toastManager) {
            const message = this.getNotificationMessage(notification);
            window.toastManager.show(message, notification.type || 'info');
        }

        // Update UI based on notification type
        switch (notification.type) {
            case 'new_email':
                this.handleNewEmail(notification);
                break;
            case 'email_sent':
                this.handleEmailSent(notification);
                break;
            case 'email_deleted':
                this.handleEmailDeleted(notification);
                break;
            default:
                console.log('Unknown notification type:', notification.type);
        }

        // Dispatch custom event for other components
        window.dispatchEvent(new CustomEvent('notification', {
            detail: notification
        }));
    }

    getNotificationMessage(notification) {
        const i18n = window.i18n;

        switch (notification.type) {
            case 'new_email':
                return i18n ? i18n.t('notification_new_email', '新着メール') : '新着メール';
            case 'email_sent':
                return i18n ? i18n.t('notification_email_sent', 'メール送信完了') : 'メール送信完了';
            case 'email_deleted':
                return i18n ? i18n.t('notification_email_deleted', 'メール削除') : 'メール削除';
            default:
                return notification.message || '';
        }
    }

    handleNewEmail(notification) {
        // Refresh email list if user is on inbox
        if (window.location.pathname === '/inbox' || window.location.pathname === '/') {
            // Use HTMX to refresh email list
            if (window.htmx) {
                const emailList = document.getElementById('email-list');
                if (emailList) {
                    htmx.trigger(emailList, 'refresh');
                }
            }
        }

        // Update unread count
        const unreadBadge = document.getElementById('unread-count');
        if (unreadBadge && notification.unread_count) {
            unreadBadge.textContent = notification.unread_count;
            unreadBadge.classList.remove('hidden');
        }
    }

    handleEmailSent(notification) {
        // Optionally refresh sent folder if viewing it
        if (window.location.pathname.includes('/folder/Sent')) {
            setTimeout(() => {
                window.location.reload();
            }, 1000);
        }
    }

    handleEmailDeleted(notification) {
        // Remove deleted email from DOM
        if (notification.email_id) {
            const emailElement = document.querySelector(`[data-email-id="${notification.email_id}"]`);
            if (emailElement) {
                emailElement.remove();
            }
        }
    }

    disconnect() {
        if (this.eventSource) {
            this.eventSource.close();
            this.isConnected = false;
        }
    }
}

// Initialize notification client on page load
document.addEventListener('DOMContentLoaded', () => {
    window.notificationClient = new NotificationClient();
    window.notificationClient.connect();
});

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    if (window.notificationClient) {
        window.notificationClient.disconnect();
    }
});
