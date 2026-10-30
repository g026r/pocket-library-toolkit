"use strict";

/**
 * Application logic for Pocket Library Toolkit web interface
 * Handles file loading, UI updates, entry editing, and binary generation
 */

class PocketLibraryApp {
    constructor() {
        // Core state
        this.binaryParser = new PocketBinary();
        this.listEntries = [];
        this.playtimesEntries = [];
        this.filteredEntries = [];
        this.selectedEntry = null;
        this.selectedSystem = null;
        this.systems = new Set();
        
        // UI elements
        this.ui = {
            uploadSection: document.getElementById('uploadSection'),
            fileUploadArea: document.getElementById('fileUploadArea'),
            listBinInput: document.getElementById('listBinInput'),
            playtimesBinInput: document.getElementById('playtimesBinInput'),
            uploadStatus: document.getElementById('uploadStatus'),
            validationErrors: document.getElementById('validationErrors'),
            libraryContainer: document.getElementById('libraryContainer'),
            loadOtherFilesBtn: document.getElementById('loadOtherFilesBtn'),
            entryCount: document.getElementById('entryCount'),
            libraryBody: document.getElementById('libraryBody'),
            searchInput: document.getElementById('searchInput'),
            clearSearchBtn: document.getElementById('clearSearchBtn'),
            systemFilters: document.getElementById('systemFilters'),
            noResults: document.getElementById('noResults'),
            exportSection: document.getElementById('exportSection'),
            downloadListBin: document.getElementById('downloadListBin'),
            downloadPlaytimesBin: document.getElementById('downloadPlaytimesBin'),
            downloadBoth: document.getElementById('downloadBoth'),
            editModal: document.getElementById('editModal'),
            editGameName: document.getElementById('editGameName'),
            editPlaytime: document.getElementById('editPlaytime'),
            closeModal: document.getElementById('closeModal'),
            cancelEdit: document.getElementById('cancelEdit'),
            saveEdit: document.getElementById('saveEdit')
        };

        // Initialize event listeners
        this.initEventListeners();
    }

    /**
     * Initialize all event listeners
     */
    initEventListeners() {
        // File input events
        this.ui.listBinInput.addEventListener('change', (e) => this.handleFileInput(e, 'list'));
        this.ui.playtimesBinInput.addEventListener('change', (e) => this.handleFileInput(e, 'playtimes'));
        
        // Load other files button
        if (this.ui.loadOtherFilesBtn) {
            this.ui.loadOtherFilesBtn.addEventListener('click', () => this.resetToUploadState());
        }
        
        // Drag and drop events
        this.ui.fileUploadArea.addEventListener('dragover', (e) => this.handleDragOver(e));
        this.ui.fileUploadArea.addEventListener('dragleave', (e) => this.handleDragLeave(e));
        this.ui.fileUploadArea.addEventListener('drop', (e) => this.handleDrop(e));
        
        // Search functionality
        this.ui.searchInput.addEventListener('input', () => this.applyFilters());
        this.ui.clearSearchBtn.addEventListener('click', () => {
            this.ui.searchInput.value = '';
            this.applyFilters();
            this.ui.searchInput.focus();
        });
        
        // Export buttons
        this.ui.downloadListBin.addEventListener('click', () => this.downloadFile('list'));
        this.ui.downloadPlaytimesBin.addEventListener('click', () => this.downloadFile('playtimes'));
        this.ui.downloadBoth.addEventListener('click', () => this.downloadFile('both'));
        
        // Modal events
        this.ui.closeModal.addEventListener('click', () => this.closeEditModal());
        this.ui.cancelEdit.addEventListener('click', () => this.closeEditModal());
        this.ui.saveEdit.addEventListener('click', () => this.saveEntryEdit());
        
        this.ui.editGameName.addEventListener('keyup', (e) => {
            if (e.key === 'Enter') {
                this.saveEntryEdit();
            } else if (e.key === 'Escape') {
                this.closeEditModal();
            }
        });

        this.ui.editPlaytime.addEventListener('keyup', (e) => {
            if (e.key === 'Enter') {
                this.saveEntryEdit();
            } else if (e.key === 'Escape') {
                this.closeEditModal();
            }
        });

        this.ui.editPlaytime.addEventListener('input', () => {
            const val = parseInt(this.ui.editPlaytime.value, 10);
            if (isNaN(val) || val < 0 || val > 67108863) {
                this.ui.editPlaytime.classList.add('error-input');
            } else {
                this.ui.editPlaytime.classList.remove('error-input');
            }
        });
        
        // Modal click outside to close
        this.ui.editModal.addEventListener('click', (e) => {
            if (e.target === this.ui.editModal) {
                this.closeEditModal();
            }
        });
    }

    /**
     * Reset app state to show file upload interface
     */
    resetToUploadState() {
        this.listEntries = [];
        this.playtimesEntries = [];
        this.filteredEntries = [];
        this.selectedSystem = null;
        this.ui.listBinInput.value = '';
        this.ui.playtimesBinInput.value = '';
        this.ui.searchInput.value = '';
        this.ui.uploadSection.style.display = 'block';
        this.ui.libraryContainer.style.display = 'none';
        this.ui.exportSection.style.display = 'none';
        this.showStatus('Ready to load list.bin and playtimes.bin files');
    }

    /**
     * Handle file input selection
     */
    handleFileInput(event, type) {
        const file = event.target.files[0];
        if (!file) return;
        
        this.readBinaryFile(file, type);
    }

    /**
     * Handle drag over event
     */
    handleDragOver(event) {
        event.preventDefault();
        this.ui.fileUploadArea.classList.add('drag-over');
    }

    /**
     * Handle drag leave event
     */
    handleDragLeave(event) {
        event.preventDefault();
        this.ui.fileUploadArea.classList.remove('drag-over');
    }

    /**
     * Handle file drop event
     */
    handleDrop(event) {
        event.preventDefault();
        this.ui.fileUploadArea.classList.remove('drag-over');
        
        const files = event.dataTransfer.files;
        if (files.length === 0) return;
        
        // Look for list.bin and playtimes.bin files
        for (let i = 0; i < files.length; i++) {
            const file = files[i];
            const fname = file.name.toLowerCase();
            if (fname === 'list.bin') {
                this.readBinaryFile(file, 'list');
            } else if (fname === 'playtimes.bin') {
                this.readBinaryFile(file, 'playtimes');
            }
        }
    }

    /**
     * Read binary file and parse it
     */
    async readBinaryFile(file, type) {
        try {
            this.showStatus(`Loading ${file.name}...`);
            
            const arrayBuffer = await this.readFileAsArrayBuffer(file);
            
            if (type === 'list') {
                this.listEntries = await this.parseListBin(arrayBuffer);
                this.showStatus(`${file.name} loaded successfully. Found ${this.listEntries.length} entries.`);
            } else if (type === 'playtimes') {
                this.playtimesEntries = await this.parsePlaytimesBin(arrayBuffer);
                this.showStatus(`${file.name} loaded successfully. Found ${this.playtimesEntries.length} entries.`);
            }
            
            // Check if both files are loaded
            if (this.listEntries.length > 0 && this.playtimesEntries.length > 0) {
                await this.validateAndLoad();
            } else if (this.listEntries.length > 0) {
                this.showStatus(`list.bin loaded (${this.listEntries.length} entries). Please select or drop playtimes.bin.`);
            } else if (this.playtimesEntries.length > 0) {
                this.showStatus(`playtimes.bin loaded (${this.playtimesEntries.length} entries). Please select or drop list.bin.`);
            }
            
        } catch (error) {
            console.error(`Error loading ${file.name}:`, error);
            this.showError(`Failed to load ${file.name}: ${error.message}`);
        }
    }

    /**
     * Parse list.bin buffer
     */
    async parseListBin(arrayBuffer) {
        return new Promise((resolve, reject) => {
            try {
                const entries = this.binaryParser.parseListBin(arrayBuffer);
                resolve(entries.entries);
            } catch (error) {
                reject(error);
            }
        });
    }

    /**
     * Parse playtimes.bin buffer
     */
    async parsePlaytimesBin(arrayBuffer) {
        return new Promise((resolve, reject) => {
            try {
                const result = this.binaryParser.parsePlaytimesBin(arrayBuffer);
                resolve(result.playtimes);
            } catch (error) {
                reject(error);
            }
        });
    }

    /**
     * Validate files and load into UI
     */
    async validateAndLoad() {
        // Synchronize entries between list.bin and playtimes.bin
        const syncResult = this.binaryParser.validateSynchronization(this.listEntries, this.playtimesEntries);
        
        if (!syncResult.valid) {
            this.showError(`Validation Error: ${syncResult.errors.join(', ')}`);
            return;
        }
        
        // Match playtime data with list entries
        this.matchPlaytimesToEntries();
        
        // Sort entries
        this.listEntries = this.binaryParser.sortEntries(this.listEntries);
        
        // Build system filters
        this.buildSystemFilters();
        
        // Hide upload section and show library interface
        this.ui.uploadSection.style.display = 'none';
        this.ui.libraryContainer.style.display = 'block';
        this.ui.exportSection.style.display = 'block';
        
        this.selectedSystem = null;
        this.ui.searchInput.value = '';
        this.applyFilters();
        
        this.showStatus(`Library loaded successfully (${this.listEntries.length} entries).`);
    }

    /**
     * Match playtime data with library entries
     */
    matchPlaytimesToEntries() {
        for (const listEntry of this.listEntries) {
            const matchingPlaytime = this.playtimesEntries.find(p => p.sig === listEntry.sig);
            if (matchingPlaytime) {
                listEntry.added = matchingPlaytime.added;
                listEntry.played = matchingPlaytime.played;
                listEntry.system = matchingPlaytime.system;
            } else {
                listEntry.added = 0;
                listEntry.played = 0;
            }
        }
    }

    /**
     * Build system filter badges
     */
    buildSystemFilters() {
        this.systems.clear();
        this.ui.systemFilters.innerHTML = '';
        
        const allActive = this.selectedSystem === null;
        const allColor = '#000000';
        
        // Add "All" filter
        const allBadge = document.createElement('div');
        allBadge.className = allActive ? 'system-badge active' : 'system-badge';
        allBadge.textContent = 'All';
        allBadge.style.borderColor = allColor;
        if (allActive) {
            allBadge.style.backgroundColor = '#ffffff';
            allBadge.style.color = allColor;
        } else {
            allBadge.style.backgroundColor = allColor;
            allBadge.style.color = '#ffffff';
        }
        allBadge.onclick = () => this.filterBySystem(null);
        this.ui.systemFilters.appendChild(allBadge);
        
        // Add system-specific filters
        for (const entry of this.listEntries) {
            const systemName = this.binaryParser.getSystemName(entry.system);
            if (!this.systems.has(systemName)) {
                this.systems.add(systemName);
                
                const isActive = this.selectedSystem === systemName;
                const sysColor = this.getSystemColor(entry.system);
                
                const badge = document.createElement('div');
                badge.className = isActive ? 'system-badge active' : 'system-badge';
                badge.textContent = systemName;
                badge.style.borderColor = sysColor;
                if (isActive) {
                    badge.style.backgroundColor = '#ffffff';
                    badge.style.color = sysColor;
                } else {
                    badge.style.backgroundColor = sysColor;
                    badge.style.color = '#ffffff';
                }
                badge.onclick = () => this.filterBySystem(systemName);
                this.ui.systemFilters.appendChild(badge);
            }
        }
    }

    /**
     * Get system color for badge
     */
    getSystemColor(system) {
        const colors = {
            0: '#10b981', // GB
            1: '#3b82f6', // GBC
            2: '#ef4444', // GBA
            3: '#8b5cf6', // GG
            4: '#f97316', // SMS
            5: '#84cc16', // NGP
            6: '#06b6d4', // NGPC
            7: '#ec4899', // PCE
            8: '#14b8a6'  // Lynx
        };
        return colors[system] || '#64748b';
    }

    /**
     * Filter entries by system
     */
    filterBySystem(systemName) {
        this.selectedSystem = systemName;
        
        // Re-build system filters to update active/inactive styling
        this.buildSystemFilters();
        
        this.applyFilters();
    }

    /**
     * Apply combined system filter and search filtering
     */
    applyFilters() {
        const searchTerm = this.ui.searchInput.value.toLowerCase().trim();
        
        if (searchTerm) {
            this.ui.clearSearchBtn.style.display = 'block';
        } else {
            this.ui.clearSearchBtn.style.display = 'none';
        }
        
        this.filteredEntries = this.listEntries.filter(entry => {
            // Check system filter
            if (this.selectedSystem !== null) {
                const systemName = this.binaryParser.getSystemName(entry.system);
                if (systemName !== this.selectedSystem) {
                    return false;
                }
            }
            
            // Check search term
            if (searchTerm) {
                const nameMatch = entry.name.toLowerCase().includes(searchTerm);
                const sysName = this.binaryParser.getSystemName(entry.system).toLowerCase();
                const sysFullName = this.binaryParser.getSystemFullName(entry.system).toLowerCase();
                
                let systemMatch = sysName.includes(searchTerm) || sysFullName.includes(searchTerm);
                
                // Synonyms support (e.g. PC Engine <-> TurboGrafx 16 / TG16)
                if (entry.system === 7 /* PCE */) {
                    const synonyms = ["turbografx 16", "turbografx-16", "turbografx", "tg16", "tg-16", "supergrafx", "sgfx"];
                    if (synonyms.some(syn => syn.includes(searchTerm) || searchTerm.includes(syn))) {
                        systemMatch = true;
                    }
                }
                
                const sigMatch = entry.sig.toString(16).padStart(8, '0').includes(searchTerm);
                const crcMatch = entry.crc32.toString(16).padStart(8, '0').includes(searchTerm);
                return nameMatch || systemMatch || sigMatch || crcMatch;
            }
            
            return true;
        });
        
        this.updateEntryCount();
        this.displayEntries();
    }

    /**
     * Display entries in the table
     */
    displayEntries() {
        this.ui.libraryBody.innerHTML = '';
        
        if (this.filteredEntries.length === 0) {
            this.ui.noResults.style.display = 'block';
            return;
        } else {
            this.ui.noResults.style.display = 'none';
        }
        
        for (const entry of this.filteredEntries) {
            const row = this.createEntryRow(entry);
            this.ui.libraryBody.appendChild(row);
        }
    }

    /**
     * Create a table row for an entry
     */
    createEntryRow(entry) {
        const row = document.createElement('tr');
        if (entry.modified) {
            row.classList.add('modified');
        }
        
        // Game Name cell
        const nameCell = document.createElement('td');
        nameCell.textContent = entry.name;
        
        // System cell
        const systemCell = document.createElement('td');
        const systemBadge = document.createElement('span');
        systemBadge.className = 'system-badge-display';
        systemBadge.textContent = this.binaryParser.getSystemName(entry.system);
        systemBadge.style.backgroundColor = this.getSystemColor(entry.system);
        systemCell.appendChild(systemBadge);
        
        // Signature cell
        const sigCell = document.createElement('td');
        sigCell.textContent = `0x${entry.sig.toString(16).padStart(8, '0')}`;
        
        // CRC32 cell
        const crc32Cell = document.createElement('td');
        crc32Cell.textContent = `0x${entry.crc32.toString(16).padStart(8, '0')}`;
        
        // Playtime cell
        const playtimeCell = document.createElement('td');
        playtimeCell.textContent = this.binaryParser.formatPlayTime(entry.played);
        
        // Action cell
        const actionCell = document.createElement('td');
        const editBtn = document.createElement('button');
        editBtn.className = 'action-btn';
        editBtn.textContent = 'Edit';
        editBtn.onclick = () => this.openEditModal(entry);
        actionCell.appendChild(editBtn);
        
        row.appendChild(nameCell);
        row.appendChild(systemCell);
        row.appendChild(sigCell);
        row.appendChild(crc32Cell);
        row.appendChild(playtimeCell);
        row.appendChild(actionCell);
        
        return row;
    }

    /**
     * Update entry count display
     */
    updateEntryCount() {
        if (this.filteredEntries.length === this.listEntries.length) {
            this.ui.entryCount.textContent = `(${this.listEntries.length} entries)`;
        } else {
            this.ui.entryCount.textContent = `(${this.filteredEntries.length} of ${this.listEntries.length} entries)`;
        }
    }

    /**
     * Open edit modal for an entry
     */
    openEditModal(entry) {
        this.selectedEntry = entry;
        this.ui.editGameName.value = entry.name;
        this.ui.editPlaytime.value = entry.played;
        this.ui.editPlaytime.classList.remove('error-input');
        this.ui.editModal.classList.add('show');
        this.ui.editGameName.focus();
        this.ui.editGameName.select();
    }

    /**
     * Close edit modal
     */
    closeEditModal() {
        this.ui.editModal.classList.remove('show');
        this.selectedEntry = null;
        this.ui.editGameName.value = '';
        this.ui.editPlaytime.value = '';
        this.ui.editPlaytime.classList.remove('error-input');
    }

    /**
     * Save entry edit
     */
    saveEntryEdit() {
        if (!this.selectedEntry) return;
        
        const newName = this.ui.editGameName.value.trim();
        if (!newName) {
            this.showError('Game name cannot be empty');
            return;
        }
        
        const newPlaytime = parseInt(this.ui.editPlaytime.value, 10);
        if (isNaN(newPlaytime) || newPlaytime < 0 || newPlaytime > 0x03FFFFFF) {
            this.ui.editPlaytime.classList.add('error-input');
            this.showError('Playtime must be a valid number between 0 and 67108863 seconds');
            return;
        }
        this.ui.editPlaytime.classList.remove('error-input');
        
        // Update entry name, playtime, and modified flag
        this.selectedEntry.name = newName;
        this.selectedEntry.played = newPlaytime;
        this.selectedEntry.modified = true;
        
        // Re-sort entries according to models.EntrySort / AC.6
        this.listEntries = this.binaryParser.sortEntries(this.listEntries);
        
        // Re-apply filters and update UI table
        this.applyFilters();
        
        // Close modal
        this.closeEditModal();
        
        // Show status message
        this.showStatus('Entry and playtime updated and library re-sorted. Click export buttons to download updated files.');
    }

    /**
     * Download generated binary file
     */
    downloadFile(type) {
        if (this.listEntries.length === 0) {
            this.showError('No library data to export');
            return;
        }
        
        try {
            let buffer, filename;
            
            if (type === 'list' || type === 'both') {
                buffer = this.binaryParser.serializeLibrary(this.listEntries);
                filename = 'list.bin';
                this.downloadBuffer(buffer, filename);
            }
            
            if (type === 'playtimes' || type === 'both') {
                buffer = this.binaryParser.serializePlaytimes(this.listEntries);
                filename = 'playtimes.bin';
                this.downloadBuffer(buffer, filename);
            }
            
            this.showStatus(`Successfully downloaded ${type === 'both' ? 'both files' : filename}`);
            
        } catch (error) {
            console.error('Error downloading file:', error);
            this.showError(`Failed to download file: ${error.message}`);
        }
    }

    /**
     * Download buffer as file
     */
    downloadBuffer(buffer, filename) {
        const blob = new Blob([buffer], { type: 'application/octet-stream' });
        const url = URL.createObjectURL(blob);
        
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        
        URL.revokeObjectURL(url);
    }

    /**
     * Show status message
     */
    showStatus(message) {
        this.ui.uploadStatus.textContent = message;
        this.ui.uploadStatus.style.display = 'block';
    }

    /**
     * Show error message
     */
    showError(message) {
        this.ui.validationErrors.textContent = message;
        this.ui.validationErrors.classList.add('show');
        
        setTimeout(() => {
            this.ui.validationErrors.classList.remove('show');
        }, 5000);
    }

    /**
     * Read file as ArrayBuffer
     */
    readFileAsArrayBuffer(file) {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = (e) => resolve(e.target.result);
            reader.onerror = (e) => reject(new Error('Failed to read file'));
            reader.readAsArrayBuffer(file);
        });
    }

    /**
     * Initialize the application
     */
    init() {
        console.log('Pocket Library Toolkit initialized');
        this.showStatus('Ready to load list.bin and playtimes.bin files');
    }
}

// Initialize application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    const app = new PocketLibraryApp();
    app.init();
});
