// I18n Client for dynamic translations
class I18nClient {
    constructor(lang) {
        this.lang = lang || this.detectLanguage();
        this.translations = {};
        this.ready = false;
        this.loadTranslations();
    }

    detectLanguage() {
        // Check cookie first
        const cookieLang = this.getCookie('lang');
        if (cookieLang) return cookieLang;

        // Check browser language
        const browserLang = navigator.language || navigator.userLanguage;
        return browserLang.startsWith('ja') ? 'ja' : 'en';
    }

    getCookie(name) {
        const value = `; ${document.cookie}`;
        const parts = value.split(`; ${name}=`);
        if (parts.length === 2) return parts.pop().split(';').shift();
        return null;
    }

    async loadTranslations() {
        try {
            const response = await fetch(`/api/i18n/${this.lang}`);
            if (response.ok) {
                this.translations = await response.json();
                this.ready = true;
                // Dispatch event for components waiting for translations
                window.dispatchEvent(new CustomEvent('i18nready', { detail: { lang: this.lang } }));
            }
        } catch (error) {
            console.error('Failed to load translations:', error);
        }
    }

    t(key, fallback) {
        if (!this.ready) {
            return fallback || key;
        }
        return this.translations[key] || fallback || key;
    }

    // Template function for parameterized translations
    tWithData(key, data, fallback) {
        let message = this.t(key, fallback);

        // Replace {{.Key}} with data values
        Object.keys(data).forEach(dataKey => {
            message = message.replace(new RegExp(`{{.${dataKey}}}`, 'g'), data[dataKey]);
        });

        return message;
    }

    async setLanguage(lang) {
        this.lang = lang;
        this.ready = false;
        await this.loadTranslations();

        // Update cookie
        document.cookie = `lang=${lang}; path=/; max-age=31536000`;

        // Reload page to apply translations
        window.location.reload();
    }
}

// Initialize global i18n instance
window.i18n = new I18nClient();
