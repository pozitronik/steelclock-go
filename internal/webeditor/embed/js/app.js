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
        this.editingProfilePath = null; // Path of profile being edited (may differ from active)

        // Device state: editor always works with devices[] internally.
        // loadedAsSingleDevice tracks original format for save-time flattening.
        this.loadedAsSingleDevice = false;
        this.selectedDeviceIndex = 0;

        // DOM elements
        this.formContainer = document.getElementById('form-container');
        this.formContent = document.getElementById('form-content');
        this.jsonContainer = document.getElementById('json-container');
        this.editorEl = document.getElementById('config-editor');
        this.saveBtn = document.getElementById('btn-save');
        this.applyBtn = document.getElementById('btn-apply');
        this.reloadBtn = document.getElementById('btn-reload');
        this.viewToggleCheckbox = document.getElementById('view-toggle-checkbox');
        this.statusEl = document.getElementById('status');
        this.profileSelector = document.getElementById('profile-selector');
        this.profileSelect = document.getElementById('profile-select');
        this.renameProfileBtn = document.getElementById('btn-rename-profile');
        this.newProfileBtn = document.getElementById('btn-new-profile');
        this.themeToggle = document.getElementById('toggle-theme');
        this.previewToggleCheckbox = document.getElementById('preview-toggle-checkbox');
    }

    /**
     * Initialize the editor
     */
    async init() {
        // Set up event listeners
        this.saveBtn.addEventListener('click', () => this.save());
        this.applyBtn.addEventListener('click', () => this.apply());
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
        this.renameProfileBtn.addEventListener('click', () => this.renameCurrentProfile());
        this.newProfileBtn.addEventListener('click', () => this.createNewProfile());

        // View toggle (checkbox: unchecked = form, checked = json)
        this.viewToggleCheckbox.addEventListener('change', () => {
            this.switchView(this.viewToggleCheckbox.checked ? 'json' : 'form');
        });
        // Allow clicking JSON label to toggle
        document.querySelector('.view-toggle-label[data-view="json"]').addEventListener('click', () => {
            this.viewToggleCheckbox.checked = !this.viewToggleCheckbox.checked;
            this.switchView(this.viewToggleCheckbox.checked ? 'json' : 'form');
        });

        // Preview toggle
        this.previewToggleCheckbox.addEventListener('change', async () => {
            if (window.previewPanel) {
                if (this.previewToggleCheckbox.checked) {
                    await window.previewPanel.show();
                } else {
                    await window.previewPanel.hide();
                }
            }
        });
        // Click on label toggles preview
        document.querySelector('.preview-toggle-label').addEventListener('click', () => {
            this.previewToggleCheckbox.checked = !this.previewToggleCheckbox.checked;
            this.previewToggleCheckbox.dispatchEvent(new Event('change'));
        });

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
            this.viewToggleCheckbox.disabled = true;
        }
    }

    /**
     * Load configuration from server (active profile)
     */
    async loadConfig() {
        try {
            this.setStatus('Loading...');
            this.config = await API.getConfig();

            // Apply schema defaults to fill in missing values
            if (this.schemaProcessor) {
                this.config = this.schemaProcessor.applyDefaults(this.config);
            }

            // Store original (save-format) and normalize to devices[]
            this.originalConfig = JSON.stringify(this.config, null, 2);
            this.editorEl.value = this.originalConfig;
            this.normalizeConfig();

            this.isDirty = false;
            this.saveBtn.classList.remove('has-changes');

            // Set editing profile to active profile
            const activeProfile = this.profiles.find(p => p.is_active);
            if (activeProfile) {
                this.editingProfilePath = activeProfile.path;
            }

            // Update device selection and preview
            this.onDevicesChanged();

            // Render form if schema is available
            if (this.schemaProcessor && this.currentView === 'form') {
                this.renderForm();
            }

            this.updateApplyButtonState();
            this.showNotification('Configuration loaded', 'success');
            this.setStatus('Ready');
        } catch (err) {
            this.showNotification('Failed to load: ' + err.message, 'error');
            this.setStatus('Error');
        }
    }

    /**
     * Load configuration from a specific path (without switching active profile)
     */
    async loadConfigFromPath(path) {
        this.setStatus('Loading...');
        this.config = await API.loadConfigByPath(path);

        // Apply schema defaults to fill in missing values
        if (this.schemaProcessor) {
            this.config = this.schemaProcessor.applyDefaults(this.config);
        }

        // Store original (save-format) and normalize to devices[]
        this.originalConfig = JSON.stringify(this.config, null, 2);
        this.editorEl.value = this.originalConfig;
        this.normalizeConfig();

        this.isDirty = false;
        this.saveBtn.classList.remove('has-changes');

        // Update device selection and preview
        this.onDevicesChanged();

        // Render form if schema is available
        if (this.schemaProcessor && this.currentView === 'form') {
            this.renderForm();
        }

        this.setStatus('Ready');
    }

    /**
     * Render the form view.
     * Always shows global settings + tabbed device panel.
     */
    renderForm() {
        if (!this.schemaProcessor || !this.config) {
            this.formContent.innerHTML = '<p>Schema or configuration not available.</p>';
            return;
        }

        this.formContent.innerHTML = '';

        // Global settings (always visible)
        const generalSection = this.formBuilder.renderGlobalConfig(this.config, () => this.onFormChange());
        if (generalSection) {
            this.formContent.appendChild(generalSection);
        }

        // Device tabs (always visible — config is always normalized to devices[])
        this.formContent.appendChild(this.buildDeviceTabPanel());
    }

    /**
     * Build the tabbed device panel with tab bar + content for the selected device
     * @returns {HTMLElement}
     */
    buildDeviceTabPanel() {
        const panel = document.createElement('div');
        panel.className = 'device-tab-panel';

        // Tab bar
        const tabBar = document.createElement('div');
        tabBar.className = 'device-tab-bar';

        this.config.devices.forEach((device, index) => {
            const tab = document.createElement('button');
            tab.className = 'device-tab' + (index === this.selectedDeviceIndex ? ' active' : '');

            const nameSpan = document.createElement('span');
            nameSpan.className = 'device-tab-name';
            nameSpan.textContent = device.id || `device_${index}`;
            tab.appendChild(nameSpan);

            if (device.display?.width && device.display?.height) {
                const sizeSpan = document.createElement('span');
                sizeSpan.className = 'device-tab-size';
                sizeSpan.textContent = `${device.display.width}x${device.display.height}`;
                tab.appendChild(sizeSpan);
            }

            if (this.config.devices.length > 1) {
                const removeBtn = document.createElement('span');
                removeBtn.className = 'remove-device';
                removeBtn.textContent = 'x';
                removeBtn.title = 'Remove device';
                removeBtn.addEventListener('click', (e) => {
                    e.stopPropagation();
                    this.removeDevice(index);
                });
                tab.appendChild(removeBtn);
            }

            tab.addEventListener('click', () => this.selectDevice(index));
            tabBar.appendChild(tab);
        });

        // Add device button (as a tab-styled button)
        const addTab = document.createElement('button');
        addTab.className = 'device-tab-add';
        addTab.textContent = '+';
        addTab.title = 'Add device';
        addTab.addEventListener('click', () => this.addDevice());
        tabBar.appendChild(addTab);

        panel.appendChild(tabBar);

        // Tab content for selected device
        const device = this.config.devices[this.selectedDeviceIndex];
        if (device) {
            const content = document.createElement('div');
            content.className = 'device-tab-content';

            const deviceSection = this.formBuilder.renderDeviceConfig(device, () => {
                this.onFormChange();
                // Update tab labels when device ID or display changes
                this.refreshDeviceTabLabels(tabBar);
            });
            if (deviceSection) {
                content.appendChild(deviceSection);
            }

            const widgetsSection = this.widgetEditor
                ? this.widgetEditor.renderWidgetsSection(device, () => this.onFormChange())
                : this.formBuilder.renderWidgetsSummary(device);
            content.appendChild(widgetsSection);

            panel.appendChild(content);
        }

        return panel;
    }

    /**
     * Update tab labels without full re-render (for live ID/size updates)
     * @param {HTMLElement} tabBar - The tab bar element
     */
    refreshDeviceTabLabels(tabBar) {
        const tabs = tabBar.querySelectorAll('.device-tab');
        tabs.forEach((tab, index) => {
            if (index >= this.config.devices.length) return;
            const device = this.config.devices[index];

            const nameSpan = tab.querySelector('.device-tab-name');
            if (nameSpan) {
                nameSpan.textContent = device.id || `device_${index}`;
            }

            const sizeSpan = tab.querySelector('.device-tab-size');
            if (sizeSpan && device.display) {
                sizeSpan.textContent = `${device.display.width || '?'}x${device.display.height || '?'}`;
            }
        });
    }

    /**
     * Handle form field changes
     */
    onFormChange() {
        // Update JSON editor with save-format config
        this.editorEl.value = JSON.stringify(this.configForSave(), null, 2);
    }

    /**
     * Switch between form and JSON views
     * @param {string} view - 'form' or 'json'
     */
    switchView(view) {
        if (view === this.currentView) return;

        // Sync data between views before switching
        if (this.currentView === 'json') {
            // Switching from JSON to Form — parse, detect format, normalize
            try {
                this.config = JSON.parse(this.editorEl.value);
                this.normalizeConfig();
                this.onDevicesChanged();
            } catch (_err) {
                this.showNotification('Invalid JSON - please fix before switching to Form view', 'error');
                return;
            }
        } else {
            // Switching from Form to JSON — show save format
            this.editorEl.value = JSON.stringify(this.configForSave(), null, 2);
        }

        this.currentView = view;

        // Update UI
        const jsonLabel = document.querySelector('.view-toggle-label[data-view="json"]');
        if (view === 'form') {
            this.formContainer.style.display = 'block';
            this.jsonContainer.style.display = 'none';
            this.viewToggleCheckbox.checked = false;
            jsonLabel.classList.remove('active');

            // Re-render form with current config
            this.renderForm();
        } else {
            this.formContainer.style.display = 'none';
            this.jsonContainer.style.display = 'block';
            this.viewToggleCheckbox.checked = true;
            jsonLabel.classList.add('active');
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
     * Save configuration to file (without applying)
     */
    async save() {
        try {
            this.setStatus('Saving...');

            // Get config in save format
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
                configToSave = this.configForSave();
            }

            // Save to the editing profile path
            await API.saveConfig(configToSave, this.editingProfilePath);

            this.originalConfig = JSON.stringify(configToSave, null, 2);
            this.editorEl.value = this.originalConfig;
            this.isDirty = false;
            this.saveBtn.classList.remove('has-changes');

            this.showNotification('Configuration saved', 'success');
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
        } catch (_err) {
            // Profile info is optional, don't show error
            this.profileSelector.style.display = 'none';
        }
    }

    /**
     * Switch to a different profile (load into editor).
     * When preview is visible, auto-applies so the preview matches.
     */
    async switchProfile() {
        const selectedPath = this.profileSelect.value;
        if (!selectedPath) return;

        // Check for unsaved changes
        if (this.isDirty) {
            if (!confirm('You have unsaved changes. Switch profile anyway?')) {
                // Reset select to currently editing profile
                this.profileSelect.value = this.editingProfilePath;
                return;
            }
        }

        const previewVisible = window.previewPanel && window.previewPanel.isVisible;

        try {
            this.setStatus('Loading profile...');
            this.editingProfilePath = selectedPath;

            // Enter transition BEFORE loading config — prevents detectMultiDevice →
            // syncPreviewDevice → setDeviceId from triggering init() on the old server
            if (previewVisible) {
                window.previewPanel.beginTransition();
            }

            // Load the config file directly (without switching active profile)
            await this.loadConfigFromPath(selectedPath);

            this.updateApplyButtonState();

            // When preview is visible, auto-apply so preview matches the editor
            if (previewVisible) {
                await this.apply();
            } else {
                this.showNotification('Profile loaded into editor', 'success');
            }
        } catch (err) {
            this.showNotification('Failed to load profile: ' + err.message, 'error');
            // Reset select to currently editing profile
            this.profileSelect.value = this.editingProfilePath;
            // End transition on error
            if (previewVisible && window.previewPanel.transitioning) {
                window.previewPanel.transitioning = false;
                window.previewPanel.showUnavailable();
            }
        }
    }

    /**
     * Apply the current profile (save if dirty, then switch/reload)
     */
    async apply() {
        if (!this.editingProfilePath) return;

        try {
            this.setStatus('Applying...');

            // Enter preview transition early to avoid error flashes during restart
            if (window.previewPanel && window.previewPanel.isVisible) {
                window.previewPanel.beginTransition();
            }

            // Save first if there are unsaved changes
            if (this.isDirty) {
                let configToSave;
                if (this.currentView === 'json') {
                    configToSave = JSON.parse(this.editorEl.value);
                } else {
                    configToSave = this.configForSave();
                }
                await API.saveConfig(configToSave, this.editingProfilePath);

                this.originalConfig = JSON.stringify(configToSave, null, 2);
                this.editorEl.value = this.originalConfig;
                this.isDirty = false;
                this.saveBtn.classList.remove('has-changes');
            }

            // Switch profile (which triggers reload) or just trigger reload
            await API.switchProfile(this.editingProfilePath);

            // Update profile selection state
            this.profiles.forEach(p => {
                p.is_active = (p.path === this.editingProfilePath);
            });

            this.updateApplyButtonState();

            // Re-initialize preview if visible (new devices may now be available)
            await this.reinitPreviewAfterApply();

            this.showNotification('Configuration applied', 'success');
            this.setStatus('Ready');
        } catch (err) {
            this.showNotification('Failed to apply: ' + err.message, 'error');
            this.setStatus('Error');
        }
    }

    /**
     * Update Apply button state based on whether editing profile differs from active
     */
    updateApplyButtonState() {
        const activeProfile = this.profiles.find(p => p.is_active);
        const needsApply = !activeProfile || activeProfile.path !== this.editingProfilePath;

        if (needsApply) {
            this.applyBtn.classList.add('contrast');
            this.applyBtn.title = 'Switch to this profile and apply';
        } else {
            this.applyBtn.classList.remove('contrast');
            this.applyBtn.title = 'Reload configuration';
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
     * Rename the current profile
     */
    async renameCurrentProfile() {
        const activeProfile = this.profiles.find(p => p.is_active);
        if (!activeProfile) {
            this.showNotification('No active profile to rename', 'error');
            return;
        }

        const newName = prompt('Enter new name for the profile:', activeProfile.name);
        if (!newName || !newName.trim() || newName.trim() === activeProfile.name) {
            return;
        }

        try {
            this.setStatus('Renaming profile...');
            await API.renameProfile(activeProfile.path, newName.trim());

            this.showNotification('Profile renamed to "' + newName.trim() + '"', 'success');

            // Reload profiles list to get updated name and order
            await this.loadProfiles();

            this.setStatus('Ready');
        } catch (err) {
            this.showNotification('Failed to rename profile: ' + err.message, 'error');
            this.setStatus('Error');
        }
    }

    // ========== Device management ==========

    /**
     * Normalize config to always use devices[] array internally.
     * Single-device configs are wrapped into devices[0].
     * Must be called after loading/parsing config.
     */
    normalizeConfig() {
        if (Array.isArray(this.config.devices) && this.config.devices.length > 0) {
            this.loadedAsSingleDevice = false;
            return;
        }

        // Wrap single-device fields into devices[0]
        this.loadedAsSingleDevice = true;
        const device = {
            id: 'default',
            display: this.config.display || { width: 128, height: 40, background: 0 },
            widgets: this.config.widgets || [],
        };
        if (this.config.backend) device.backend = this.config.backend;
        if (this.config.direct_driver) device.direct_driver = this.config.direct_driver;
        if (this.config.webclient) device.webclient = this.config.webclient;

        this.config.devices = [device];
        delete this.config.display;
        delete this.config.widgets;
        delete this.config.backend;
        delete this.config.direct_driver;
        delete this.config.webclient;
    }

    /**
     * Produce the config in save format.
     * If originally single-device and still has exactly one device, flattens back.
     * Otherwise keeps devices[] array.
     * @returns {Object} Config object ready for save/JSON display
     */
    configForSave() {
        const result = JSON.parse(JSON.stringify(this.config));

        if (this.loadedAsSingleDevice && result.devices && result.devices.length === 1) {
            // Flatten back to single-device format
            const device = result.devices[0];
            result.display = device.display;
            result.widgets = device.widgets;
            if (device.backend) result.backend = device.backend;
            if (device.direct_driver) result.direct_driver = device.direct_driver;
            if (device.webclient) result.webclient = device.webclient;
            delete result.devices;
        }

        return result;
    }

    /**
     * Called after devices[] changes (load, add, remove).
     * Clamps selected index and syncs preview.
     */
    onDevicesChanged() {
        if (this.selectedDeviceIndex >= this.config.devices.length) {
            this.selectedDeviceIndex = 0;
        }
        this.syncPreviewDevice();
    }

    /**
     * Select a device by index
     * @param {number} index - Device index
     */
    selectDevice(index) {
        if (index === this.selectedDeviceIndex) return;
        this.selectedDeviceIndex = index;

        if (this.currentView === 'form' && this.schemaProcessor) {
            this.renderForm();
        }

        this.syncPreviewDevice();
    }

    /**
     * Add a new device to the config
     */
    addDevice() {
        const newIndex = this.config.devices.length;
        this.config.devices.push({
            id: `device_${newIndex}`,
            display: { width: 128, height: 40, background: 0 },
            widgets: [],
        });

        // Once we have multiple devices, always save as devices[] format
        if (this.config.devices.length > 1) {
            this.loadedAsSingleDevice = false;
        }

        this.markDirty();
        this.selectedDeviceIndex = this.config.devices.length - 1;
        this.onDevicesChanged();

        if (this.currentView === 'form' && this.schemaProcessor) {
            this.renderForm();
        }
        this.onFormChange();
        this.showNotification('Device added', 'success');
    }

    /**
     * Remove a device by index
     * @param {number} index - Device index to remove
     */
    removeDevice(index) {
        if (this.config.devices.length <= 1) return;

        const deviceId = this.config.devices[index].id || `device_${index}`;
        if (!confirm(`Remove device "${deviceId}"?`)) return;

        this.config.devices.splice(index, 1);

        // Back to single device — save in original format if it was single-device
        if (this.config.devices.length === 1 && this.loadedAsSingleDevice) {
            // loadedAsSingleDevice stays true — will flatten on save
        }

        this.markDirty();
        this.onDevicesChanged();

        if (this.currentView === 'form' && this.schemaProcessor) {
            this.renderForm();
        }
        this.onFormChange();
        this.showNotification('Device removed', 'success');
    }

    /**
     * Re-initialize preview after config apply / profile switch.
     * Enters transition state (suppresses errors), re-enables override,
     * then reconnects with retries.
     */
    async reinitPreviewAfterApply() {
        const panel = window.previewPanel;
        if (!panel || !panel.isVisible) return;

        // Enter transition — shows "Reloading..." and suppresses error flashes
        panel.beginTransition();

        // Re-enable preview override for the (possibly new) config
        try {
            await API.setPreviewOverride(true);
        } catch (_err) {
            // Ignore — backends may already be webclient
        }

        // Sync device ID to match the currently selected tab
        this.syncPreviewDevice();

        // Reconnect with retries (handles variable backend startup time)
        await panel.reconnect();
    }

    /**
     * Sync the preview panel to show the currently selected device
     */
    syncPreviewDevice() {
        if (!window.previewPanel) return;

        if (this.isMultiDevice && this.config.devices[this.selectedDeviceIndex]) {
            const device = this.config.devices[this.selectedDeviceIndex];
            window.previewPanel.setDeviceId(device.id || null);
        } else {
            window.previewPanel.setDeviceId(null);
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
