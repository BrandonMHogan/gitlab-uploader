// Initialize select2
$(document).ready(function() {
    $('#project_id').select2({
        placeholder: 'Search for a partner or project ID',
        allowClear: true,
        width: '100%'
    });
});

class FileUploader {
    constructor() {
        this.form = document.getElementById('uploadForm');
        this.fileInput = document.getElementById('files');
        this.button = this.form.querySelector('button');
        this.messageDiv = document.getElementById('message');
        this.detailsDiv = document.getElementById('details');
        this.progressContainer = document.querySelector('.progress-container');
        this.progressFill = document.querySelector('.progress-fill');
        this.progressPercentage = document.querySelector('.progress-percentage');
        
        this.initializeEvents();
    }

    async checkFiles(files, projectId, version, deployToken) {
        const existingFiles = [];
        for (const file of files) {
            const url = this.constructFileUrl(projectId, version, file.name);
            try {
                const response = await fetch(`/check-file?url=${encodeURIComponent(url)}&token=${deployToken}`);
                const result = await response.json();
                if (result.exists) {
                    existingFiles.push(result);
                }
            } catch (error) {
                console.error('Error checking file:', error);
            }
        }
        return existingFiles;
    }

    async confirmOverwrite(existingFiles) {
        return new Promise((resolve) => {
            const modal = document.createElement('div');
            modal.className = 'modal';
            modal.innerHTML = `
                <div class="modal-content">
                    <h2>Warning: Files Already Exist</h2>
                    <p>The following files already exist:</p>
                    <ul>
                        ${existingFiles.map(file => `
                            <li>${file.fileName} (Last modified: ${file.updatedAt})</li>
                        `).join('')}
                    </ul>
                    <p>Do you want to replace these files?</p>
                    <div class="modal-buttons">
                        <button class="btn-cancel">Cancel</button>
                        <button class="btn-confirm">Replace</button>
                    </div>
                </div>
            `;

            document.body.appendChild(modal);

            modal.querySelector('.btn-cancel').onclick = () => {
                modal.remove();
                resolve(false);
            };

            modal.querySelector('.btn-confirm').onclick = () => {
                modal.remove();
                resolve(true);
            };
        });
    }

    async handleSubmit(e) {
        e.preventDefault();
        
        const formData = new FormData(this.form);
        const files = this.fileInput.files;
        
        if (!this.validateFiles(files)) {
            return;
        }

        const projectId = formData.get('project_id');
        const version = formData.get('version');
        const deployToken = formData.get('deploy_token');

        // Check if files exist
        const existingFiles = await this.checkFiles(files, projectId, version, deployToken);
        
        if (existingFiles.length > 0) {
            const shouldOverwrite = await this.confirmOverwrite(existingFiles);
            if (!shouldOverwrite) {
                return;
            }
        }

        await this.uploadFiles(formData);
    }

    // ... rest of your existing JavaScript methods ...
}

// Initialize the uploader
const uploader = new FileUploader();