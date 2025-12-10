/**
 * SteelClock Configuration Editor - Main Application
 * Phase 1: Raw JSON editing with load/save functionality
 */

class ConfigEditor {
    constructor() {
        this.config = null;
        this.originalConfig = null;
        this.isDirty = false;

        // DOM elements
        this.editorEl = document.getElementById('config-editor');
        this.saveBtn = document.getElementById('btn-save');
        this.reloadBtn = document.getElementById('btn-reload');
        this.notificationEl = document.getElementById('notification');
        this.statusEl = document.getElementById('status');
        this.profileInfoEl = document.getElementById('profile-info');
        this.themeToggle = document.getElementById('toggle-theme');
    }

    /**
     * Initialize the editor
     */
    async init() {
        // Set up event listeners
        this.saveBtn.addEventListener('click', () => this.save());
        this.reloadBtn.addEventListener('click', () => this.reload());
        this.editorEl.addEventListener('input', () => this.markDirty());
        this.themeToggle.addEventListener('click', (e) => {
            e.preventDefault();
            this.toggleTheme();
        });

        // Warn before leaving with unsaved changes
        window.addEventListener('beforeunload', (e) => {
            if (this.isDirty) {
                e.preventDefault();
                e.returnValue = '';
            }
        });

        // Load initial configuration
        await this.loadConfig();

        // Load profile info
        await this.loadProfileInfo();

        this.setStatus('Ready');
    }

    /**
     * Load configuration from server
     */
    async loadConfig() {
        try {
            this.setStatus('Loading...');
            this.config = await API.getConfig();
            this.originalConfig = JSON.stringify(this.config, null, 2);
            this.editorEl.value = this.originalConfig;
            this.isDirty = false;
            this.saveBtn.classList.remove('has-changes');
            this.showNotification('Configuration loaded', 'success');
            this.setStatus('Ready');
        } catch (err) {
            this.showNotification('Failed to load: ' + err.message, 'error');
            this.setStatus('Error');
        }
    }

    /**
     * Reload configuration (discard changes)
     */
    async reload() {
        if (this.isDirty) {
            if (!confirm('Discard unsaved changes and reload from file?')) {
                return;
            }
        }
        await this.loadConfig();
    }

    /**
     * Save configuration to server
     */
    async save() {
        try {
            this.setStatus('Saving...');

            // Parse JSON to validate
            let parsed;
            try {
                parsed = JSON.parse(this.editorEl.value);
            } catch (err) {
                this.showNotification('Invalid JSON: ' + err.message, 'error');
                this.setStatus('Error');
                return;
            }

            // Save to server
            const result = await API.saveConfig(parsed);

            this.config = parsed;
            this.originalConfig = JSON.stringify(parsed, null, 2);
            this.isDirty = false;
            this.saveBtn.classList.remove('has-changes');

            let message = result.message || 'Configuration saved';
            if (result.warning) {
                this.showNotification(message + ' (Warning: ' + result.warning + ')', 'warning');
            } else {
                this.showNotification(message, 'success');
            }
            this.setStatus('Saved');

        } catch (err) {
            this.showNotification('Failed to save: ' + err.message, 'error');
            this.setStatus('Error');
        }
    }

    /**
     * Load and display profile information
     */
    async loadProfileInfo() {
        try {
            const profiles = await API.getProfiles();
            const active = profiles.find(p => p.is_active);

            if (active) {
                this.profileInfoEl.textContent = `Profile: ${active.name}`;
            } else if (profiles.length > 0) {
                this.profileInfoEl.textContent = `${profiles.length} profile(s) available`;
            } else {
                this.profileInfoEl.textContent = '';
            }
        } catch (err) {
            // Profile info is optional, don't show error
            this.profileInfoEl.textContent = '';
        }
    }

    /**
     * Mark configuration as modified
     */
    markDirty() {
        if (!this.isDirty) {
            this.isDirty = true;
            this.saveBtn.classList.add('has-changes');
            this.setStatus('Modified');
        }
    }

    /**
     * Show a notification message
     * @param {string} message - The message to show
     * @param {string} type - 'success', 'error', or 'warning'
     */
    showNotification(message, type) {
        this.notificationEl.textContent = message;
        this.notificationEl.className = type;

        // Auto-hide success messages after 3 seconds
        if (type === 'success') {
            setTimeout(() => {
                if (this.notificationEl.className === 'success') {
                    this.notificationEl.className = '';
                    this.notificationEl.style.display = 'none';
                }
            }, 3000);
        }
    }

    /**
     * Set the status bar text
     * @param {string} text - The status text
     */
    setStatus(text) {
        this.statusEl.textContent = text;
    }

    /**
     * Toggle between light and dark theme
     */
    toggleTheme() {
        const html = document.documentElement;
        const currentTheme = html.getAttribute('data-theme');
        const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
        html.setAttribute('data-theme', newTheme);
        localStorage.setItem('theme', newTheme);
    }

    /**
     * Load saved theme preference
     */
    loadTheme() {
        const savedTheme = localStorage.getItem('theme');
        if (savedTheme) {
            document.documentElement.setAttribute('data-theme', savedTheme);
        }
    }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    const editor = new ConfigEditor();
    editor.loadTheme();
    editor.init().catch(err => {
        console.error('Failed to initialize editor:', err);
        document.getElementById('notification').textContent = 'Failed to initialize: ' + err.message;
        document.getElementById('notification').className = 'error';
    });
});
