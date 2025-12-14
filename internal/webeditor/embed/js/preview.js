/**
 * PreviewPanel - Live display preview for SteelClock web editor
 * Renders packed bit frames to a canvas element with zoom support
 */

/**
 * @typedef {Object} PreviewInfo
 * @property {boolean} available - Whether preview is available
 * @property {number} width - Display width
 * @property {number} height - Display height
 * @property {number} target_fps - Target frame rate
 */

/**
 * @typedef {Object} PreviewConfigMessage
 * @property {string} type - Message type ('config')
 * @property {number} width - Display width
 * @property {number} height - Display height
 * @property {number} target_fps - Target frame rate
 */

class PreviewPanel {
    static STORAGE_KEY = 'steelclock_preview_settings';

    constructor() {
        this.container = null;
        this.canvas = null;
        this.ctx = null;
        this.ws = null;

        this.config = {
            width: 128,
            height: 40,
            targetFPS: 30,
        };

        this.zoom = 4; // Default zoom level
        this.isLive = false;
        this.isVisible = false;
        this.available = false;
        this.position = null; // {left, top} or null for default

        this.frameCount = 0;
        this.lastFrameTime = 0;
        this.fps = 0;

        // Load saved settings
        this.loadSettings();

        // Create panel elements
        this.createPanel();

        // Restore settings after panel is created
        this.restoreSettings();

        // Add page unload listener to reset preview override
        window.addEventListener('beforeunload', () => {
            if (this.isVisible) {
                // Use sendBeacon for reliable delivery during page unload
                navigator.sendBeacon('/api/preview/override', JSON.stringify({ enable: false }));
            }
        });
    }

    /**
     * Load settings from localStorage
     */
    loadSettings() {
        try {
            const saved = localStorage.getItem(PreviewPanel.STORAGE_KEY);
            if (saved) {
                const settings = JSON.parse(saved);
                if (settings.zoom) this.zoom = settings.zoom;
                if (settings.position) this.position = settings.position;
                if (settings.wasVisible) this.shouldAutoShow = true;
            }
        } catch (err) {
            console.warn('Failed to load preview settings:', err);
        }
    }

    /**
     * Save settings to localStorage
     */
    saveSettings() {
        try {
            const settings = {
                zoom: this.zoom,
                position: this.position,
                wasVisible: this.isVisible,
            };
            localStorage.setItem(PreviewPanel.STORAGE_KEY, JSON.stringify(settings));
        } catch (err) {
            console.warn('Failed to save preview settings:', err);
        }
    }

    /**
     * Restore settings after panel creation
     */
    restoreSettings() {
        // Restore zoom
        const zoomSelect = document.getElementById('preview-zoom');
        if (zoomSelect) {
            zoomSelect.value = this.zoom.toString();
        }

        // Restore position
        if (this.position) {
            this.container.style.left = this.position.left + 'px';
            this.container.style.top = this.position.top + 'px';
            this.container.style.right = 'auto';
        }

        // Auto-show if was visible before
        if (this.shouldAutoShow) {
            // Defer to allow page to fully load
            setTimeout(() => this.show(), 100);
        }
    }

    /**
     * Create the preview panel DOM elements
     */
    createPanel() {
        // Create container
        this.container = document.createElement('div');
        this.container.id = 'preview-panel';
        this.container.className = 'preview-panel';
        this.container.innerHTML = `
            <div class="preview-header">
                <span class="preview-title">Display Preview</span>
                <div class="preview-controls">
                    <select id="preview-zoom" title="Zoom level" style="width: 6em;">
                        <option value="1">1x</option>
                        <option value="2">2x</option>
                        <option value="4" selected>4x</option>
                        <option value="8">8x</option>
                    </select>
                    <button id="preview-mode" class="outline secondary" title="Toggle live/static mode" style="display: none;">Static</button>
                    <button id="preview-close" class="outline secondary" title="Close preview">X</button>
                </div>
            </div>
            <div class="preview-content">
                <canvas id="preview-canvas"></canvas>
                <div class="preview-status">
                    <span id="preview-fps">0 FPS</span>
                    <span id="preview-size">128x40</span>
                </div>
            </div>
            <div class="preview-unavailable" style="display: none;">
                <p>Preview not available.</p>
                <p>Set <code>backend: "webclient"</code> in config to enable.</p>
            </div>
        `;

        // Add to body
        document.body.appendChild(this.container);

        // Get references
        this.canvas = document.getElementById('preview-canvas');
        this.ctx = this.canvas.getContext('2d');

        // Set up event listeners
        document.getElementById('preview-zoom').addEventListener('change', (e) => {
            this.setZoom(parseInt(e.target.value, 10));
        });

        document.getElementById('preview-mode').addEventListener('click', () => {
            this.toggleMode();
        });

        document.getElementById('preview-close').addEventListener('click', async () => {
            await this.hide();
        });

        // Make draggable
        this.makeDraggable();
    }

    /**
     * Make the panel draggable by its header
     */
    makeDraggable() {
        const header = this.container.querySelector('.preview-header');
        let isDragging = false;
        let startX, startY, startLeft, startTop;

        header.addEventListener('mousedown', (e) => {
            if (e.target.tagName === 'BUTTON' || e.target.tagName === 'SELECT') return;

            isDragging = true;
            startX = e.clientX;
            startY = e.clientY;

            const rect = this.container.getBoundingClientRect();
            startLeft = rect.left;
            startTop = rect.top;

            header.style.cursor = 'grabbing';
            e.preventDefault();
        });

        document.addEventListener('mousemove', (e) => {
            if (!isDragging) return;

            const dx = e.clientX - startX;
            const dy = e.clientY - startY;

            this.container.style.left = (startLeft + dx) + 'px';
            this.container.style.top = (startTop + dy) + 'px';
            this.container.style.right = 'auto';
        });

        document.addEventListener('mouseup', () => {
            if (isDragging) {
                isDragging = false;
                header.style.cursor = 'grab';
                // Save position after drag
                const rect = this.container.getBoundingClientRect();
                this.position = { left: rect.left, top: rect.top };
                this.saveSettings();
            }
        });
    }

    /**
     * Initialize the preview panel by checking availability
     */
    async init() {
        try {
            /** @type {PreviewInfo} */
            const info = await API.getPreviewInfo();

            if (info.available) {
                this.available = true;
                this.config.width = info.width;
                this.config.height = info.height;
                this.config.targetFPS = info.target_fps;

                this.updateCanvasSize();

                // Update status display
                document.getElementById('preview-size').textContent =
                    `${info.width}x${info.height}`;

                // Show canvas, hide unavailable message
                this.container.querySelector('.preview-content').style.display = 'block';
                this.container.querySelector('.preview-unavailable').style.display = 'none';
            } else {
                this.available = false;
                // Show unavailable message
                this.container.querySelector('.preview-content').style.display = 'none';
                this.container.querySelector('.preview-unavailable').style.display = 'block';
            }
        } catch (err) {
            console.warn('Preview not available:', err);
            this.available = false;
            this.container.querySelector('.preview-content').style.display = 'none';
            this.container.querySelector('.preview-unavailable').style.display = 'block';
        }
    }

    /**
     * Show the preview panel
     */
    async show() {
        if (!this.isVisible) {
            // Enable preview override (temporary switch to preview backend)
            try {
                await API.setPreviewOverride(true);
                // Wait a moment for backend to switch and become available
                await new Promise(resolve => setTimeout(resolve, 500));
            } catch (err) {
                console.warn('Failed to enable preview override:', err);
                // Continue anyway - might already be using preview backend
            }

            await this.init();
            this.container.style.display = 'block';
            this.isVisible = true;
            this.saveSettings();
            this.syncCheckbox(true);

            // Start in live mode by default
            if (this.available && !this.isLive) {
                this.startLive();
            }
        }
    }

    /**
     * Hide the preview panel
     */
    async hide() {
        this.container.style.display = 'none';
        this.isVisible = false;
        this.stopLive();
        this.saveSettings();
        this.syncCheckbox(false);

        // Disable preview override (restore original backend)
        try {
            await API.setPreviewOverride(false);
        } catch (err) {
            console.warn('Failed to disable preview override:', err);
        }
    }

    /**
     * Sync the header checkbox with panel visibility
     * @param {boolean} checked - Whether checkbox should be checked
     */
    syncCheckbox(checked) {
        const checkbox = document.getElementById('preview-toggle-checkbox');
        if (checkbox) {
            checkbox.checked = checked;
        }
    }

    /**
     * Set zoom level
     * @param {number} zoom - Zoom multiplier (1, 2, 4, 8)
     */
    setZoom(zoom) {
        this.zoom = zoom;
        this.updateCanvasSize();
        this.saveSettings();
    }

    /**
     * Update canvas dimensions based on config and zoom
     */
    updateCanvasSize() {
        this.canvas.width = this.config.width * this.zoom;
        this.canvas.height = this.config.height * this.zoom;
        this.ctx.imageSmoothingEnabled = false;
    }

    /**
     * Toggle between live and static mode
     */
    toggleMode() {
        if (this.isLive) {
            this.stopLive();
        } else {
            this.startLive();
        }
    }

    /**
     * Start live preview via WebSocket
     */
    startLive() {
        if (!this.available || this.isLive) return;

        this.isLive = true;
        document.getElementById('preview-mode').textContent = 'Live';
        document.getElementById('preview-mode').classList.remove('secondary');

        this.frameCount = 0;
        this.lastFrameTime = performance.now();

        // Connect WebSocket
        this.ws = API.createPreviewWebSocket();

        this.ws.onmessage = (event) => {
            try {
                /** @type {{type: string, frame?: string, width?: number, height?: number, target_fps?: number}} */
                const data = JSON.parse(event.data);

                if (data.type === 'frame') {
                    this.renderFrame(data.frame);
                    this.updateFPS();
                } else if (data.type === 'config') {
                    // Update config if server sends it
                    this.config.width = data.width;
                    this.config.height = data.height;
                    this.config.targetFPS = data.target_fps;
                    this.updateCanvasSize();
                    document.getElementById('preview-size').textContent =
                        `${data.width}x${data.height}`;
                }
            } catch (err) {
                console.error('Preview: failed to parse message:', err);
            }
        };

        this.ws.onclose = () => {
            if (this.isLive) {
                // Reconnect after a delay
                setTimeout(() => {
                    if (this.isLive) {
                        this.startLive();
                    }
                }, 1000);
            }
        };

        this.ws.onerror = (err) => {
            console.error('Preview WebSocket error:', err);
        };
    }

    /**
     * Stop live preview
     */
    stopLive() {
        this.isLive = false;
        document.getElementById('preview-mode').textContent = 'Static';
        document.getElementById('preview-mode').classList.add('secondary');

        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }

        document.getElementById('preview-fps').textContent = '0 FPS';
    }

    /**
     * Render a frame from packed bit data
     * @param {Array|string} frameData - Packed bits as byte array or base64 string
     */
    renderFrame(frameData) {
        // Convert from base64 if needed
        let bytes;
        if (typeof frameData === 'string') {
            const binary = atob(frameData);
            bytes = new Uint8Array(binary.length);
            for (let i = 0; i < binary.length; i++) {
                bytes[i] = binary.charCodeAt(i);
            }
        } else {
            bytes = new Uint8Array(frameData);
        }

        const width = this.config.width;
        const height = this.config.height;

        // Create ImageData at 1x scale
        const imageData = this.ctx.createImageData(width, height);
        const data = imageData.data;

        // Unpack bits to RGBA pixels
        // Each byte contains 8 pixels, MSB first
        let bitIndex = 0;
        for (let y = 0; y < height; y++) {
            for (let x = 0; x < width; x++) {
                const byteIndex = Math.floor(bitIndex / 8);
                const bitPos = 7 - (bitIndex % 8); // MSB first

                let pixelOn = false;
                if (byteIndex < bytes.length) {
                    pixelOn = (bytes[byteIndex] & (1 << bitPos)) !== 0;
                }

                // Convert to RGBA (white = on, black = off for OLED)
                const pixelIndex = (y * width + x) * 4;
                const color = pixelOn ? 255 : 0;
                data[pixelIndex] = color;     // R
                data[pixelIndex + 1] = color; // G
                data[pixelIndex + 2] = color; // B
                data[pixelIndex + 3] = 255;   // A

                bitIndex++;
            }
        }

        // Draw at 1x scale first, then scale up
        // Create a temporary canvas for 1x rendering
        const tempCanvas = document.createElement('canvas');
        tempCanvas.width = width;
        tempCanvas.height = height;
        const tempCtx = tempCanvas.getContext('2d');
        tempCtx.putImageData(imageData, 0, 0);

        // Clear and draw scaled
        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
        this.ctx.drawImage(tempCanvas, 0, 0, this.canvas.width, this.canvas.height);
    }

    /**
     * Update FPS counter
     */
    updateFPS() {
        this.frameCount++;
        const now = performance.now();
        const elapsed = now - this.lastFrameTime;

        if (elapsed >= 1000) {
            this.fps = Math.round(this.frameCount * 1000 / elapsed);
            document.getElementById('preview-fps').textContent = `${this.fps} FPS`;
            this.frameCount = 0;
            this.lastFrameTime = now;
        }
    }
}

// Create global instance
window.previewPanel = new PreviewPanel();
