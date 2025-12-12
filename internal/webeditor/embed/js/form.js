/**
 * FormBuilder - Creates form fields from schema properties
 */

class FormBuilder {
    constructor(schemaProcessor, onChange) {
        this.schema = schemaProcessor;
        this.onChange = onChange || (() => {});
    }

    /**
     * Create a form section with a title
     * @param {string} title - Section title
     * @param {string} id - Section ID
     * @returns {HTMLElement}
     */
    createSection(title, id) {
        const section = document.createElement('section');
        section.className = 'form-section';
        section.id = id;

        const header = document.createElement('h2');
        header.textContent = title;
        section.appendChild(header);

        return section;
    }

    /**
     * Create a collapsible section
     * @param {string} title - Section title
     * @param {boolean} expanded - Whether initially expanded
     * @returns {HTMLElement}
     */
    createCollapsibleSection(title, expanded = false) {
        const container = document.createElement('div');
        container.className = 'collapsible-section';

        const header = document.createElement('h3');
        header.className = 'collapsible-header' + (expanded ? ' expanded' : '');
        header.textContent = title;
        header.addEventListener('click', () => {
            header.classList.toggle('expanded');
            content.classList.toggle('expanded');
        });

        const content = document.createElement('div');
        content.className = 'collapsible-content' + (expanded ? ' expanded' : '');

        container.appendChild(header);
        container.appendChild(content);

        return { container, content };
    }

    /**
     * Create a form field based on property schema
     * @param {string} name - Property name
     * @param {Object} propSchema - Property schema
     * @param {*} value - Current value
     * @param {Function} onUpdate - Callback when value changes
     * @returns {HTMLElement}
     */
    createField(name, propSchema, value, onUpdate) {
        const type = this.schema.getPropertyType(propSchema);
        const label = this.schema.getLabel(name);
        const description = this.schema.getDescription(propSchema);

        switch (type) {
            case 'string':
                return this.createStringField(name, label, description, propSchema, value, onUpdate);
            case 'number':
            case 'integer':
                return this.createNumberField(name, label, description, propSchema, value, onUpdate);
            case 'boolean':
                return this.createBooleanField(name, label, description, value, onUpdate);
            case 'enum':
                return this.createEnumField(name, label, description, propSchema, value, onUpdate);
            case 'object':
                return this.createObjectField(name, label, description, propSchema, value, onUpdate);
            case 'array':
                return this.createArrayInfo(name, label, description);
            default:
                return this.createReadonlyField(name, label, value);
        }
    }

    /**
     * Create a string input field
     */
    createStringField(name, label, description, propSchema, value, onUpdate) {
        const container = document.createElement('div');
        container.className = 'form-field';

        const labelEl = document.createElement('label');
        labelEl.setAttribute('for', `field-${name}`);
        labelEl.textContent = label;

        const input = document.createElement('input');
        input.type = 'text';
        input.id = `field-${name}`;
        input.name = name;
        input.value = value ?? propSchema.default ?? '';
        if (propSchema.pattern) {
            input.pattern = propSchema.pattern;
        }
        if (propSchema.maxLength) {
            input.maxLength = propSchema.maxLength;
        }

        input.addEventListener('input', () => {
            onUpdate(input.value);
            this.onChange();
        });

        container.appendChild(labelEl);
        container.appendChild(input);

        if (description) {
            const desc = document.createElement('small');
            desc.textContent = description;
            container.appendChild(desc);
        }

        return container;
    }

    /**
     * Create a number input field
     */
    createNumberField(name, label, description, propSchema, value, onUpdate) {
        const container = document.createElement('div');
        container.className = 'form-field';

        const labelEl = document.createElement('label');
        labelEl.setAttribute('for', `field-${name}`);
        labelEl.textContent = label;

        const input = document.createElement('input');
        input.type = 'number';
        input.id = `field-${name}`;
        input.name = name;
        input.value = value ?? propSchema.default ?? '';

        if (propSchema.minimum !== undefined) {
            input.min = propSchema.minimum;
        }
        if (propSchema.maximum !== undefined) {
            input.max = propSchema.maximum;
        }

        // Check if this is a color field (0-255 or -1 for transparent)
        const isColor = name.toLowerCase().includes('color') ||
                        name.toLowerCase().includes('background') ||
                        (propSchema.minimum === -1 && propSchema.maximum === 255);

        if (isColor) {
            input.classList.add('color-input');
        }

        input.addEventListener('input', () => {
            const val = input.value === '' ? undefined : Number(input.value);
            onUpdate(val);
            this.onChange();

            // Update color preview if present
            if (isColor && input.nextElementSibling?.classList.contains('color-preview')) {
                this.updateColorPreview(input.nextElementSibling, val);
            }
        });

        container.appendChild(labelEl);
        container.appendChild(input);

        // Add color preview for color fields
        if (isColor) {
            const preview = document.createElement('span');
            preview.className = 'color-preview';
            this.updateColorPreview(preview, value);
            container.appendChild(preview);
        }

        if (description) {
            const desc = document.createElement('small');
            desc.textContent = description;
            container.appendChild(desc);
        }

        return container;
    }

    /**
     * Update color preview element
     */
    updateColorPreview(preview, value) {
        if (value === -1 || value === undefined) {
            preview.style.background = 'repeating-linear-gradient(45deg, #ccc, #ccc 5px, #fff 5px, #fff 10px)';
            preview.title = 'Transparent';
        } else {
            const gray = Math.max(0, Math.min(255, value ?? 0));
            preview.style.background = `rgb(${gray}, ${gray}, ${gray})`;
            preview.title = `Gray: ${gray}`;
        }
    }

    /**
     * Create a boolean checkbox field
     */
    createBooleanField(name, label, description, value, onUpdate) {
        const container = document.createElement('div');
        container.className = 'form-field checkbox-field';

        const input = document.createElement('input');
        input.type = 'checkbox';
        input.id = `field-${name}`;
        input.name = name;
        input.checked = value ?? false;

        input.addEventListener('change', () => {
            onUpdate(input.checked);
            this.onChange();
        });

        const labelEl = document.createElement('label');
        labelEl.setAttribute('for', `field-${name}`);
        labelEl.textContent = label;

        container.appendChild(input);
        container.appendChild(labelEl);

        if (description) {
            const desc = document.createElement('small');
            desc.textContent = description;
            container.appendChild(desc);
        }

        return container;
    }

    /**
     * Create an enum select field
     */
    createEnumField(name, label, description, propSchema, value, onUpdate) {
        const container = document.createElement('div');
        container.className = 'form-field';

        const labelEl = document.createElement('label');
        labelEl.setAttribute('for', `field-${name}`);
        labelEl.textContent = label;

        const select = document.createElement('select');
        select.id = `field-${name}`;
        select.name = name;

        // Add empty option if field is not required
        const emptyOpt = document.createElement('option');
        emptyOpt.value = '';
        emptyOpt.textContent = '-- Select --';
        select.appendChild(emptyOpt);

        for (const opt of propSchema.enum) {
            const option = document.createElement('option');
            option.value = opt;
            option.textContent = opt;
            if (opt === value) {
                option.selected = true;
            }
            select.appendChild(option);
        }

        select.addEventListener('change', () => {
            onUpdate(select.value || undefined);
            this.onChange();
        });

        container.appendChild(labelEl);
        container.appendChild(select);

        if (description) {
            const desc = document.createElement('small');
            desc.textContent = description;
            container.appendChild(desc);
        }

        return container;
    }

    /**
     * Create nested object fields
     */
    createObjectField(name, label, description, propSchema, value, onUpdate) {
        const { container, content } = this.createCollapsibleSection(label, false);

        if (description) {
            const desc = document.createElement('small');
            desc.textContent = description;
            desc.style.display = 'block';
            desc.style.marginBottom = '0.5rem';
            content.appendChild(desc);
        }

        const grid = document.createElement('div');
        grid.className = 'form-grid';

        const objValue = value || {};
        const properties = propSchema.properties || {};

        for (const [propName, propDef] of Object.entries(properties)) {
            const field = this.createField(
                propName,
                propDef,
                objValue[propName],
                (newVal) => {
                    if (newVal === undefined) {
                        delete objValue[propName];
                    } else {
                        objValue[propName] = newVal;
                    }
                    onUpdate(objValue);
                }
            );
            grid.appendChild(field);
        }

        content.appendChild(grid);
        return container;
    }

    /**
     * Create info for array fields (not fully editable in form view)
     */
    createArrayInfo(name, label, description) {
        const container = document.createElement('div');
        container.className = 'form-field';

        const labelEl = document.createElement('label');
        labelEl.textContent = label;

        const info = document.createElement('small');
        info.textContent = description || 'Array fields are best edited in JSON view.';

        container.appendChild(labelEl);
        container.appendChild(info);

        return container;
    }

    /**
     * Create a readonly display field
     */
    createReadonlyField(name, label, value) {
        const container = document.createElement('div');
        container.className = 'form-field';

        const labelEl = document.createElement('label');
        labelEl.textContent = label;

        const display = document.createElement('code');
        display.textContent = JSON.stringify(value);
        display.style.display = 'block';
        display.style.padding = '0.5rem';
        display.style.background = 'var(--code-background-color)';
        display.style.borderRadius = 'var(--border-radius)';

        container.appendChild(labelEl);
        container.appendChild(display);

        return container;
    }

    /**
     * Render global config section
     * @param {Object} config - Current configuration
     * @param {Function} onUpdate - Callback when config changes
     * @returns {HTMLElement}
     */
    renderGlobalConfig(config, onUpdate) {
        const section = this.createSection('General Settings', 'section-general');
        const grid = document.createElement('div');
        grid.className = 'form-grid';

        const rootProps = this.schema.getRootProperties();

        // Define which properties to show in general section
        // Note: game_name and game_display_name are hidden (advanced settings, edit in JSON)
        const generalProps = ['config_name', 'refresh_rate_ms', 'backend'];

        for (const propName of generalProps) {
            if (!rootProps[propName]) continue;

            const field = this.createField(
                propName,
                rootProps[propName],
                config[propName],
                (newVal) => {
                    if (newVal === undefined) {
                        delete config[propName];
                    } else {
                        config[propName] = newVal;
                    }
                    onUpdate(config);
                }
            );
            grid.appendChild(field);
        }

        section.appendChild(grid);
        return section;
    }

    /**
     * Render display config section
     * Note: This section is hidden as display settings are advanced options.
     * Users can edit display settings in JSON view if needed.
     * @param {Object} config - Current configuration
     * @param {Function} onUpdate - Callback when config changes
     * @returns {HTMLElement|null}
     */
    renderDisplayConfig(config, onUpdate) {
        // Return null to hide this section - display settings are advanced
        // and rarely need to be changed (device-specific dimensions)
        return null;
    }

    /**
     * Render defaults config section
     * Note: This section is hidden as it contains complex dynamic properties.
     * Users can edit defaults in JSON view if needed.
     * @param {Object} config - Current configuration
     * @param {Function} onUpdate - Callback when config changes
     * @returns {HTMLElement|null}
     */
    renderDefaultsConfig(config, onUpdate) {
        // Return null to hide this section - defaults contain dynamic color
        // definitions that are better edited in JSON view
        return null;
    }

    /**
     * Render widgets summary (placeholder for Phase 3)
     * @param {Object} config - Current configuration
     * @returns {HTMLElement}
     */
    renderWidgetsSummary(config) {
        const section = this.createSection('Widgets', 'section-widgets');

        const widgets = config.widgets || [];

        if (widgets.length === 0) {
            const info = document.createElement('p');
            info.textContent = 'No widgets configured. Use JSON view to add widgets.';
            section.appendChild(info);
            return section;
        }

        const list = document.createElement('div');
        list.className = 'widget-list';

        for (let i = 0; i < widgets.length; i++) {
            const widget = widgets[i];
            const item = document.createElement('div');
            item.className = 'widget-item';

            const header = document.createElement('div');
            header.className = 'widget-item-header';

            const title = document.createElement('h4');
            title.textContent = `${i + 1}. ${widget.type || 'Unknown'}`;
            if (widget.position) {
                title.textContent += ` (${widget.position.w}x${widget.position.h})`;
            }

            header.appendChild(title);
            item.appendChild(header);

            // Show basic widget info
            const info = document.createElement('small');
            const props = [];
            if (widget.mode) props.push(`mode: ${widget.mode}`);
            if (widget.position?.x !== undefined) props.push(`x: ${widget.position.x}`);
            if (widget.position?.y !== undefined) props.push(`y: ${widget.position.y}`);
            info.textContent = props.join(', ') || 'No additional properties';
            item.appendChild(info);

            list.appendChild(item);
        }

        const hint = document.createElement('p');
        hint.innerHTML = '<small>Full widget editing coming soon. Use JSON view for detailed editing.</small>';

        section.appendChild(list);
        section.appendChild(hint);
        return section;
    }
}

// Export for use in other modules
window.FormBuilder = FormBuilder;
