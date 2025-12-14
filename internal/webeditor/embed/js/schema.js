/**
 * SchemaProcessor - Resolves $ref and extracts widget-specific schemas
 */

/**
 * @typedef {Object} JSONSchemaProperty
 * @property {string} [$ref] - Reference to another definition
 * @property {string} [type] - Property type
 * @property {string} [description] - Property description
 * @property {*} [default] - Default value
 * @property {number} [minimum] - Minimum value
 * @property {number} [maximum] - Maximum value
 * @property {number} [minLength] - Minimum string length
 * @property {number} [maxLength] - Maximum string length
 * @property {string} [pattern] - Regex pattern for strings
 * @property {Array<string|number>} [enum] - Enumeration of allowed values
 * @property {string} [const] - Constant value
 * @property {Object<string, JSONSchemaProperty>} [properties] - Nested properties for objects
 * @property {JSONSchemaProperty} [items] - Schema for array items
 * @property {Array<string>} [required] - Required property names
 * @property {Array<JSONSchemaProperty|JSONSchemaConditional>} [allOf] - All of these schemas must match
 */

/**
 * @typedef {Object} JSONSchemaConditional
 * @property {JSONSchemaProperty} [if] - Condition schema
 * @property {JSONSchemaProperty} [then] - Schema if condition matches
 * @property {JSONSchemaProperty} [else] - Schema if condition doesn't match
 */

/**
 * @typedef {Object} JSONSchema
 * @property {Object<string, JSONSchemaProperty>} [properties] - Root properties
 * @property {Object<string, JSONSchemaProperty>} [definitions] - Schema definitions
 * @property {Array<string>} [required] - Required properties
 */

class SchemaProcessor {
    /**
     * @param {JSONSchema} schema - The JSON Schema to process
     */
    constructor(schema) {
        /** @type {JSONSchema} */
        this.schema = schema;
        /** @type {Object<string, JSONSchemaProperty>} */
        this.definitions = /** @type {Object<string, JSONSchemaProperty>} */ (schema.definitions || {});
        /** @type {Object<string, {base: Object, specific: Object, required: string[]}>} */
        this.widgetSchemas = {};
        /** @type {Map<string, JSONSchemaProperty>} */
        this.processedRefs = new Map();
    }

    /**
     * Initialize the processor by extracting widget schemas
     */
    init() {
        this.extractWidgetSchemas();
    }

    /**
     * Resolve a $ref reference to its definition
     * @param {string} ref - The $ref string (e.g., "#/definitions/position")
     * @returns {Object|null} The resolved definition
     */
    resolveRef(ref) {
        if (!ref || !ref.startsWith('#/definitions/')) {
            return null;
        }

        // Check cache
        if (this.processedRefs.has(ref)) {
            return this.processedRefs.get(ref);
        }

        const defName = ref.replace('#/definitions/', '');
        const definition = this.definitions[defName];

        if (!definition) {
            return null;
        }

        // Deep clone to avoid mutation
        const resolved = JSON.parse(JSON.stringify(definition));

        // Recursively resolve nested $refs
        this.resolveNestedRefs(resolved);

        this.processedRefs.set(ref, resolved);
        return resolved;
    }

    /**
     * Recursively resolve $ref and allOf in an object
     * @param {JSONSchemaProperty|JSONSchemaProperty[]} obj - The object to process
     */
    resolveNestedRefs(obj) {
        if (!obj || typeof obj !== 'object') {
            return;
        }

        if (Array.isArray(obj)) {
            obj.forEach(item => this.resolveNestedRefs(item));
            return;
        }

        // If this object has a $ref, merge the resolved definition
        if (obj.$ref) {
            const resolved = this.resolveRef(obj.$ref);
            if (resolved) {
                delete obj.$ref;
                Object.assign(obj, resolved);
            }
        }

        // Handle allOf by merging all schemas
        if (Array.isArray(obj.allOf)) {
            this.mergeAllOf(obj);
        }

        // Recurse into nested objects
        for (const key of Object.keys(obj)) {
            if (typeof obj[key] === 'object') {
                this.resolveNestedRefs(obj[key]);
            }
        }
    }

    /**
     * Merge allOf array into a single schema
     * @param {JSONSchemaProperty} obj - Object containing allOf array
     */
    mergeAllOf(obj) {
        if (!Array.isArray(obj.allOf)) {
            return;
        }

        const allOfItems = obj.allOf;
        delete obj.allOf;

        for (const item of allOfItems) {
            // First resolve any $ref in the item
            if (item.$ref) {
                const resolved = this.resolveRef(item.$ref);
                if (resolved) {
                    delete item.$ref;
                    Object.assign(item, resolved);
                }
            }

            // Recursively handle nested allOf
            if (Array.isArray(item.allOf)) {
                this.mergeAllOf(item);
            }

            // Merge properties
            if (item.properties) {
                obj.properties = /** @type {Object<string, JSONSchemaProperty>} */ (obj.properties || {});
                for (const [key, value] of Object.entries(item.properties)) {
                    // If property already exists, merge them
                    if (obj.properties[key]) {
                        Object.assign(obj.properties[key], value);
                    } else {
                        obj.properties[key] = value;
                    }
                }
            }

            // Merge required arrays
            if (Array.isArray(item.required)) {
                obj.required = /** @type {Array<string>} */ (obj.required || []);
                for (const req of item.required) {
                    if (!obj.required.includes(req)) {
                        obj.required.push(req);
                    }
                }
            }

            // Merge type (prefer explicit type)
            if (item.type && !obj.type) {
                obj.type = item.type;
            }

            // Merge description (prefer first non-empty)
            if (item.description && !obj.description) {
                obj.description = item.description;
            }

            // Merge other scalar properties
            for (const key of ['default', 'minimum', 'maximum', 'minLength', 'maxLength', 'pattern']) {
                if (item[key] !== undefined && obj[key] === undefined) {
                    obj[key] = item[key];
                }
            }
        }
    }

    /**
     * Extract widget-specific schemas from allOf conditionals
     */
    extractWidgetSchemas() {
        const widgetDef = this.definitions.widget;
        if (!widgetDef) {
            console.warn('No widget definition found in schema');
            return;
        }

        // Get base properties that apply to all widgets
        const baseProps = widgetDef.properties || {};
        const resolvedBaseProps = JSON.parse(JSON.stringify(baseProps));
        this.resolveNestedRefs(resolvedBaseProps);

        // Extract widget-specific properties from allOf conditionals
        const conditionals = widgetDef.allOf || [];

        for (const cond of conditionals) {
            if (!cond.if || !cond.then) {
                continue;
            }

            // Extract widget type from if condition
            const typeConst = cond.if?.properties?.type?.const;
            if (!typeConst) {
                continue;
            }

            // Get widget-specific properties
            const specificProps = cond.then.properties || {};
            const resolvedSpecificProps = JSON.parse(JSON.stringify(specificProps));
            this.resolveNestedRefs(resolvedSpecificProps);

            this.widgetSchemas[typeConst] = {
                base: resolvedBaseProps,
                specific: resolvedSpecificProps,
                required: cond.then.required || []
            };
        }
    }

    /**
     * Get all available widget types
     * @returns {string[]} Array of widget type names
     */
    getWidgetTypes() {
        return Object.keys(this.widgetSchemas).sort();
    }

    /**
     * Get merged properties for a widget type
     * @param {string} widgetType - The widget type
     * @returns {Object} Merged properties object
     */
    getWidgetProperties(widgetType) {
        const schema = this.widgetSchemas[widgetType];
        if (!schema) {
            // Return base widget properties if type not found
            const widgetDef = this.definitions.widget;
            return widgetDef?.properties || {};
        }

        return {
            ...schema.base,
            ...schema.specific
        };
    }

    /**
     * Get the root schema properties (global config)
     * @returns {Object} Root properties
     */
    getRootProperties() {
        const props = this.schema.properties || {};
        const resolved = JSON.parse(JSON.stringify(props));
        this.resolveNestedRefs(resolved);
        return resolved;
    }

    /**
     * Check if a property is an enum
     * @param {Object} propSchema - The property schema
     * @returns {boolean}
     */
    isEnum(propSchema) {
        return Array.isArray(propSchema?.enum);
    }

    /**
     * Get the type of property
     * @param {Object} propSchema - The property schema
     * @returns {string} The type (string, number, boolean, object, array, enum)
     */
    getPropertyType(propSchema) {
        if (!propSchema) {
            return 'unknown';
        }

        if (this.isEnum(propSchema)) {
            return 'enum';
        }

        if (propSchema.type) {
            return propSchema.type;
        }

        // Infer type from other properties
        if (propSchema.properties) {
            return 'object';
        }
        if (propSchema.items) {
            return 'array';
        }

        return 'unknown';
    }

    /**
     * Get human-readable label from property name
     * @param {string} name - Property name (e.g., "refresh_rate_ms")
     * @returns {string} Human-readable label
     */
    getLabel(name) {
        return name
            .replace(/_/g, ' ')
            .replace(/\b\w/g, c => c.toUpperCase());
    }

    /**
     * Get description for a property
     * @param {Object} propSchema - The property schema
     * @returns {string} Description or empty string
     */
    getDescription(propSchema) {
        return propSchema?.description || '';
    }

    /**
     * Apply schema defaults to a config object
     * @param {Object} config - The config object to fill with defaults
     * @returns {Object} The config with defaults applied
     */
    applyDefaults(config) {
        if (!config) {
            config = {};
        }

        // Apply root-level defaults
        const rootProps = this.getRootProperties();
        this.applyDefaultsToObject(config, rootProps);

        // Apply defaults to widgets based on their type
        if (Array.isArray(config.widgets)) {
            for (const widget of config.widgets) {
                if (widget.type) {
                    const widgetProps = this.getWidgetProperties(widget.type);
                    this.applyDefaultsToObject(widget, widgetProps);
                }
            }
        }

        return config;
    }

    /**
     * Apply defaults from property schemas to an object
     * @param {Object} obj - The object to fill with defaults
     * @param {Object<string, JSONSchemaProperty>} properties - Schema properties object
     */
    applyDefaultsToObject(obj, properties) {
        if (!properties || typeof properties !== 'object') {
            return;
        }

        for (const [key, propSchema] of Object.entries(properties)) {
            // Skip widgets array - handled separately
            if (key === 'widgets') {
                continue;
            }

            // Handle nested objects
            if (propSchema.type === 'object' && propSchema.properties) {
                if (obj[key] === undefined) {
                    obj[key] = {};
                }
                if (typeof obj[key] === 'object' && obj[key] !== null) {
                    this.applyDefaultsToObject(obj[key], propSchema.properties);
                }
            } else if (obj[key] === undefined && propSchema.default !== undefined) {
                // Apply default value for missing fields
                obj[key] = JSON.parse(JSON.stringify(propSchema.default));
            }
        }
    }
}

// Export for use in other modules
window.SchemaProcessor = SchemaProcessor;
