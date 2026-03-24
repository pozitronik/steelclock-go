/**
 * API communication layer for SteelClock Configuration Editor
 */

const API = {
    /**
     * Get the JSON schema
     * @returns {Promise<Object>} The JSON schema
     */
    async getSchema() {
        const response = await fetch('/api/schema');
        if (!response.ok) {
            throw new Error(`Failed to load schema: ${response.statusText}`);
        }
        return response.json();
    },

    /**
     * Get the current configuration
     * @returns {Promise<Object>} The current configuration
     */
    async getConfig() {
        const response = await fetch('/api/config');
        if (!response.ok) {
            throw new Error(`Failed to load config: ${response.statusText}`);
        }
        return response.json();
    },

    /**
     * Load configuration from a specific path (without switching active profile)
     * @param {string} path - The profile path to load from
     * @returns {Promise<Object>} The configuration
     */
    async loadConfigByPath(path) {
        const response = await fetch('/api/config/load', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ path }),
        });

        if (!response.ok) {
            const result = await response.json();
            throw new Error(result.error || response.statusText);
        }
        return response.json();
    },

    /**
     * Save the configuration
     * @param {Object} config - The configuration to save
     * @param {string} [path] - Optional path to save to (if not active profile)
     * @returns {Promise<Object>} The save result
     */
    async saveConfig(config, path) {
        const body = path
            ? { path, config }
            : config;

        const response = await fetch('/api/config', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(body),
        });

        const result = await response.json();

        if (result.error) {
            throw new Error(result.error);
        }

        return result;
    },

    /**
     * Get all profiles
     * @returns {Promise<Array>} List of profiles
     */
    async getProfiles() {
        const response = await fetch('/api/profiles');
        if (!response.ok) {
            throw new Error(`Failed to load profiles: ${response.statusText}`);
        }
        return response.json();
    },

    /**
     * Switch to a different profile
     * @param {string} path - The profile path to switch to
     * @returns {Promise<Object>} The switch result
     */
    async switchProfile(path) {
        const response = await fetch('/api/profiles/active', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ path }),
        });

        const result = await response.json();

        if (result.error) {
            throw new Error(result.error);
        }

        return result;
    },

    /**
     * Create a new profile
     * @param {string} name - The name for the new profile
     * @returns {Promise<Object>} The creation result with path
     */
    async createProfile(name) {
        const response = await fetch('/api/profiles', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ name }),
        });

        const result = await response.json();

        if (result.error) {
            throw new Error(result.error);
        }

        return result;
    },

    /**
     * Rename a profile
     * @param {string} path - The current profile path
     * @param {string} newName - The new name for the profile
     * @returns {Promise<Object>} The rename result with new path
     */
    async renameProfile(path, newName) {
        const response = await fetch('/api/profiles/rename', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ path, new_name: newName }),
        });

        const result = await response.json();

        if (result.error) {
            throw new Error(result.error);
        }

        return result;
    },

    /**
     * Get preview availability and configuration
     * @param {string} [deviceId] - Optional device ID for multi-device preview
     * @returns {Promise<Object>} Preview info with available, width, height, target_fps
     */
    async getPreviewInfo(deviceId) {
        const url = deviceId
            ? `/api/preview?device=${encodeURIComponent(deviceId)}`
            : '/api/preview';
        const response = await fetch(url);
        if (!response.ok) {
            throw new Error(`Failed to get preview info: ${response.statusText}`);
        }
        return response.json();
    },

    /**
     * Get list of preview devices
     * @returns {Promise<Object>} Device list with devices array
     */
    async getPreviewDevices() {
        const response = await fetch('/api/preview/devices');
        if (!response.ok) {
            throw new Error(`Failed to get preview devices: ${response.statusText}`);
        }
        return response.json();
    },

    /**
     * Create a WebSocket connection for live preview
     * @param {string} [deviceId] - Optional device ID for multi-device preview
     * @returns {WebSocket} WebSocket connection
     */
    createPreviewWebSocket(deviceId) {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        let url = `${protocol}//${window.location.host}/api/preview/ws`;
        if (deviceId) {
            url += `?device=${encodeURIComponent(deviceId)}`;
        }
        return new WebSocket(url);
    },

    /**
     * Enable or disable preview backend override.
     * When enabled, temporarily switches to preview backend regardless of config.
     * When disabled, restores the original backend.
     * @param {boolean} enable - Whether to enable or disable preview override
     * @returns {Promise<Object>} Result with success and enabled status
     */
    async setPreviewOverride(enable) {
        const response = await fetch('/api/preview/override', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ enable }),
        });

        const result = await response.json();

        if (result.error) {
            throw new Error(result.error);
        }

        return result;
    },
};
