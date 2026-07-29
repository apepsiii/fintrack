// IndexedDB manager for offline storage
class OfflineDB {
    constructor() {
        this.dbName = 'fintrack-offline';
        this.version = 1;
        this.db = null;
    }

    // Initialize IndexedDB
    async init() {
        return new Promise((resolve, reject) => {
            const request = indexedDB.open(this.dbName, this.version);

            request.onerror = () => reject(request.error);
            request.onsuccess = () => {
                this.db = request.result;
                resolve(this.db);
            };

            request.onupgradeneeded = (event) => {
                const db = event.target.result;

                // Create object stores
                if (!db.objectStoreNames.contains('transactions')) {
                    const transactionStore = db.createObjectStore('transactions', { 
                        keyPath: 'id', 
                        autoIncrement: true 
                    });
                    transactionStore.createIndex('synced', 'synced', { unique: false });
                    transactionStore.createIndex('date', 'date', { unique: false });
                }

                if (!db.objectStoreNames.contains('cache')) {
                    db.createObjectStore('cache', { keyPath: 'key' });
                }

                if (!db.objectStoreNames.contains('settings')) {
                    db.createObjectStore('settings', { keyPath: 'key' });
                }

                console.log('[OfflineDB] Database initialized');
            };
        });
    }

    // Add offline transaction
    async addTransaction(transaction) {
        return new Promise((resolve, reject) => {
            const tx = this.db.transaction(['transactions'], 'readwrite');
            const store = tx.objectStore('transactions');
            
            const data = {
                ...transaction,
                synced: false,
                createdAt: new Date().toISOString()
            };

            const request = store.add(data);
            
            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    }

    // Get all unsynced transactions
    async getUnsyncedTransactions() {
        return new Promise((resolve, reject) => {
            const tx = this.db.transaction(['transactions'], 'readonly');
            const store = tx.objectStore('transactions');
            const index = store.index('synced');
            const request = index.getAll(false);

            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    }

    // Mark transaction as synced
    async markAsSynced(id) {
        return new Promise((resolve, reject) => {
            const tx = this.db.transaction(['transactions'], 'readwrite');
            const store = tx.objectStore('transactions');
            const request = store.get(id);

            request.onsuccess = () => {
                const data = request.result;
                if (data) {
                    data.synced = true;
                    data.syncedAt = new Date().toISOString();
                    const updateRequest = store.put(data);
                    updateRequest.onsuccess = () => resolve();
                    updateRequest.onerror = () => reject(updateRequest.error);
                } else {
                    resolve();
                }
            };
            request.onerror = () => reject(request.error);
        });
    }

    // Delete synced transaction
    async deleteSyncedTransaction(id) {
        return new Promise((resolve, reject) => {
            const tx = this.db.transaction(['transactions'], 'readwrite');
            const store = tx.objectStore('transactions');
            const request = store.delete(id);

            request.onsuccess = () => resolve();
            request.onerror = () => reject(request.error);
        });
    }

    // Cache data
    async cacheData(key, value) {
        return new Promise((resolve, reject) => {
            const tx = this.db.transaction(['cache'], 'readwrite');
            const store = tx.objectStore('cache');
            const request = store.put({ 
                key, 
                value, 
                timestamp: Date.now() 
            });

            request.onsuccess = () => resolve();
            request.onerror = () => reject(request.error);
        });
    }

    // Get cached data
    async getCachedData(key, maxAge = 3600000) { // default 1 hour
        return new Promise((resolve, reject) => {
            const tx = this.db.transaction(['cache'], 'readonly');
            const store = tx.objectStore('cache');
            const request = store.get(key);

            request.onsuccess = () => {
                const data = request.result;
                if (!data) {
                    resolve(null);
                    return;
                }

                // Check if cache is still valid
                if (Date.now() - data.timestamp < maxAge) {
                    resolve(data.value);
                } else {
                    resolve(null);
                }
            };
            request.onerror = () => reject(request.error);
        });
    }

    // Save setting
    async saveSetting(key, value) {
        return new Promise((resolve, reject) => {
            const tx = this.db.transaction(['settings'], 'readwrite');
            const store = tx.objectStore('settings');
            const request = store.put({ key, value });

            request.onsuccess = () => resolve();
            request.onerror = () => reject(request.error);
        });
    }

    // Get setting
    async getSetting(key, defaultValue = null) {
        return new Promise((resolve, reject) => {
            const tx = this.db.transaction(['settings'], 'readonly');
            const store = tx.objectStore('settings');
            const request = store.get(key);

            request.onsuccess = () => {
                const data = request.result;
                resolve(data ? data.value : defaultValue);
            };
            request.onerror = () => reject(request.error);
        });
    }

    // Get all transactions (for display)
    async getAllTransactions() {
        return new Promise((resolve, reject) => {
            const tx = this.db.transaction(['transactions'], 'readonly');
            const store = tx.objectStore('transactions');
            const request = store.getAll();

            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    }

    // Clear all data
    async clearAll() {
        return new Promise((resolve, reject) => {
            const tx = this.db.transaction(['transactions', 'cache', 'settings'], 'readwrite');
            
            tx.objectStore('transactions').clear();
            tx.objectStore('cache').clear();
            tx.objectStore('settings').clear();

            tx.oncomplete = () => resolve();
            tx.onerror = () => reject(tx.error);
        });
    }
}

// Offline sync manager
class OfflineSync {
    constructor(offlineDB) {
        this.offlineDB = offlineDB;
        this.syncing = false;
    }

    // Check if online
    isOnline() {
        return navigator.onLine;
    }

    // Sync all pending transactions
    async syncTransactions() {
        if (this.syncing || !this.isOnline()) {
            return;
        }

        this.syncing = true;
        console.log('[OfflineSync] Starting sync...');

        try {
            const transactions = await this.offlineDB.getUnsyncedTransactions();
            
            if (transactions.length === 0) {
                console.log('[OfflineSync] No transactions to sync');
                this.syncing = false;
                return;
            }

            let synced = 0;
            let failed = 0;

            for (const transaction of transactions) {
                try {
                    // Send to server
                    const formData = new FormData();
                    formData.append('type', transaction.type);
                    formData.append('amount', transaction.amount);
                    formData.append('category_id', transaction.category_id);
                    if (transaction.note) {
                        formData.append('note', transaction.note);
                    }

                    const response = await fetch('/transaction', {
                        method: 'POST',
                        body: formData,
                        credentials: 'include'
                    });

                    if (response.ok) {
                        // Mark as synced and delete from local
                        await this.offlineDB.deleteSyncedTransaction(transaction.id);
                        synced++;
                        console.log('[OfflineSync] Synced transaction:', transaction.id);
                    } else {
                        failed++;
                        console.error('[OfflineSync] Failed to sync transaction:', transaction.id);
                    }
                } catch (error) {
                    failed++;
                    console.error('[OfflineSync] Error syncing transaction:', error);
                }
            }

            console.log(`[OfflineSync] Sync complete: ${synced} synced, ${failed} failed`);

            // Show notification to user
            if (synced > 0) {
                this.showSyncNotification(synced);
            }

        } catch (error) {
            console.error('[OfflineSync] Sync failed:', error);
        } finally {
            this.syncing = false;
        }
    }

    // Show sync notification
    showSyncNotification(count) {
        // You can replace this with a toast notification
        console.log(`✅ ${count} transaksi offline telah disinkronkan`);
        
        // Trigger page reload to show synced data
        if (count > 0) {
            setTimeout(() => {
                window.location.reload();
            }, 1000);
        }
    }

    // Listen for online event
    startListening() {
        window.addEventListener('online', () => {
            console.log('[OfflineSync] Back online, syncing...');
            this.syncTransactions();
        });

        // Try to sync on page load if online
        if (this.isOnline()) {
            this.syncTransactions();
        }
    }
}

// Initialize offline functionality
let offlineDB = null;
let offlineSync = null;

async function initOfflineMode() {
    try {
        offlineDB = new OfflineDB();
        await offlineDB.init();
        
        offlineSync = new OfflineSync(offlineDB);
        offlineSync.startListening();
        
        console.log('[FinTrack] Offline mode initialized');
    } catch (error) {
        console.error('[FinTrack] Failed to initialize offline mode:', error);
    }
}

// Call init when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initOfflineMode);
} else {
    initOfflineMode();
}
