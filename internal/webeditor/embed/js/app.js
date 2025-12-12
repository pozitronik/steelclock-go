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
        this.statusEl = document.getElementById('status');
        this.profileSelector = document.getElementById('profile-selector');
        this.profileSelect = document.getElementById('profile-select');
        this.newProfileBtn = document.getElementById('btn-new-profile');
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
        });
        this.themeToggle.addEventListener('click', (e) => {
            e.preventDefault();
            this.toggleTheme();
        });

        // Profile selector
        this.profileSelect.addEventListener('change', () => this.switchProfile());
        this.newProfileBtn.addEventListener('click', () => this.createNewProfile());

        // View toggle
        this.viewFormBtn.addEventListener('click', () => this.switchView('form'));
        this.viewJsonBtn.addEventListener('click', () => this.switchView('json'));

        // Warn before leaving with unsaved changes (temporarily disabled)
        // window.addEventListener('beforeunload', (e) => {
        //     if (this.isDirty) {
        //         e.preventDefault();
        //         e.returnValue = '';
        //     }
        // });

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

        if (generalSection) {
            this.formContent.appendChild(generalSection);
        }
        if (displaySection) {
            this.formContent.appendChild(displaySection);
        }
        if (defaultsSection) {
            this.formContent.appendChild(defaultsSection);
        }
        this.formContent.appendChild(widgetsSection);
    }

    /**
     * Handle form field changes
     */
    onFormChange() {
        // Update JSON editor with current config
        this.editorEl.value = JSON.stringify(this.config, null, 2);
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

            if (this.profiles.length > 0) {
                // Show profile selector if profiles are available
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
     * Create a new profile
     */
    async createNewProfile() {
        const name = prompt('Enter name for the new profile:');
        if (!name || !name.trim()) {
            return;
        }

        try {
            this.setStatus('Creating profile...');
            const result = await API.createProfile(name.trim());

            this.showNotification('Profile "' + name.trim() + '" created', 'success');

            // Reload profiles list
            await this.loadProfiles();

            // Switch to the new profile
            if (result.path) {
                this.profileSelect.value = result.path;
                await this.switchProfile();
            }

            this.setStatus('Ready');
        } catch (err) {
            this.showNotification('Failed to create profile: ' + err.message, 'error');
            this.setStatus('Error');
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
     * Show a notification message in the status indicator
     * @param {string} message - The message to show
     * @param {string} type - 'success', 'error', or 'warning'
     */
    showNotification(message, type) {
        // Clear any pending revert timeout
        if (this.notificationTimeout) {
            clearTimeout(this.notificationTimeout);
            this.notificationTimeout = null;
        }

        // Show message in status indicator
        this.statusEl.textContent = message;
        this.statusEl.className = 'status-indicator';
        if (type === 'error') {
            this.statusEl.classList.add('error');
        } else if (type === 'warning') {
            this.statusEl.classList.add('modified');
        } else {
            this.statusEl.classList.add('success');
        }

        // Auto-revert to appropriate status
        const hideDelay = type === 'success' ? 2000 : (type === 'warning' ? 4000 : 6000);
        this.notificationTimeout = setTimeout(() => {
            if (this.isDirty) {
                this.setStatus('Modified');
            } else {
                this.setStatus('Ready');
            }
        }, hideDelay);
    }

    /**
     * Set the status indicator text and style
     * @param {string} text - The status text
     */
    setStatus(text) {
        // Clear any pending notification timeout
        if (this.notificationTimeout) {
            clearTimeout(this.notificationTimeout);
            this.notificationTimeout = null;
        }

        this.statusEl.textContent = text;
        this.statusEl.className = 'status-indicator';
        if (text === 'Modified') {
            this.statusEl.classList.add('modified');
        } else if (text === 'Error') {
            this.statusEl.classList.add('error');
        }
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
