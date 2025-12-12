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
     * Validate configuration without saving
     * @param {Object} config - The configuration to validate
     * @returns {Promise<Object>} Validation result with valid (boolean) and errors (array)
     */
    async validateConfig(config) {
        const response = await fetch('/api/validate', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(config),
        });

        return response.json();
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
};
