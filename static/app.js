class ChitChat {
    constructor() {
        this.ws = null;
        this.user = null;
        this.token = null;
        this.activeChat = null;
        this.chats = new Map();
        this.users = new Map();
        this.contacts = new Map();
        this.isTyping = false;
        this.typingTimeout = null;
        
        this.init();
    }

    init() {
        this.bindEvents();
        this.checkAuth();
    }

    bindEvents() {
        // Auth screen
        document.getElementById('login-btn').addEventListener('click', () => this.handleLogin());
        document.getElementById('phone').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.handleLogin();
        });
        document.getElementById('name').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.handleLogin();
        });

        // Chat screen
        document.getElementById('start-chat-btn').addEventListener('click', () => this.showNewChatModal());
        document.getElementById('new-chat-btn').addEventListener('click', () => this.showNewChatModal());
        document.getElementById('send-btn').addEventListener('click', () => this.sendMessage());
        document.getElementById('message-input').addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.sendMessage();
            } else {
                this.handleTyping();
            }
        });
        
        // Search
        document.getElementById('chat-search').addEventListener('input', (e) => this.searchChats(e.target.value));
        document.getElementById('search-user').addEventListener('input', (e) => this.searchUsers(e.target.value));

        // Tabs - FIXED
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                this.switchTab(e.currentTarget.dataset.tab);
            });
        });

        // Modals
        document.querySelector('.close-modal').addEventListener('click', () => this.hideModal());
        document.querySelector('.close-panel').addEventListener('click', () => this.hideRightPanel());

        // Chat list clicks
        document.getElementById('chat-list').addEventListener('click', (e) => {
            const chatItem = e.target.closest('.chat-item');
            if (chatItem && chatItem.dataset.chatId) {
                this.selectChat(chatItem.dataset.chatId);
            }
        });

        // User list clicks (in modal) - Add contact functionality
        document.getElementById('user-list').addEventListener('click', (e) => {
            const userItem = e.target.closest('.user-item');
            if (userItem && userItem.dataset.userId) {
                if (e.target.classList.contains('add-contact-btn')) {
                    this.addContact(userItem.dataset.userId);
                } else if (e.target.classList.contains('start-chat-btn')) {
                    this.startChatWithUser(userItem.dataset.userId);
                } else {
                    this.showUserActions(userItem.dataset.userId);
                }
            }
        });

        // Menu button - FIXED
        document.getElementById('menu-btn').addEventListener('click', (e) => this.showMenu(e.currentTarget));
        
        // Voice and video call buttons (placeholder)
        document.querySelectorAll('.chat-header-actions .icon-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const icon = e.currentTarget.querySelector('i').className;
                if (icon.includes('phone')) {
                    this.showToast('Voice call feature coming soon!');
                } else if (icon.includes('video')) {
                    this.showToast('Video call feature coming soon!');
                } else if (icon.includes('ellipsis')) {
                    this.showChatOptions();
                }
            });
        });

        // Input action buttons
        document.querySelectorAll('.input-actions .icon-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const icon = e.currentTarget.querySelector('i').className;
                if (icon.includes('paperclip')) {
                    this.showToast('File attachment coming soon!');
                } else if (icon.includes('smile')) {
                    this.showToast('Emoji picker coming soon!');
                }
            });
        });
    }

    async handleLogin() {
        const phone = document.getElementById('phone').value.trim();
        const name = document.getElementById('name').value.trim();

        if (!phone || phone.length !== 10) {
            this.showError('Please enter a valid 10-digit phone number');
            return;
        }

        if (!name) {
            this.showError('Please enter your name');
            return;
        }

        try {
            const response = await fetch('/api/auth/register', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ phone, name })
            });

            if (!response.ok) throw new Error('Login failed');

            const data = await response.json();
            this.token = data.token;
            this.user = data.user;
            
            localStorage.setItem('chitchat_token', this.token);
            localStorage.setItem('chitchat_user', JSON.stringify(this.user));
            
            this.showChatScreen();
            this.connectWebSocket();
            this.loadChats();
            this.loadContacts();
        } catch (error) {
            console.error('Login error:', error);
            this.showError('Login failed. Please try again.');
        }
    }

    checkAuth() {
        const token = localStorage.getItem('chitchat_token');
        const userStr = localStorage.getItem('chitchat_user');
        
        if (token && userStr) {
            this.token = token;
            this.user = JSON.parse(userStr);
            this.showChatScreen();
            this.connectWebSocket();
            this.loadChats();
            this.loadContacts();
        }
    }

    showChatScreen() {
        document.getElementById('auth-screen').classList.add('hidden');
        document.getElementById('chat-screen').classList.remove('hidden');
        
        if (this.user) {
            document.getElementById('user-name').textContent = this.user.name;
            document.getElementById('user-status').textContent = 'Online';
            document.getElementById('user-avatar').innerHTML = `<span>${this.user.name.charAt(0).toUpperCase()}</span>`;
        }
    }

    showNewChatModal() {
        document.getElementById('new-chat-modal').classList.remove('hidden');
        this.loadUsers();
    }

    hideModal() {
        document.getElementById('new-chat-modal').classList.add('hidden');
        document.getElementById('search-user').value = '';
    }

    connectWebSocket() {
        if (!this.token) return;

        const wsUrl = `ws://${window.location.host}/ws?token=${this.token}`;
        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.updateUserStatus(true);
        };

        this.ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                this.handleWebSocketMessage(data);
            } catch (error) {
                console.error('Error parsing WebSocket message:', error);
            }
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            this.updateUserStatus(false);
            
            // Try to reconnect after 5 seconds
            setTimeout(() => this.connectWebSocket(), 5000);
        };
    }

    handleWebSocketMessage(data) {
        switch (data.type) {
            case 'message':
                this.handleIncomingMessage(data.payload);
                break;
            case 'typing':
                this.handleTypingIndicator(data.payload);
                break;
            case 'presence':
                this.handlePresenceUpdate(data.payload);
                break;
            case 'status_update':
                this.handleStatusUpdate(data.payload);
                break;
            case 'chat_update':
                this.handleChatUpdate(data.payload);
                break;
        }
    }

    handleIncomingMessage(message) {
        // Add message to chat
        const chatId = message.chat_id;
        if (!this.chats.has(chatId)) {
            this.loadChat(chatId);
        }
        
        if (this.activeChat === chatId) {
            this.addMessageToUI(message, false);
            this.scrollToBottom();
            this.markMessageAsRead(message.id);
        } else {
            this.updateChatUnreadCount(chatId);
        }
        
        this.updateChatLastMessage(chatId, message);
    }

    async loadChats() {
        try {
            const response = await fetch('/api/chats', {
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
            
            if (!response.ok) throw new Error('Failed to load chats');
            
            const data = await response.json();
            this.renderChatList(data.chats);
            
            data.chats.forEach(chat => {
                this.chats.set(chat.id, chat);
            });
        } catch (error) {
            console.error('Error loading chats:', error);
            this.showError('Failed to load chats');
        }
    }

    async loadUsers() {
        try {
            const response = await fetch('/api/users/search?q=', {
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
            
            if (!response.ok) throw new Error('Failed to load users');
            
            const users = await response.json();
            this.renderUserList(users);
            
            users.forEach(user => {
                this.users.set(user.id, user);
            });
        } catch (error) {
            console.error('Error loading users:', error);
        }
    }

    async searchUsers(query) {
        try {
            const response = await fetch(`/api/users/search?q=${encodeURIComponent(query)}`, {
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
            
            if (!response.ok) throw new Error('Search failed');
            
            const users = await response.json();
            this.renderUserList(users);
        } catch (error) {
            console.error('Search error:', error);
        }
    }

    renderUserList(users) {
        const container = document.getElementById('user-list');
        container.innerHTML = '';

        if (users.length === 0) {
            container.innerHTML = '<p class="no-users">No users found</p>';
            return;
        }

        users.forEach(user => {
            if (user.id === this.user.id) return;
            
            const isContact = this.contacts.has(user.id);
            const element = this.createUserElement(user, isContact);
            container.appendChild(element);
        });
    }

    createUserElement(user, isContact) {
        const div = document.createElement('div');
        div.className = 'user-item';
        div.dataset.userId = user.id;
        
        div.innerHTML = `
            <div class="user-avatar">
                <span>${user.name.charAt(0).toUpperCase()}</span>
            </div>
            <div class="user-info">
                <h4>${user.name}</h4>
                <p>${user.phone} • ${user.is_online ? 'Online' : 'Offline'}</p>
            </div>
            <div class="user-actions">
                ${isContact ? 
                    `<button class="btn-secondary start-chat-btn">Chat</button>
                     <button class="btn-text remove-contact-btn">Remove</button>` :
                    `<button class="btn-primary add-contact-btn">Add</button>`
                }
            </div>
        `;

        return div;
    }

    async addContact(userId) {
        try {
            const response = await fetch('/api/contacts', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.token}`
                },
                body: JSON.stringify({ user_id: userId })
            });
            
            if (!response.ok) throw new Error('Failed to add contact');
            
            this.showToast('Contact added successfully');
            this.loadContacts();
            
            // Update user list
            const user = this.users.get(userId);
            if (user) {
                this.contacts.set(userId, user);
                this.renderUserList(Array.from(this.users.values()));
            }
        } catch (error) {
            console.error('Error adding contact:', error);
            this.showError('Failed to add contact');
        }
    }

    async removeContact(userId) {
        try {
            const response = await fetch(`/api/contacts/${userId}`, {
                method: 'DELETE',
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
            
            if (!response.ok) throw new Error('Failed to remove contact');
            
            this.showToast('Contact removed');
            this.contacts.delete(userId);
            this.loadContacts();
            
            // Update user list
            this.renderUserList(Array.from(this.users.values()));
        } catch (error) {
            console.error('Error removing contact:', error);
            this.showError('Failed to remove contact');
        }
    }

    async loadContacts() {
        try {
            const response = await fetch('/api/contacts', {
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
            
            if (!response.ok) throw new Error('Failed to load contacts');
            
            const contacts = await response.json();
            this.contacts.clear();
            
            contacts.forEach(user => {
                this.contacts.set(user.id, user);
            });
            
            // If contacts tab is active, render contacts
            const activeTab = document.querySelector('.tab-btn.active').dataset.tab;
            if (activeTab === 'contacts') {
                this.renderContactsList();
            }
        } catch (error) {
            console.error('Error loading contacts:', error);
        }
    }

    renderContactsList() {
        const container = document.getElementById('chat-list');
        container.innerHTML = '';
        
        if (this.contacts.size === 0) {
            container.innerHTML = `
                <div class="empty-chat-list">
                    <p>No contacts yet</p>
                    <button class="btn-secondary" id="add-contact-btn">Add Contacts</button>
                </div>
            `;
            
            document.getElementById('add-contact-btn').addEventListener('click', () => {
                this.showNewChatModal();
            });
            return;
        }
        
        Array.from(this.contacts.values()).forEach(user => {
            const element = this.createContactElement(user);
            container.appendChild(element);
        });
    }

    createContactElement(user) {
        const div = document.createElement('div');
        div.className = 'chat-item';
        div.dataset.userId = user.id;
        
        div.innerHTML = `
            <div class="chat-avatar">
                <span>${user.name.charAt(0).toUpperCase()}</span>
            </div>
            <div class="chat-info">
                <div class="chat-name">${user.name}</div>
                <div class="last-message">${user.phone}</div>
                <div class="chat-meta">
                    <span class="timestamp">${user.is_online ? 'Online' : 'Offline'}</span>
                </div>
            </div>
        `;
        
        div.addEventListener('click', () => {
            this.startChatWithUser(user.id);
        });
        
        return div;
    }

    // FIXED: Tab switching functionality
    switchTab(tab) {
        // Update active tab button
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.tab === tab);
        });

        // Clear active chat if switching away from chats
        if (tab !== 'chats') {
            this.activeChat = null;
            document.getElementById('empty-chat').classList.remove('hidden');
            document.getElementById('active-chat').classList.add('hidden');
        }

        // Load content for the tab
        switch (tab) {
            case 'chats':
                this.loadChats();
                document.querySelector('.chat-list').style.display = 'block';
                break;
            case 'groups':
                this.loadGroups();
                break;
            case 'contacts':
                this.loadContacts();
                this.renderContactsList();
                break;
            case 'settings':
                this.loadSettings();
                break;
        }
    }

    // Menu functionality
    showMenu(button) {
        // Create menu if it doesn't exist
        let menu = document.getElementById('user-menu');
        if (!menu) {
            menu = document.createElement('div');
            menu.id = 'user-menu';
            menu.className = 'dropdown-menu';
            menu.innerHTML = `
                <ul>
                    <li><a href="#" id="profile-btn"><i class="fas fa-user"></i> Profile</a></li>
                    <li><a href="#" id="settings-btn"><i class="fas fa-cog"></i> Settings</a></li>
                    <li><hr></li>
                    <li><a href="#" id="logout-btn" class="logout"><i class="fas fa-sign-out-alt"></i> Logout</a></li>
                </ul>
            `;
            document.body.appendChild(menu);
            
            // Add event listeners for menu items
            document.getElementById('logout-btn').addEventListener('click', (e) => {
                e.preventDefault();
                this.handleLogout();
                this.hideMenu();
            });
            
            document.getElementById('profile-btn').addEventListener('click', (e) => {
                e.preventDefault();
                this.showProfile();
                this.hideMenu();
            });
            
            document.getElementById('settings-btn').addEventListener('click', (e) => {
                e.preventDefault();
                this.switchTab('settings');
                this.hideMenu();
            });
        }
        
        // Position and show menu
        const rect = button.getBoundingClientRect();
        menu.style.top = `${rect.bottom + 5}px`;
        menu.style.left = `${rect.left}px`;
        menu.classList.add('show');
        
        // Close menu when clicking outside
        const clickHandler = (e) => {
            if (!menu.contains(e.target) && e.target !== button) {
                this.hideMenu();
                document.removeEventListener('click', clickHandler);
            }
        };
        
        setTimeout(() => {
            document.addEventListener('click', clickHandler);
        }, 100);
    }

    hideMenu() {
        const menu = document.getElementById('user-menu');
        if (menu) {
            menu.classList.remove('show');
        }
    }

    async handleLogout() {
        try {
            await fetch('/api/auth/logout', {
                method: 'POST',
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
        } catch (error) {
            console.error('Logout error:', error);
        }
        
        // Clear local storage
        localStorage.removeItem('chitchat_token');
        localStorage.removeItem('chitchat_user');
        
        // Reset state
        this.token = null;
        this.user = null;
        this.ws?.close();
        this.ws = null;
        
        // Show auth screen
        document.getElementById('chat-screen').classList.add('hidden');
        document.getElementById('auth-screen').classList.remove('hidden');
        
        // Clear form
        document.getElementById('phone').value = '';
        document.getElementById('name').value = '';
        
        this.showToast('Logged out successfully');
    }

    showProfile() {
        if (!this.user) return;
        
        // Create profile modal
        const modal = document.createElement('div');
        modal.className = 'modal';
        modal.innerHTML = `
            <div class="modal-content profile-modal">
                <div class="modal-header">
                    <h3>My Profile</h3>
                    <button class="close-modal">&times;</button>
                </div>
                <div class="modal-body">
                    <div class="profile-header">
                        <div class="profile-avatar large">
                            <span>${this.user.name.charAt(0).toUpperCase()}</span>
                        </div>
                        <h2>${this.user.name}</h2>
                        <p>${this.user.phone}</p>
                    </div>
                    <div class="profile-info">
                        <div class="info-item">
                            <label>Status</label>
                            <p>${this.user.status || 'Hey there! I am using ChitChat'}</p>
                        </div>
                        <div class="info-item">
                            <label>Last Seen</label>
                            <p>${this.formatTime(this.user.last_seen)}</p>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        document.body.appendChild(modal);
        modal.classList.remove('hidden');
        
        // Close modal
        modal.querySelector('.close-modal').addEventListener('click', () => {
            modal.remove();
        });
        
        // Close on background click
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.remove();
            }
        });
    }

    loadGroups() {
        const container = document.getElementById('chat-list');
        container.innerHTML = `
            <div class="empty-chat-list">
                <p>Groups feature coming soon!</p>
                <button class="btn-secondary" onclick="chitchat.showToast('Create group feature coming soon!')">
                    Create Group
                </button>
            </div>
        `;
    }

    loadSettings() {
        const container = document.getElementById('chat-list');
        container.innerHTML = `
            <div class="settings-container">
                <h3>Settings</h3>
                <div class="settings-item">
                    <label>Notifications</label>
                    <label class="switch">
                        <input type="checkbox" checked>
                        <span class="slider"></span>
                    </label>
                </div>
                <div class="settings-item">
                    <label>Dark Mode</label>
                    <label class="switch">
                        <input type="checkbox">
                        <span class="slider"></span>
                    </label>
                </div>
                <div class="settings-item">
                    <label>Privacy</label>
                    <button class="btn-text">View Privacy Settings</button>
                </div>
                <div class="settings-item">
                    <label>About</label>
                    <button class="btn-text">App Version 1.0.0</button>
                </div>
            </div>
        `;
    }

    showChatOptions() {
        const menu = document.createElement('div');
        menu.className = 'dropdown-menu';
        menu.innerHTML = `
            <ul>
                <li><a href="#"><i class="fas fa-users"></i> View Members</a></li>
                <li><a href="#"><i class="fas fa-bell"></i> Mute Chat</a></li>
                <li><a href="#"><i class="fas fa-archive"></i> Archive Chat</a></li>
                <li><hr></li>
                <li><a href="#" class="danger"><i class="fas fa-trash"></i> Delete Chat</a></li>
            </ul>
        `;
        
        document.body.appendChild(menu);
        
        // Position near the menu button
        const button = document.querySelector('.chat-header-actions .icon-btn:last-child');
        const rect = button.getBoundingClientRect();
        menu.style.top = `${rect.bottom + 5}px`;
        menu.style.right = `${window.innerWidth - rect.right}px`;
        menu.classList.add('show');
        
        // Close menu when clicking outside
        setTimeout(() => {
            const clickHandler = (e) => {
                if (!menu.contains(e.target) && e.target !== button) {
                    menu.remove();
                    document.removeEventListener('click', clickHandler);
                }
            };
            document.addEventListener('click', clickHandler);
        }, 100);
    }

    showToast(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;
        toast.textContent = message;
        toast.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: ${type === 'error' ? '#f44336' : '#4CAF50'};
            color: white;
            padding: 12px 24px;
            border-radius: 8px;
            z-index: 1000;
            animation: slideIn 0.3s ease;
            max-width: 300px;
            word-wrap: break-word;
        `;
        
        document.body.appendChild(toast);
        
        setTimeout(() => {
            toast.style.animation = 'slideOut 0.3s ease';
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    }

    // Other methods remain the same as your original code...
    // (sendMessage, handleTyping, selectChat, renderChatList, etc.)
    // Include all the other methods from your original app.js

    // Add this helper method for error display
    showError(message) {
        this.showToast(message, 'error');
    }
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.chitchat = new ChitChat();
});

// Add CSS for dropdown menu and other UI elements
const style = document.createElement('style');
style.textContent = `
    .dropdown-menu {
        position: absolute;
        background: white;
        border-radius: 8px;
        box-shadow: 0 4px 20px rgba(0,0,0,0.15);
        padding: 8px 0;
        min-width: 200px;
        z-index: 1000;
        display: none;
        border: 1px solid #e0e0e0;
    }
    
    .dropdown-menu.show {
        display: block;
    }
    
    .dropdown-menu ul {
        list-style: none;
        margin: 0;
        padding: 0;
    }
    
    .dropdown-menu li {
        margin: 0;
    }
    
    .dropdown-menu a {
        display: flex;
        align-items: center;
        padding: 12px 16px;
        color: #333;
        text-decoration: none;
        transition: background 0.2s;
    }
    
    .dropdown-menu a:hover {
        background: #f5f5f5;
    }
    
    .dropdown-menu a i {
        margin-right: 10px;
        width: 20px;
        text-align: center;
    }
    
    .dropdown-menu hr {
        margin: 8px 0;
        border: none;
        border-top: 1px solid #e0e0e0;
    }
    
    .dropdown-menu .logout {
        color: #f44336;
    }
    
    .dropdown-menu .danger {
        color: #f44336;
    }
    
    .user-actions {
        display: flex;
        gap: 8px;
        margin-left: auto;
    }
    
    .user-actions button {
        padding: 6px 12px;
        font-size: 14px;
    }
    
    .btn-text {
        background: none;
        border: none;
        color: #667eea;
        cursor: pointer;
        padding: 6px 12px;
        font-size: 14px;
    }
    
    .btn-text:hover {
        text-decoration: underline;
    }
    
    .remove-contact-btn {
        color: #f44336;
    }
    
    .settings-container {
        padding: 20px;
    }
    
    .settings-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 16px 0;
        border-bottom: 1px solid #e0e0e0;
    }
    
    .settings-item:last-child {
        border-bottom: none;
    }
    
    .switch {
        position: relative;
        display: inline-block;
        width: 50px;
        height: 24px;
    }
    
    .switch input {
        opacity: 0;
        width: 0;
        height: 0;
    }
    
    .slider {
        position: absolute;
        cursor: pointer;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background-color: #ccc;
        transition: .4s;
        border-radius: 24px;
    }
    
    .slider:before {
        position: absolute;
        content: "";
        height: 16px;
        width: 16px;
        left: 4px;
        bottom: 4px;
        background-color: white;
        transition: .4s;
        border-radius: 50%;
    }
    
    input:checked + .slider {
        background-color: #667eea;
    }
    
    input:checked + .slider:before {
        transform: translateX(26px);
    }
    
    .profile-modal .modal-content {
        max-width: 400px;
    }
    
    .profile-header {
        text-align: center;
        padding: 20px 0;
    }
    
    .profile-avatar.large {
        width: 80px;
        height: 80px;
        font-size: 32px;
        margin: 0 auto 16px;
    }
    
    .profile-info {
        margin-top: 20px;
    }
    
    .info-item {
        margin-bottom: 16px;
    }
    
    .info-item label {
        font-weight: bold;
        color: #666;
        display: block;
        margin-bottom: 4px;
    }
    
    @keyframes slideIn {
        from { transform: translateX(100%); opacity: 0; }
        to { transform: translateX(0); opacity: 1; }
    }
    
    @keyframes slideOut {
        from { transform: translateX(0); opacity: 1; }
        to { transform: translateX(100%); opacity: 0; }
    }
    
    .toast {
        font-family: inherit;
        box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    }
`;
document.head.appendChild(style);
