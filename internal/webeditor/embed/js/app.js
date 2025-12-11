/**
 * SteelClock Configuration Editor - Main Application
 * Phase 2: Schema-driven form editing with JSON view option
 */

class ConfigEditor {
    constructor() {
        this.config = null;
        this.originalConfig = null;
        this.schema = null;
        this.schemaProcessor = null;
        this.formBuilder = null;
        this.widgetEditor = null;
        this.isDirty = false;
        this.currentView = 'form'; // 'form' or 'json'
        this.validationErrors = [];
        this.validationTimeout = null;
        this.notificationTimeout = null;
        this.profiles = [];

        // DOM elements
        this.formContainer = document.getElementById('form-container');
        this.formContent = document.getElementById('form-content');
        this.jsonContainer = document.getElementById('json-container');
        this.editorEl = document.getElementById('config-editor');
        this.saveBtn = document.getElementById('btn-save');
        this.reloadBtn = document.getElementById('btn-reload');
        this.viewFormBtn = document.getElementById('btn-view-form');
        this.viewJsonBtn = document.getElementById('btn-view-json');
        this.notificationEl = document.getElementById('notification');
        this.validationEl = document.getElementById('validation-status');
        this.statusEl = document.getElementById('status');
        this.profileSelector = document.getElementById('profile-selector');
        this.profileSelect = document.getElementById('profile-select');
        this.themeToggle = document.getElementById('toggle-theme');
    }

    /**
     * Initialize the editor
     */
    async init() {
        // Set up event listeners
        this.saveBtn.addEventListener('click', () => this.save());
        this.reloadBtn.addEventListener('click', () => this.reload());
        this.editorEl.addEventListener('input', () => {
            this.markDirty();
            this.scheduleValidation();
        });
        this.themeToggle.addEventListener('click', (e) => {
            e.preventDefault();
            this.toggleTheme();
        });

        // Profile selector
        this.profileSelect.addEventListener('change', () => this.switchProfile());

        // View toggle
        this.viewFormBtn.addEventListener('click', () => this.switchView('form'));
        this.viewJsonBtn.addEventListener('click', () => this.switchView('json'));

        // Warn before leaving with unsaved changes
        window.addEventListener('beforeunload', (e) => {
            if (this.isDirty) {
                e.preventDefault();
                e.returnValue = '';
            }
        });

        // Load schema first
        await this.loadSchema();

        // Load profiles
        await this.loadProfiles();

        // Load initial configuration
        await this.loadConfig();

        this.setStatus('Ready');
    }

    /**
     * Load JSON schema
     */
    async loadSchema() {
        try {
            this.setStatus('Loading schema...');
            this.schema = await API.getSchema();
            this.schemaProcessor = new SchemaProcessor(this.schema);
            this.schemaProcessor.init();
            this.formBuilder = new FormBuilder(this.schemaProcessor, () => this.markDirty());
            this.widgetEditor = new WidgetEditor(this.schemaProcessor, this.formBuilder, () => this.markDirty());
        } catch (err) {
            console.warn('Failed to load schema:', err);
            this.showNotification('Schema not available - using JSON view only', 'warning');
            // Force JSON view if schema fails
            this.switchView('json');
            this.viewFormBtn.disabled = true;
        }
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

            // Render form if schema is available
            if (this.schemaProcessor && this.currentView === 'form') {
                this.renderForm();
            }

            this.showNotification('Configuration loaded', 'success');
            this.setStatus('Ready');
        } catch (err) {
            this.showNotification('Failed to load: ' + err.message, 'error');
            this.setStatus('Error');
        }
    }

    /**
     * Render the form view
     */
    renderForm() {
        if (!this.schemaProcessor || !this.config) {
            this.formContent.innerHTML = '<p>Schema or configuration not available.</p>';
            return;
        }

        this.formContent.innerHTML = '';

        // Render sections
        const generalSection = this.formBuilder.renderGlobalConfig(this.config, () => this.onFormChange());
        const displaySection = this.formBuilder.renderDisplayConfig(this.config, () => this.onFormChange());
        const defaultsSection = this.formBuilder.renderDefaultsConfig(this.config, () => this.onFormChange());

        // Use WidgetEditor for full widget editing
        const widgetsSection = this.widgetEditor
            ? this.widgetEditor.renderWidgetsSection(this.config, () => this.onFormChange())
            : this.formBuilder.renderWidgetsSummary(this.config);

        this.formContent.appendChild(generalSection);
        this.formContent.appendChild(displaySection);
        this.formContent.appendChild(defaultsSection);
        this.formContent.appendChild(widgetsSection);
    }

    /**
     * Handle form field changes
     */
    onFormChange() {
        // Update JSON editor with current config
        this.editorEl.value = JSON.stringify(this.config, null, 2);
        this.scheduleValidation();
    }

    /**
     * Schedule validation with debounce
     */
    scheduleValidation() {
        if (this.validationTimeout) {
            clearTimeout(this.validationTimeout);
        }
        this.validationTimeout = setTimeout(() => this.validate(), 500);
    }

    /**
     * Validate current configuration
     */
    async validate() {
        try {
            let configToValidate;
            if (this.currentView === 'json') {
                try {
                    configToValidate = JSON.parse(this.editorEl.value);
                } catch (err) {
                    this.showValidationStatus(false, ['Invalid JSON: ' + err.message]);
                    return;
                }
            } else {
                configToValidate = this.config;
            }

            const result = await API.validateConfig(configToValidate);
            this.validationErrors = result.errors || [];
            this.showValidationStatus(result.valid, this.validationErrors);
        } catch (err) {
            // Silent fail for validation - don't interrupt user
            console.warn('Validation failed:', err);
        }
    }

    /**
     * Show validation status
     * @param {boolean} valid - Whether config is valid
     * @param {string[]} errors - Array of error messages
     */
    showValidationStatus(valid, errors) {
        if (valid) {
            this.validationEl.className = 'validation-valid';
            this.validationEl.textContent = 'Configuration is valid';
            this.validationEl.style.display = 'block';
            // Hide after 3 seconds if valid
            setTimeout(() => {
                if (this.validationEl.className === 'validation-valid') {
                    this.validationEl.style.display = 'none';
                }
            }, 3000);
        } else {
            this.validationEl.className = 'validation-error';
            this.validationEl.innerHTML = '<strong>Validation errors:</strong><ul>' +
                errors.map(e => `<li>${this.escapeHtml(e)}</li>`).join('') +
                '</ul>';
            this.validationEl.style.display = 'block';
        }
    }

    /**
     * Escape HTML special characters
     * @param {string} text - Text to escape
     * @returns {string} Escaped text
     */
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    /**
     * Switch between form and JSON views
     * @param {string} view - 'form' or 'json'
     */
    switchView(view) {
        if (view === this.currentView) return;

        // Sync data between views before switching
        if (this.currentView === 'json') {
            // Switching from JSON to Form - parse JSON and update config
            try {
                this.config = JSON.parse(this.editorEl.value);
            } catch (err) {
                this.showNotification('Invalid JSON - please fix before switching to Form view', 'error');
                return;
            }
        } else {
            // Switching from Form to JSON - update textarea
            this.editorEl.value = JSON.stringify(this.config, null, 2);
        }

        this.currentView = view;

        // Update UI
        if (view === 'form') {
            this.formContainer.style.display = 'block';
            this.jsonContainer.style.display = 'none';
            this.viewFormBtn.setAttribute('aria-pressed', 'true');
            this.viewFormBtn.classList.remove('secondary');
            this.viewJsonBtn.setAttribute('aria-pressed', 'false');
            this.viewJsonBtn.classList.add('secondary');

            // Re-render form with current config
            this.renderForm();
        } else {
            this.formContainer.style.display = 'none';
            this.jsonContainer.style.display = 'block';
            this.viewFormBtn.setAttribute('aria-pressed', 'false');
            this.viewFormBtn.classList.add('secondary');
            this.viewJsonBtn.setAttribute('aria-pressed', 'true');
            this.viewJsonBtn.classList.remove('secondary');
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

            // Get config from current view
            let configToSave;
            if (this.currentView === 'json') {
                try {
                    configToSave = JSON.parse(this.editorEl.value);
                } catch (err) {
                    this.showNotification('Invalid JSON: ' + err.message, 'error');
                    this.setStatus('Error');
                    return;
                }
            } else {
                configToSave = this.config;
            }

            // Save to server
            const result = await API.saveConfig(configToSave);

            this.config = configToSave;
            this.originalConfig = JSON.stringify(configToSave, null, 2);
            this.editorEl.value = this.originalConfig;
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
     * Load available profiles and display selector
     */
    async loadProfiles() {
        try {
            this.profiles = await API.getProfiles();

            if (this.profiles.length > 1) {
                // Show profile selector if there are multiple profiles
                this.profileSelector.style.display = 'flex';
                this.profileSelect.innerHTML = '';

                for (const profile of this.profiles) {
                    const opt = document.createElement('option');
                    opt.value = profile.path;
                    opt.textContent = profile.name;
                    if (profile.is_active) {
                        opt.selected = true;
                    }
                    this.profileSelect.appendChild(opt);
                }
            } else {
                this.profileSelector.style.display = 'none';
            }
        } catch (err) {
            // Profile info is optional, don't show error
            this.profileSelector.style.display = 'none';
        }
    }

    /**
     * Switch to a different profile
     */
    async switchProfile() {
        const selectedPath = this.profileSelect.value;
        if (!selectedPath) return;

        // Check for unsaved changes
        if (this.isDirty) {
            if (!confirm('You have unsaved changes. Switch profile anyway?')) {
                // Reset select to current active profile
                const active = this.profiles.find(p => p.is_active);
                if (active) {
                    this.profileSelect.value = active.path;
                }
                return;
            }
        }

        try {
            this.setStatus('Switching profile...');
            await API.switchProfile(selectedPath);

            // Update profile selection state
            this.profiles.forEach(p => {
                p.is_active = (p.path === selectedPath);
            });

            // Reload configuration
            await this.loadConfig();
            this.showNotification('Profile switched successfully', 'success');
        } catch (err) {
            this.showNotification('Failed to switch profile: ' + err.message, 'error');
            // Reset select to current active profile
            const active = this.profiles.find(p => p.is_active);
            if (active) {
                this.profileSelect.value = active.path;
            }
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
        // Clear any pending hide timeout
        if (this.notificationTimeout) {
            clearTimeout(this.notificationTimeout);
            this.notificationTimeout = null;
        }

        // Reset state
        this.notificationEl.classList.remove('fade-out');
        this.notificationEl.style.display = 'block';
        this.notificationEl.className = type;

        // Build content with close button for errors
        if (type === 'error') {
            this.notificationEl.innerHTML = `
                <span class="notification-message">${this.escapeHtml(message)}</span>
                <button class="notification-close" onclick="window.configEditor.hideNotification()" title="Dismiss">&times;</button>
            `;
        } else {
            this.notificationEl.textContent = message;
        }

        // Auto-hide success and warning messages
        const hideDelay = type === 'success' ? 3000 : (type === 'warning' ? 5000 : 0);
        if (hideDelay > 0) {
            this.notificationTimeout = setTimeout(() => {
                this.hideNotification();
            }, hideDelay);
        }
    }

    /**
     * Hide the notification with fade-out animation
     */
    hideNotification() {
        this.notificationEl.classList.add('fade-out');
        setTimeout(() => {
            if (this.notificationEl.classList.contains('fade-out')) {
                this.notificationEl.style.display = 'none';
                this.notificationEl.className = '';
            }
        }, 300);
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
    window.configEditor = editor; // Expose globally for notification close button
    editor.loadTheme();
    editor.init().catch(err => {
        console.error('Failed to initialize editor:', err);
        document.getElementById('notification').textContent = 'Failed to initialize: ' + err.message;
        document.getElementById('notification').className = 'error';
    });
});
