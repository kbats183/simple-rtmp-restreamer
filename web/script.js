class RTMPRestreamer {
    constructor() {
        this.apiBase = '/api/streams';
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadStreams();
        setInterval(() => this.loadStreams(), 5000); // Refresh every 5 seconds
    }

    setupEventListeners() {
        document.getElementById('addStreamBtn').addEventListener('click', () => this.showAddStreamModal());
        document.getElementById('refreshBtn').addEventListener('click', () => this.loadStreams());
        
        // Modal controls
        document.querySelectorAll('.close').forEach(closeBtn => {
            closeBtn.addEventListener('click', () => this.closeModals());
        });
        
        document.querySelectorAll('.cancel-btn').forEach(cancelBtn => {
            cancelBtn.addEventListener('click', () => this.closeModals());
        });

        // Click outside modal to close
        window.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal')) {
                this.closeModals();
            }
        });

        // Form submissions
        document.getElementById('addStreamForm').addEventListener('submit', (e) => this.handleAddStream(e));
        document.getElementById('addTargetForm').addEventListener('submit', (e) => this.handleAddTarget(e));
    }

    async loadStreams() {
        try {
            const response = await fetch(this.apiBase);
            if (!response.ok) throw new Error(`HTTP ${response.status}`);
            
            const streams = await response.json();
            await this.renderStreams(streams || []);
        } catch (error) {
            console.error('Failed to load streams:', error);
            this.showError('Failed to load streams: ' + error.message);
        }
    }

    async renderStreams(streams) {
        const container = document.getElementById('streamsList');
        
        if (streams.length === 0) {
            container.innerHTML = '<div class="loading">No streams configured. Click "Add New Stream" to get started.</div>';
            return;
        }

        // Load status for all streams
        const streamStatuses = {};
        for (const stream of streams) {
            try {
                const statusResponse = await fetch(`${this.apiBase}/${encodeURIComponent(stream.name)}/status`);
                if (statusResponse.ok) {
                    streamStatuses[stream.name] = await statusResponse.json();
                }
            } catch (error) {
                console.error(`Failed to load status for ${stream.name}:`, error);
            }
        }

        const html = streams.map(stream => this.renderStreamCard(stream, streamStatuses[stream.name])).join('');
        container.innerHTML = html;

        // Add event listeners for stream actions
        container.querySelectorAll('.delete-stream').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const streamName = e.target.dataset.streamName;
                this.deleteStream(streamName);
            });
        });

        container.querySelectorAll('.add-target').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const streamName = e.target.dataset.streamName;
                this.showAddTargetModal(streamName);
            });
        });
    }

    renderStreamCard(stream, status) {
        const isLive = status?.is_live || false;
        const bitrate = status?.bitrate || 0;
        const lastFrameTime = status?.last_frame_time ? new Date(status.last_frame_time * 1000).toLocaleString() : 'Never';
        
        const targetsHtml = stream.targets && stream.targets.length > 0 
            ? stream.targets.map(target => `<div class="target-item">${this.escapeHtml(target)}</div>`).join('')
            : '<div class="target-item">No targets configured</div>';

        return `
            <div class="stream-card ${isLive ? 'live' : 'offline'}">
                <div class="stream-header">
                    <div class="stream-name">${this.escapeHtml(stream.name)}</div>
                    <div class="stream-status ${isLive ? 'live' : 'offline'}">
                        <div class="status-dot"></div>
                        ${isLive ? 'LIVE' : 'OFFLINE'}
                    </div>
                </div>
                
                <div class="stream-info">
                    <div><strong>Bitrate:</strong> ${bitrate} kbps</div>
                    <div><strong>Last Frame:</strong> ${lastFrameTime}</div>
                    <div><strong>RTMP URL:</strong> rtmp://localhost:1935/live/${this.escapeHtml(stream.name)}</div>
                </div>

                <div class="stream-targets">
                    <h4>Push Targets (${stream.targets ? stream.targets.length : 0}):</h4>
                    ${targetsHtml}
                </div>

                <div class="stream-actions">
                    <button class="btn btn-primary btn-small add-target" data-stream-name="${this.escapeHtml(stream.name)}">
                        Add Target
                    </button>
                    <button class="btn btn-danger btn-small delete-stream" data-stream-name="${this.escapeHtml(stream.name)}">
                        Delete Stream
                    </button>
                </div>
            </div>
        `;
    }

    showAddStreamModal() {
        document.getElementById('addStreamModal').style.display = 'block';
        document.getElementById('streamName').focus();
    }

    showAddTargetModal(streamName) {
        this.currentStreamName = streamName;
        document.getElementById('addTargetModal').style.display = 'block';
        document.getElementById('targetUrl').focus();
    }

    closeModals() {
        document.querySelectorAll('.modal').forEach(modal => {
            modal.style.display = 'none';
        });
        this.currentStreamName = null;
    }

    async handleAddStream(e) {
        e.preventDefault();
        
        const name = document.getElementById('streamName').value.trim();
        const targetsText = document.getElementById('streamTargets').value.trim();
        
        if (!name) {
            this.showError('Stream name is required');
            return;
        }

        const targets = targetsText ? 
            targetsText.split('\n').map(t => t.trim()).filter(t => t) : 
            [];

        try {
            const response = await fetch(this.apiBase, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    name: name,
                    targets: targets
                })
            });

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new Error(errorData.error || `HTTP ${response.status}`);
            }

            this.closeModals();
            document.getElementById('addStreamForm').reset();
            this.loadStreams();
            this.showSuccess('Stream created successfully');
        } catch (error) {
            console.error('Failed to create stream:', error);
            this.showError('Failed to create stream: ' + error.message);
        }
    }

    async handleAddTarget(e) {
        e.preventDefault();
        
        const targetUrl = document.getElementById('targetUrl').value.trim();
        
        if (!targetUrl) {
            this.showError('Target URL is required');
            return;
        }

        if (!this.currentStreamName) {
            this.showError('No stream selected');
            return;
        }

        try {
            const response = await fetch(`${this.apiBase}/${encodeURIComponent(this.currentStreamName)}/targets`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    target: targetUrl
                })
            });

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new Error(errorData.error || `HTTP ${response.status}`);
            }

            this.closeModals();
            document.getElementById('addTargetForm').reset();
            this.loadStreams();
            this.showSuccess('Target added successfully');
        } catch (error) {
            console.error('Failed to add target:', error);
            this.showError('Failed to add target: ' + error.message);
        }
    }

    async deleteStream(streamName) {
        if (!confirm(`Are you sure you want to delete stream "${streamName}"?`)) {
            return;
        }

        try {
            const response = await fetch(`${this.apiBase}/${encodeURIComponent(streamName)}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new Error(errorData.error || `HTTP ${response.status}`);
            }

            this.loadStreams();
            this.showSuccess('Stream deleted successfully');
        } catch (error) {
            console.error('Failed to delete stream:', error);
            this.showError('Failed to delete stream: ' + error.message);
        }
    }

    showError(message) {
        this.showNotification(message, 'error');
    }

    showSuccess(message) {
        this.showNotification(message, 'success');
    }

    showNotification(message, type) {
        // Remove existing notifications
        document.querySelectorAll('.notification').forEach(n => n.remove());
        
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 15px 20px;
            border-radius: 5px;
            color: white;
            font-weight: 500;
            z-index: 1001;
            max-width: 400px;
            word-wrap: break-word;
            ${type === 'error' ? 'background-color: #e74c3c;' : 'background-color: #27ae60;'}
        `;
        notification.textContent = message;
        
        document.body.appendChild(notification);
        
        setTimeout(() => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 5000);
    }

    escapeHtml(unsafe) {
        return unsafe
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#039;");
    }
}

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    new RTMPRestreamer();
});