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
     * Save the configuration
     * @param {Object} config - The configuration to save
     * @returns {Promise<Object>} The save result
     */
    async saveConfig(config) {
        const response = await fetch('/api/config', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(config),
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
};
