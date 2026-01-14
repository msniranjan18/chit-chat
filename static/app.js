class ChitChat {
    constructor() {
        this.ws = null;
        this.user = null;
        this.token = null;
        this.activeChat = null;
        this.chats = new Map();
        this.users = new Map();
        this.isTyping = false;
        this.typingTimeout = null;
        
        this.init();
    }

    init() {
        this.bindEvents();
        this.checkAuth();
        this.setupWebSocket();
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

        // Tabs
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.addEventListener('click', (e) => this.switchTab(e.target.dataset.tab));
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

        // User list clicks (in modal)
        document.getElementById('user-list').addEventListener('click', (e) => {
            const userItem = e.target.closest('.user-item');
            if (userItem && userItem.dataset.userId) {
                this.startChatWithUser(userItem.dataset.userId);
            }
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
        } catch (error) {
            this.showError('Login failed. Please try again.');
            console.error('Login error:', error);
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

    setupWebSocket() {
        // WebSocket setup is done in connectWebSocket
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
            const data = JSON.parse(event.data);
            this.handleWebSocketMessage(data);
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
            // If it's a new chat, load it
            this.loadChat(chatId);
        }
        
        // Add message to active chat if it's the current one
        if (this.activeChat === chatId) {
            this.addMessageToUI(message, false);
            this.scrollToBottom();
            
            // Mark as read
            this.markMessageAsRead(message.id);
        } else {
            // Update chat list with unread count
            this.updateChatUnreadCount(chatId);
        }
        
        // Update last message in chat list
        this.updateChatLastMessage(chatId, message);
    }

    handleTypingIndicator(data) {
        if (data.chat_id === this.activeChat && data.user_id !== this.user.id) {
            this.showTypingIndicator(data.is_typing);
        }
    }

    handlePresenceUpdate(data) {
        // Update user status in UI
        const user = this.users.get(data.user_id);
        if (user) {
            user.is_online = data.status === 'online';
            user.last_seen = new Date(data.last_seen);
            this.updateUserStatusInUI(data.user_id, data.status);
        }
    }

    handleStatusUpdate(data) {
        // Update message status (delivered/read)
        this.updateMessageStatus(data.message_id, data.status);
    }

    handleChatUpdate(data) {
        // Handle chat updates (new members, name changes, etc.)
        this.loadChat(data.chat_id);
    }

    async loadChats() {
        try {
            const response = await fetch('/api/chats', {
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
            
            if (!response.ok) throw new Error('Failed to load chats');
            
            const data = await response.json();
            this.renderChatList(data.chats);
            
            // Store chats in map for quick access
            data.chats.forEach(chat => {
                this.chats.set(chat.id, chat);
            });
        } catch (error) {
            console.error('Error loading chats:', error);
        }
    }

    async loadChat(chatId) {
        try {
            const response = await fetch(`/api/chats/${chatId}`, {
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
            
            if (!response.ok) throw new Error('Failed to load chat');
            
            const data = await response.json();
            this.chats.set(chatId, data.chat);
            
            // Load messages for this chat
            this.loadMessages(chatId);
            
            // Update chat in list
            this.updateChatInList(data.chat);
        } catch (error) {
            console.error('Error loading chat:', error);
        }
    }

    async loadMessages(chatId, offset = 0, limit = 50) {
        try {
            const response = await fetch(`/api/messages?chat_id=${chatId}&offset=${offset}&limit=${limit}`, {
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
            
            if (!response.ok) throw new Error('Failed to load messages');
            
            const data = await response.json();
            this.renderMessages(data.messages);
            
            // Mark all messages as read
            this.markChatAsRead(chatId);
        } catch (error) {
            console.error('Error loading messages:', error);
        }
    }

    async loadUsers() {
        try {
            const response = await fetch('/api/users', {
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
            
            if (!response.ok) throw new Error('Failed to load users');
            
            const users = await response.json();
            this.renderUserList(users);
            
            // Store users in map
            users.forEach(user => {
                this.users.set(user.id, user);
            });
        } catch (error) {
            console.error('Error loading users:', error);
        }
    }

    async searchUsers(query) {
        if (!query.trim()) {
            this.loadUsers();
            return;
        }

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

    async startChatWithUser(userId) {
        try {
            const response = await fetch('/api/chats', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.token}`
                },
                body: JSON.stringify({
                    type: 'direct',
                    user_ids: [userId]
                })
            });
            
            if (!response.ok) throw new Error('Failed to start chat');
            
            const data = await response.json();
            this.hideModal();
            this.selectChat(data.chat.id);
        } catch (error) {
            console.error('Error starting chat:', error);
            this.showError('Failed to start chat');
        }
    }

    async sendMessage() {
        const input = document.getElementById('message-input');
        const content = input.value.trim();
        
        if (!content || !this.activeChat) return;

        // Clear input
        input.value = '';
        
        // Create message object
        const message = {
            chat_id: this.activeChat,
            content: content,
            content_type: 'text',
            sent_at: new Date().toISOString(),
            status: 'sending'
        };

        // Add to UI immediately (optimistic update)
        this.addMessageToUI(message, true);
        this.scrollToBottom();

        try {
            // Send via WebSocket
            this.ws.send(JSON.stringify({
                type: 'message',
                room_id: this.activeChat,
                sender: this.user.id,
                payload: {
                    chat_id: this.activeChat,
                    content: content,
                    content_type: 'text'
                }
            }));

            // Also send via REST API for persistence
            const response = await fetch('/api/messages', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.token}`
                },
                body: JSON.stringify({
                    chat_id: this.activeChat,
                    content: content,
                    content_type: 'text'
                })
            });

            if (!response.ok) throw new Error('Failed to send message');

            const savedMessage = await response.json();
            
            // Update message with server ID and status
            this.updateMessageInUI(savedMessage.message.id, {
                id: savedMessage.message.id,
                status: 'sent'
            });

        } catch (error) {
            console.error('Error sending message:', error);
            
            // Update message status to failed
            this.updateMessageInUI('temp', {
                status: 'failed'
            });
            
            this.showError('Failed to send message');
        }
    }

    handleTyping() {
        if (!this.activeChat) return;

        if (!this.isTyping) {
            this.isTyping = true;
            
            // Send typing start
            this.ws.send(JSON.stringify({
                type: 'typing',
                room_id: this.activeChat,
                sender: this.user.id,
                payload: {
                    chat_id: this.activeChat,
                    user_id: this.user.id,
                    is_typing: true
                }
            }));
        }

        // Clear existing timeout
        if (this.typingTimeout) {
            clearTimeout(this.typingTimeout);
        }

        // Set timeout to send typing stop after 2 seconds of inactivity
        this.typingTimeout = setTimeout(() => {
            this.isTyping = false;
            
            this.ws.send(JSON.stringify({
                type: 'typing',
                room_id: this.activeChat,
                sender: this.user.id,
                payload: {
                    chat_id: this.activeChat,
                    user_id: this.user.id,
                    is_typing: false
                }
            }));
        }, 2000);
    }

    async markMessageAsRead(messageId) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;

        this.ws.send(JSON.stringify({
            type: 'status_update',
            sender: this.user.id,
            payload: {
                message_id: messageId,
                status: 'read'
            }
        }));
    }

    async markChatAsRead(chatId) {
        try {
            await fetch(`/api/chats/${chatId}/read`, {
                method: 'POST',
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
        } catch (error) {
            console.error('Error marking chat as read:', error);
        }
    }

    selectChat(chatId) {
        this.activeChat = chatId;
        
        // Update UI
        document.querySelectorAll('.chat-item').forEach(item => {
            item.classList.remove('active');
            if (item.dataset.chatId === chatId) {
                item.classList.add('active');
            }
        });

        const chat = this.chats.get(chatId);
        if (chat) {
            this.showChatArea();
            this.updateChatHeader(chat);
            this.loadMessages(chatId);
            this.markChatAsRead(chatId);
        }
    }

    showChatArea() {
        document.getElementById('empty-chat').classList.add('hidden');
        document.getElementById('active-chat').classList.remove('hidden');
    }

    updateChatHeader(chat) {
        document.getElementById('chat-title').textContent = chat.name || 'Chat';
        document.getElementById('chat-status').textContent = this.getChatStatus(chat);
    }

    getChatStatus(chat) {
        if (chat.type === 'direct') {
            const otherUser = chat.members?.find(m => m.user_id !== this.user.id);
            if (otherUser) {
                const user = this.users.get(otherUser.user_id);
                return user?.is_online ? 'online' : 'last seen recently';
            }
        }
        return `${chat.members?.length || 0} members`;
    }

    renderChatList(chats) {
        const container = document.getElementById('chat-list');
        container.innerHTML = '';

        if (chats.length === 0) {
            container.innerHTML = `
                <div class="empty-chat-list">
                    <p>No chats yet</p>
                    <button class="btn-secondary" id="start-chat-btn">Start a chat</button>
                </div>
            `;
            return;
        }

        chats.forEach(chat => {
            const element = this.createChatElement(chat);
            container.appendChild(element);
        });
    }

    createChatElement(chat) {
        const div = document.createElement('div');
        div.className = 'chat-item';
        div.dataset.chatId = chat.id;

        const lastMessage = chat.last_message?.content || 'No messages yet';
        const time = chat.last_message?.sent_at ? this.formatTime(chat.last_message.sent_at) : '';
        
        div.innerHTML = `
            <div class="chat-avatar">
                <i class="fas fa-${chat.type === 'group' ? 'users' : 'user'}"></i>
            </div>
            <div class="chat-info">
                <div class="chat-name">${chat.name || 'Chat'}</div>
                <div class="last-message">${lastMessage}</div>
                <div class="chat-meta">
                    <span class="timestamp">${time}</span>
                    ${chat.unread_count > 0 ? `<span class="unread-count">${chat.unread_count}</span>` : ''}
                </div>
            </div>
        `;

        return div;
    }

    updateChatInList(chat) {
        const element = document.querySelector(`.chat-item[data-chat-id="${chat.id}"]`);
        if (element) {
            const lastMessage = chat.last_message?.content || 'No messages yet';
            const time = chat.last_message?.sent_at ? this.formatTime(chat.last_message.sent_at) : '';
            
            element.querySelector('.chat-name').textContent = chat.name || 'Chat';
            element.querySelector('.last-message').textContent = lastMessage;
            element.querySelector('.timestamp').textContent = time;
            
            const unreadCount = element.querySelector('.unread-count');
            if (chat.unread_count > 0) {
                if (!unreadCount) {
                    const span = document.createElement('span');
                    span.className = 'unread-count';
                    span.textContent = chat.unread_count;
                    element.querySelector('.chat-meta').appendChild(span);
                } else {
                    unreadCount.textContent = chat.unread_count;
                }
            } else if (unreadCount) {
                unreadCount.remove();
            }
        }
    }

    updateChatUnreadCount(chatId) {
        const element = document.querySelector(`.chat-item[data-chat-id="${chatId}"]`);
        if (element) {
            const unreadCount = element.querySelector('.unread-count');
            const currentCount = unreadCount ? parseInt(unreadCount.textContent) : 0;
            
            if (currentCount + 1 > 0) {
                if (!unreadCount) {
                    const span = document.createElement('span');
                    span.className = 'unread-count';
                    span.textContent = currentCount + 1;
                    element.querySelector('.chat-meta').appendChild(span);
                } else {
                    unreadCount.textContent = currentCount + 1;
                }
            }
        }
    }

    updateChatLastMessage(chatId, message) {
        const element = document.querySelector(`.chat-item[data-chat-id="${chatId}"]`);
        if (element) {
            element.querySelector('.last-message').textContent = message.content;
            element.querySelector('.timestamp').textContent = this.formatTime(message.sent_at);
            element.parentNode.prepend(element); // Move to top
        }
    }

    renderMessages(messages) {
        const container = document.getElementById('messages-container');
        container.innerHTML = '';
        
        messages.forEach(message => {
            this.addMessageToUI(message, message.sender_id === this.user.id);
        });
        
        this.scrollToBottom();
    }

    addMessageToUI(message, isSent) {
        const container = document.getElementById('messages-container');
        const element = this.createMessageElement(message, isSent);
        container.appendChild(element);
    }

    createMessageElement(message, isSent) {
        const div = document.createElement('div');
        div.className = `message ${isSent ? 'sent' : 'received'}`;
        div.dataset.messageId = message.id || 'temp';
        
        const time = this.formatTime(message.sent_at);
        const statusIcon = isSent ? this.getStatusIcon(message.status) : '';
        
        div.innerHTML = `
            <div class="message-content">${this.escapeHtml(message.content)}</div>
            <div class="message-time">
                ${time}
                ${statusIcon}
            </div>
        `;

        return div;
    }

    updateMessageInUI(messageId, updates) {
        const element = document.querySelector(`.message[data-message-id="${messageId}"]`);
        if (element) {
            if (updates.id) {
                element.dataset.messageId = updates.id;
            }
            
            if (updates.status) {
                const statusIcon = this.getStatusIcon(updates.status);
                element.querySelector('.message-time').innerHTML = 
                    element.querySelector('.message-time').textContent.replace(/[✓✓✓]/g, '') + statusIcon;
            }
        }
    }

    updateMessageStatus(messageId, status) {
        const element = document.querySelector(`.message[data-message-id="${messageId}"]`);
        if (element) {
            const statusIcon = this.getStatusIcon(status);
            const timeElement = element.querySelector('.message-time');
            timeElement.innerHTML = timeElement.textContent.replace(/[✓✓✓]/g, '') + statusIcon;
        }
    }

    getStatusIcon(status) {
        switch (status) {
            case 'sent': return ' ✓';
            case 'delivered': return ' ✓✓';
            case 'read': return ' ✓✓';
            default: return '';
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
            if (user.id === this.user.id) return; // Skip self
            
            const element = this.createUserElement(user);
            container.appendChild(element);
        });
    }

    createUserElement(user) {
        const div = document.createElement('div');
        div.className = 'user-item';
        div.dataset.userId = user.id;
        
        div.innerHTML = `
            <div class="avatar">
                <span>${user.name.charAt(0).toUpperCase()}</span>
            </div>
            <div class="user-info">
                <h4>${user.name}</h4>
                <p>${user.phone} • ${user.is_online ? 'Online' : 'Offline'}</p>
            </div>
        `;

        return div;
    }

    searchChats(query) {
        const items = document.querySelectorAll('.chat-item');
        items.forEach(item => {
            const name = item.querySelector('.chat-name').textContent.toLowerCase();
            const matches = name.includes(query.toLowerCase());
            item.style.display = matches ? '' : 'none';
        });
    }

    switchTab(tab) {
        // Update active tab button
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.tab === tab);
        });

        // Load content for the tab
        switch (tab) {
            case 'chats':
                this.loadChats();
                break;
            case 'groups':
                this.loadGroups();
                break;
            case 'contacts':
                this.loadContacts();
                break;
            case 'settings':
                this.loadSettings();
                break;
        }
    }

    showTypingIndicator(show) {
        const indicator = document.getElementById('typing-indicator');
        indicator.classList.toggle('active', show);
    }

    updateUserStatus(isOnline) {
        const statusElement = document.getElementById('user-status');
        statusElement.textContent = isOnline ? 'Online' : 'Connecting...';
        statusElement.style.color = isOnline ? '#4CAF50' : '#FF9800';
    }

    updateUserStatusInUI(userId, status) {
        // Update in user list if visible
        const userItem = document.querySelector(`.user-item[data-user-id="${userId}"]`);
        if (userItem) {
            const statusText = userItem.querySelector('p');
            const parts = statusText.textContent.split('•');
            parts[1] = ` ${status === 'online' ? 'Online' : 'Offline'}`;
            statusText.textContent = parts.join('•');
        }

        // Update in active chat if this user is the other participant
        if (this.activeChat) {
            const chat = this.chats.get(this.activeChat);
            if (chat?.type === 'direct') {
                const otherUser = chat.members?.find(m => m.user_id === userId);
                if (otherUser) {
                    document.getElementById('chat-status').textContent = 
                        status === 'online' ? 'online' : 'last seen recently';
                }
            }
        }
    }

    scrollToBottom() {
        const container = document.getElementById('messages-container');
        container.scrollTop = container.scrollHeight;
    }

    formatTime(dateString) {
        const date = new Date(dateString);
        const now = new Date();
        const diffMs = now - date;
        const diffMins = Math.floor(diffMs / 60000);
        const diffHours = Math.floor(diffMs / 3600000);
        const diffDays = Math.floor(diffMs / 86400000);

        if (diffMins < 1) return 'Just now';
        if (diffMins < 60) return `${diffMins}m`;
        if (diffHours < 24) return `${diffHours}h`;
        if (diffDays < 7) return `${diffDays}d`;
        
        return date.toLocaleDateString();
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    showError(message) {
        // Create error toast
        const toast = document.createElement('div');
        toast.className = 'error-toast';
        toast.textContent = message;
        toast.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: #f44336;
            color: white;
            padding: 12px 24px;
            border-radius: 8px;
            z-index: 1000;
            animation: slideIn 0.3s ease;
        `;
        
        document.body.appendChild(toast);
        
        setTimeout(() => {
            toast.style.animation = 'slideOut 0.3s ease';
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    }

    showRightPanel() {
        document.querySelector('.right-panel').classList.add('active');
    }

    hideRightPanel() {
        document.querySelector('.right-panel').classList.remove('active');
    }

    async loadGroups() {
        // Implement group loading
        console.log('Loading groups...');
    }

    async loadContacts() {
        // Implement contact loading
        console.log('Loading contacts...');
    }

    async loadSettings() {
        // Implement settings loading
        console.log('Loading settings...');
    }
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.chitchat = new ChitChat();
});

// Add CSS animations for toasts
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from { transform: translateX(100%); opacity: 0; }
        to { transform: translateX(0); opacity: 1; }
    }
    
    @keyframes slideOut {
        from { transform: translateX(0); opacity: 1; }
        to { transform: translateX(100%); opacity: 0; }
    }
    
    .error-toast {
        font-family: inherit;
    }
`;
document.head.appendChild(style);