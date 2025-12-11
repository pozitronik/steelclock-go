/**
 * WidgetEditor - Handles widget-specific form editing
 */

class WidgetEditor {
    constructor(schemaProcessor, formBuilder, onChange) {
        this.schema = schemaProcessor;
        this.formBuilder = formBuilder;
        this.onChange = onChange || (() => {});
    }

    /**
     * Render the complete widgets section with list and controls
     * @param {Object} config - Full configuration object
     * @param {Function} onUpdate - Callback when widgets change
     * @returns {HTMLElement}
     */
    renderWidgetsSection(config, onUpdate) {
        const section = document.createElement('section');
        section.className = 'form-section';
        section.id = 'section-widgets';

        const header = document.createElement('div');
        header.className = 'widgets-header';

        const title = document.createElement('h2');
        title.textContent = 'Widgets';

        const addBtn = document.createElement('button');
        addBtn.className = 'outline';
        addBtn.textContent = '+ Add Widget';
        addBtn.addEventListener('click', () => {
            this.addWidget(config, listContainer, onUpdate);
        });

        header.appendChild(title);
        header.appendChild(addBtn);
        section.appendChild(header);

        const listContainer = document.createElement('div');
        listContainer.className = 'widget-list';
        listContainer.id = 'widget-list';

        config.widgets = config.widgets || [];

        this.renderWidgetList(config, listContainer, onUpdate);

        section.appendChild(listContainer);
        return section;
    }

    /**
     * Render the list of widgets
     */
    renderWidgetList(config, container, onUpdate) {
        container.innerHTML = '';

        if (config.widgets.length === 0) {
            const empty = document.createElement('p');
            empty.className = 'empty-message';
            empty.textContent = 'No widgets configured. Click "+ Add Widget" to add one.';
            container.appendChild(empty);
            return;
        }

        config.widgets.forEach((widget, index) => {
            const item = this.renderWidgetItem(config, widget, index, container, onUpdate);
            container.appendChild(item);
        });
    }

    /**
     * Render a single widget item
     */
    renderWidgetItem(config, widget, index, listContainer, onUpdate) {
        const item = document.createElement('div');
        item.className = 'widget-item';
        item.dataset.index = index;

        // Header with title and actions
        const header = document.createElement('div');
        header.className = 'widget-item-header';

        const titleContainer = document.createElement('div');
        titleContainer.className = 'widget-title-container';

        const dragHandle = document.createElement('span');
        dragHandle.className = 'drag-handle';
        dragHandle.textContent = '≡';
        dragHandle.title = 'Drag to reorder';

        const title = document.createElement('h4');
        title.textContent = this.getWidgetTitle(widget, index);

        titleContainer.appendChild(dragHandle);
        titleContainer.appendChild(title);

        const actions = document.createElement('div');
        actions.className = 'widget-actions';

        // Move up button
        const upBtn = document.createElement('button');
        upBtn.className = 'outline secondary';
        upBtn.textContent = '↑';
        upBtn.title = 'Move up';
        upBtn.disabled = index === 0;
        upBtn.addEventListener('click', () => {
            this.moveWidget(config, index, -1, listContainer, onUpdate);
        });

        // Move down button
        const downBtn = document.createElement('button');
        downBtn.className = 'outline secondary';
        downBtn.textContent = '↓';
        downBtn.title = 'Move down';
        downBtn.disabled = index === config.widgets.length - 1;
        downBtn.addEventListener('click', () => {
            this.moveWidget(config, index, 1, listContainer, onUpdate);
        });

        // Duplicate button
        const dupBtn = document.createElement('button');
        dupBtn.className = 'outline secondary';
        dupBtn.textContent = '⧉';
        dupBtn.title = 'Duplicate';
        dupBtn.addEventListener('click', () => {
            this.duplicateWidget(config, index, listContainer, onUpdate);
        });

        // Delete button
        const delBtn = document.createElement('button');
        delBtn.className = 'outline secondary';
        delBtn.textContent = '✕';
        delBtn.title = 'Delete';
        delBtn.addEventListener('click', () => {
            if (confirm(`Delete widget "${widget.type || 'Unknown'}"?`)) {
                this.removeWidget(config, index, listContainer, onUpdate);
            }
        });

        actions.appendChild(upBtn);
        actions.appendChild(downBtn);
        actions.appendChild(dupBtn);
        actions.appendChild(delBtn);

        header.appendChild(titleContainer);
        header.appendChild(actions);

        // Collapsible content
        const toggleBtn = document.createElement('button');
        toggleBtn.className = 'widget-toggle outline secondary';
        toggleBtn.textContent = '▼';
        toggleBtn.title = 'Expand/Collapse';

        const content = document.createElement('div');
        content.className = 'widget-content';
        content.style.display = 'none';

        toggleBtn.addEventListener('click', () => {
            const isExpanded = content.style.display !== 'none';
            content.style.display = isExpanded ? 'none' : 'block';
            toggleBtn.textContent = isExpanded ? '▼' : '▲';
        });

        actions.insertBefore(toggleBtn, upBtn);

        // Render widget fields
        this.renderWidgetFields(widget, content, () => {
            title.textContent = this.getWidgetTitle(widget, index);
            onUpdate(config);
            this.onChange();
        });

        item.appendChild(header);
        item.appendChild(content);

        return item;
    }

    /**
     * Get display title for a widget
     */
    getWidgetTitle(widget, index) {
        let title = `${index + 1}. ${widget.type || 'New Widget'}`;
        if (widget.position?.w && widget.position?.h) {
            title += ` (${widget.position.w}×${widget.position.h})`;
        }
        if (widget.mode) {
            title += ` [${widget.mode}]`;
        }
        return title;
    }

    /**
     * Render fields for a widget
     */
    renderWidgetFields(widget, container, onUpdate) {
        container.innerHTML = '';

        // Type selector (always shown first)
        const typeSection = document.createElement('div');
        typeSection.className = 'widget-type-section';

        const typeField = this.createTypeSelector(widget, () => {
            // Re-render fields when type changes
            this.renderWidgetFields(widget, container, onUpdate);
            onUpdate();
        });
        typeSection.appendChild(typeField);
        container.appendChild(typeSection);

        if (!widget.type) {
            const hint = document.createElement('p');
            hint.className = 'hint';
            hint.textContent = 'Select a widget type to see available options.';
            container.appendChild(hint);
            return;
        }

        // Get properties for this widget type
        const properties = this.schema.getWidgetProperties(widget.type);

        // Group properties by category
        const groups = this.groupProperties(properties, widget.type);

        // Render each group
        for (const [groupName, props] of Object.entries(groups)) {
            if (Object.keys(props).length === 0) continue;

            const group = this.renderPropertyGroup(groupName, props, widget, onUpdate);
            container.appendChild(group);
        }
    }

    /**
     * Create widget type selector
     */
    createTypeSelector(widget, onTypeChange) {
        const container = document.createElement('div');
        container.className = 'form-field';

        const label = document.createElement('label');
        label.textContent = 'Widget Type';
        label.setAttribute('for', 'widget-type-select');

        const select = document.createElement('select');
        select.id = 'widget-type-select';

        const emptyOpt = document.createElement('option');
        emptyOpt.value = '';
        emptyOpt.textContent = '-- Select Type --';
        select.appendChild(emptyOpt);

        const types = this.schema.getWidgetTypes();
        for (const type of types) {
            const opt = document.createElement('option');
            opt.value = type;
            opt.textContent = type;
            if (type === widget.type) {
                opt.selected = true;
            }
            select.appendChild(opt);
        }

        select.addEventListener('change', () => {
            widget.type = select.value || undefined;
            onTypeChange();
        });

        container.appendChild(label);
        container.appendChild(select);

        return container;
    }

    /**
     * Group properties by category for better organization
     */
    groupProperties(properties, widgetType) {
        const groups = {
            'Position': {},
            'Style': {},
            'Display': {},
            'Widget Settings': {},
            'Advanced': {}
        };

        for (const [name, schema] of Object.entries(properties)) {
            // Skip type (already handled)
            if (name === 'type') continue;

            // Categorize by property name
            if (name === 'position') {
                groups['Position'][name] = schema;
            } else if (name === 'style') {
                groups['Style'][name] = schema;
            } else if (name === 'mode' || name === 'update_interval') {
                groups['Display'][name] = schema;
            } else if (name === 'auto_hide' || name === 'direct_driver') {
                groups['Advanced'][name] = schema;
            } else {
                groups['Widget Settings'][name] = schema;
            }
        }

        return groups;
    }

    /**
     * Render a group of properties
     */
    renderPropertyGroup(groupName, properties, widget, onUpdate) {
        const { container, content } = this.formBuilder.createCollapsibleSection(
            groupName,
            groupName === 'Position' || groupName === 'Display'
        );

        const grid = document.createElement('div');
        grid.className = 'form-grid';

        for (const [propName, propSchema] of Object.entries(properties)) {
            // Handle nested objects like position and style
            if (propSchema.properties) {
                widget[propName] = widget[propName] || {};
                const nestedFields = this.renderNestedObject(propName, propSchema, widget[propName], onUpdate);
                grid.appendChild(nestedFields);
            } else {
                const field = this.formBuilder.createField(
                    propName,
                    propSchema,
                    widget[propName],
                    (newVal) => {
                        if (newVal === undefined) {
                            delete widget[propName];
                        } else {
                            widget[propName] = newVal;
                        }
                        onUpdate();
                    }
                );
                grid.appendChild(field);
            }
        }

        content.appendChild(grid);
        return container;
    }

    /**
     * Render nested object properties inline
     */
    renderNestedObject(name, schema, obj, onUpdate) {
        const container = document.createElement('div');
        container.className = 'nested-object';
        container.style.gridColumn = '1 / -1';

        const title = document.createElement('h4');
        title.textContent = this.schema.getLabel(name);
        title.style.marginBottom = '0.5rem';
        container.appendChild(title);

        const grid = document.createElement('div');
        grid.className = 'form-grid';

        for (const [propName, propSchema] of Object.entries(schema.properties || {})) {
            const field = this.formBuilder.createField(
                propName,
                propSchema,
                obj[propName],
                (newVal) => {
                    if (newVal === undefined) {
                        delete obj[propName];
                    } else {
                        obj[propName] = newVal;
                    }
                    onUpdate();
                }
            );
            grid.appendChild(field);
        }

        container.appendChild(grid);
        return container;
    }

    /**
     * Add a new widget
     */
    addWidget(config, listContainer, onUpdate) {
        const newWidget = {
            type: '',
            position: { x: 0, y: 0, w: 128, h: 40 }
        };
        config.widgets.push(newWidget);
        this.renderWidgetList(config, listContainer, onUpdate);
        onUpdate(config);
        this.onChange();

        // Expand the new widget
        setTimeout(() => {
            const items = listContainer.querySelectorAll('.widget-item');
            const lastItem = items[items.length - 1];
            if (lastItem) {
                const toggle = lastItem.querySelector('.widget-toggle');
                if (toggle) toggle.click();
            }
        }, 0);
    }

    /**
     * Remove a widget
     */
    removeWidget(config, index, listContainer, onUpdate) {
        config.widgets.splice(index, 1);
        this.renderWidgetList(config, listContainer, onUpdate);
        onUpdate(config);
        this.onChange();
    }

    /**
     * Move a widget up or down
     */
    moveWidget(config, index, direction, listContainer, onUpdate) {
        const newIndex = index + direction;
        if (newIndex < 0 || newIndex >= config.widgets.length) return;

        const widget = config.widgets[index];
        config.widgets.splice(index, 1);
        config.widgets.splice(newIndex, 0, widget);

        this.renderWidgetList(config, listContainer, onUpdate);
        onUpdate(config);
        this.onChange();
    }

    /**
     * Duplicate a widget
     */
    duplicateWidget(config, index, listContainer, onUpdate) {
        const widget = config.widgets[index];
        const copy = JSON.parse(JSON.stringify(widget));

        // Offset position slightly so it's visible
        if (copy.position) {
            copy.position.x = (copy.position.x || 0) + 10;
            copy.position.y = (copy.position.y || 0) + 10;
        }

        config.widgets.splice(index + 1, 0, copy);
        this.renderWidgetList(config, listContainer, onUpdate);
        onUpdate(config);
        this.onChange();
    }
}

// Export for use in other modules
window.WidgetEditor = WidgetEditor;
