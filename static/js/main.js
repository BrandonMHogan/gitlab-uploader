document.addEventListener('DOMContentLoaded', function() {
    // Initialize select2
    $('#project_id').select2({
        placeholder: 'Search for a partner project',
        allowClear: true,
        width: '100%'
    });

    // Get DOM elements
    const fileUpload = document.querySelector('.file-upload');
    const fileInput = document.getElementById('files');
    const selectedFilesDiv = document.querySelector('.selected-files');
    const form = document.getElementById('uploadForm');
    const button = form.querySelector('button');
    const messageDiv = document.getElementById('message');
    const detailsDiv = document.getElementById('details');
    const progressContainer = document.querySelector('.progress-container');
    const progressFill = document.querySelector('.progress-fill');
    const progressPercentage = document.querySelector('.progress-percentage');

    // File upload click handling
    fileUpload.addEventListener('click', () => {
        fileInput.click();
    });

    fileInput.addEventListener('click', (e) => {
        e.stopPropagation();
    });

    // Drag and drop handlers
    fileUpload.addEventListener('dragover', (e) => {
        e.preventDefault();
        e.stopPropagation();
        fileUpload.style.borderColor = '#0366d6';
        fileUpload.style.backgroundColor = 'rgba(3, 102, 214, 0.1)';
    });

    fileUpload.addEventListener('dragleave', (e) => {
        e.preventDefault();
        e.stopPropagation();
        fileUpload.style.borderColor = '#e1e4e8';
        fileUpload.style.backgroundColor = 'transparent';
    });

    fileUpload.addEventListener('drop', (e) => {
        e.preventDefault();
        e.stopPropagation();
        fileUpload.style.borderColor = '#e1e4e8';
        fileUpload.style.backgroundColor = 'transparent';
        
        const dt = e.dataTransfer;
        const files = dt.files;
        
        fileInput.files = files;
        updateFileList();
    });

    // Update file list
    function updateFileList() {
        selectedFilesDiv.innerHTML = '';
        const files = Array.from(fileInput.files);
        
        if (files.length === 0) {
            return;
        }

        files.forEach(file => {
            const fileItem = document.createElement('div');
            fileItem.className = 'file-item';
            fileItem.innerHTML = `
                <span class="file-name">${file.name}</span>
                <span class="file-size">(${formatFileSize(file.size)})</span>
            `;
            selectedFilesDiv.appendChild(fileItem);
        });
    }

    function formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    // File selection handler
    fileInput.addEventListener('change', updateFileList);

    // Form submission
    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        
        const formData = new FormData(form);
        const files = fileInput.files;
        
        if (files.length === 0) {
            showMessage('Please select files to upload', 'error');
            return;
        }

        if (files.length > 2) {
            showMessage('Please select a maximum of 2 files', 'error');
            return;
        }

        let hasPom = false;
        let hasAar = false;
        for (let file of files) {
            if (file.name.endsWith('.pom')) hasPom = true;
            if (file.name.endsWith('.aar')) hasAar = true;
        }

        if (!hasPom || !hasAar) {
            showMessage('Please select both AAR and POM files', 'error');
            return;
        }

        // Show loading state
        showLoading(true);
        
        try {
            let progress = 0;
            const progressInterval = setInterval(() => {
                if (progress < 90) {
                    progress += 10;
                    updateProgress(progress);
                }
            }, 500);

            const response = await fetch('/upload', {
                method: 'POST',
                body: formData
            });
            
            clearInterval(progressInterval);
            updateProgress(100);
            
            const result = await response.json();
            
            showMessage(result.message, result.success ? 'success' : 'error');
            if (result.details) {
                showDetails(result.details, result.success ? 'success' : 'error');
            }
        } catch (error) {
            showMessage('An error occurred while uploading the files', 'error');
        } finally {
            showLoading(false);
        }
    });

    function showMessage(text, type) {
        messageDiv.textContent = text;
        messageDiv.className = `message ${type}`;
        messageDiv.style.display = 'block';
    }

    function showDetails(text, type) {
        detailsDiv.textContent = text;
        detailsDiv.className = `message ${type}`;
        detailsDiv.style.display = 'block';
    }

    function showLoading(loading) {
        button.classList.toggle('uploading', loading);
        button.disabled = loading;
        button.querySelector('.button-text').textContent = loading ? 'Uploading...' : 'Upload Files';
        
        if (!loading) {
            setTimeout(() => {
                progressContainer.classList.remove('visible');
                updateProgress(0);
            }, 1000);
        }
    }

    function updateProgress(percent) {
        progressFill.style.width = `${percent}%`;
        progressPercentage.textContent = `${percent}%`;
        if (percent > 0) {
            progressContainer.classList.add('visible');
        }
    }
});