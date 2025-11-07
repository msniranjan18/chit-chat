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

        // // User list clicks (in modal) - Add contact functionality
        // document.getElementById('user-list').addEventListener('click', (e) => {
        //     const userItem = e.target.closest('.user-item');
        //     if (userItem && userItem.dataset.userId) {
        //         if (e.target.classList.contains('add-contact-btn')) {
        //             this.addContact(userItem.dataset.userId);
        //         } else if (e.target.classList.contains('start-chat-btn')) {
        //             this.startChatWithUser(userItem.dataset.userId);
        //         } else {
        //             this.showUserActions(userItem.dataset.userId);
        //         }
        //     }
        // });


        // User list clicks (in modal) - Add contact functionality
        document.getElementById('user-list').addEventListener('click', (e) => {
            // Check if click is on an ADD button
            if (e.target.classList.contains('add-contact-btn')) {
                e.preventDefault();
                e.stopPropagation();
                const userItem = e.target.closest('.user-item');
                if (userItem && userItem.dataset.userId) {
                    this.addContact(userItem.dataset.userId);
                }
                return;
            }
            
            // Check if click is on a START CHAT button
            if (e.target.classList.contains('start-chat-btn')) {
                e.preventDefault();
                e.stopPropagation();
                const userItem = e.target.closest('.user-item');
                if (userItem && userItem.dataset.userId) {
                    this.startChatWithUser(userItem.dataset.userId);
                }
                return;
            }
            
            // Check if click is on a REMOVE button
            if (e.target.classList.contains('remove-contact-btn')) {
                e.preventDefault();
                e.stopPropagation();
                const userItem = e.target.closest('.user-item');
                if (userItem && userItem.dataset.userId) {
                    this.removeContact(userItem.dataset.userId);
                }
                return;
            }
            
            // If click is anywhere else on the user item (but not on buttons), show user actions
            const userItem = e.target.closest('.user-item');
            if (userItem && userItem.dataset.userId && 
                !e.target.classList.contains('btn-primary') && 
                !e.target.classList.contains('btn-secondary') && 
                !e.target.classList.contains('btn-text')) {
                this.showUserActions(userItem.dataset.userId);
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
            // Validate token before using it
            this.validateToken(token).then(isValid => {
                if (isValid) {
                    this.token = token;
                    this.user = JSON.parse(userStr);
                    this.showChatScreen();
                    this.connectWebSocket();
                    this.loadChats();
                    this.loadContacts();
                } else {
                    // Invalid token, clear storage
                    this.clearAuthStorage();
                    this.showToast('Session expired. Please login again.', 'error');
                }
            }).catch(() => {
                this.clearAuthStorage();
            });
        }
    }

    async validateToken(token) {
        try {
            const response = await fetch('/api/auth/verify', {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            return response.ok;
        } catch (error) {
            return false;
        }
    }

    clearAuthStorage() {
        localStorage.removeItem('chitchat_token');
        localStorage.removeItem('chitchat_user');
        this.token = null;
        this.user = null;
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
            console.log('Chats data:', data.chats); // Debug log
            
            // Store chats
            data.chats.forEach(chat => {
                this.chats.set(chat.id, chat);
            });
            
            this.renderChatList(data.chats);
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
                    `<button class="btn-secondary start-chat-btn" data-user-id="${user.id}">Chat</button>
                    <button class="btn-text remove-contact-btn" data-user-id="${user.id}">Remove</button>` :
                    `<button class="btn-primary add-contact-btn" data-user-id="${user.id}">Add</button>`
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
            
            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`Failed to add contact: ${response.status} - ${errorText}`);
            }
            
            this.showToast('Contact added successfully');
            await this.loadContacts();
            
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
        
        // div.addEventListener('click', () => {
        //     this.startChatWithUser(user.id);
        // });

        // FIXED: Proper click handler for contact items
        div.addEventListener('click', async (e) => {
            e.stopPropagation();
            await this.startChatWithUser(user.id);
        });
        
        return div;
    }

    async startChatWithUser(userId) {
        try {
            // Check if a direct chat already exists with this user
            const existingChat = Array.from(this.chats.values()).find(chat => {
                return chat.type === 'direct' && 
                    chat.participants && 
                    chat.participants.some(p => p.id === userId);
            });
            
            if (existingChat) {
                // Open existing chat
                this.selectChat(existingChat.id);
            } else {
                // Create a new direct chat
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
                
                if (!response.ok) throw new Error('Failed to create chat');
                
                const data = await response.json();
                const chat = data.chat;
                
                // Add chat to local state
                this.chats.set(chat.id, chat);
                
                // Render chat list with new chat
                this.loadChats();
                
                // Select the new chat
                this.selectChat(chat.id);
            }
            
            // Hide new chat modal if open
            this.hideModal();
        } catch (error) {
            console.error('Error starting chat:', error);
            this.showError('Failed to start chat');
        }
    }

    selectChat(chatId) {
        this.activeChat = chatId;
        const chat = this.chats.get(chatId);
        
        if (!chat) {
            console.error('Chat not found:', chatId);
            return;
        }
        
        // Update UI to show active chat
        document.getElementById('empty-chat').classList.add('hidden');
        document.getElementById('active-chat').classList.remove('hidden');
        
        // Set chat title
        const chatTitle = document.getElementById('chat-title');
        const chatStatus = document.getElementById('chat-status');
        
        // Get participants from multiple possible locations
        let participants = chat.participants || chat.users || [];
        
        if (chat.type === 'direct') {
            // Find the other participant
            const otherUser = participants.find(p => p.id !== this.user.id);
            
            if (otherUser) {
                chatTitle.textContent = otherUser.name || otherUser.phone || 'Direct Chat';
                chatStatus.textContent = otherUser.is_online ? 'Online' : 'Offline';
            } else {
                chatTitle.textContent = 'Direct Chat';
                chatStatus.textContent = 'Chat';
            }
        } else {
            chatTitle.textContent = chat.name || 'Group Chat';
            chatStatus.textContent = `${participants.length || 1} members`;
        }
        
        // Load messages for this chat
        this.loadMessages(chatId);
        
        // Mark chat as active in sidebar
        document.querySelectorAll('.chat-item').forEach(item => {
            item.classList.toggle('active', item.dataset.chatId === chatId);
        });
    }

    async loadMessages(chatId) {
        try {
            const response = await fetch(`/api/messages?chat_id=${chatId}`, {
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
            
            if (!response.ok) throw new Error('Failed to load messages');
            
            const data = await response.json();
            this.renderMessages(data.messages);
        } catch (error) {
            console.error('Error loading messages:', error);
        }
    }

    renderMessages(messages) {
        const container = document.getElementById('messages-container');
        container.innerHTML = '';
        
        if (messages.length === 0) {
            container.innerHTML = '<p class="no-messages">No messages yet. Start the conversation!</p>';
            return;
        }
        
        messages.forEach(message => {
            const isSentByMe = message.sender_id === this.user.id;
            const messageElement = this.createMessageElement(message, isSentByMe);
            container.appendChild(messageElement);
        });
        
        this.scrollToBottom();
    }

    createMessageElement(message, isSentByMe) {
        const div = document.createElement('div');
        div.className = `message ${isSentByMe ? 'sent' : 'received'}`;
        
        const time = new Date(message.sent_at).toLocaleTimeString([], { 
            hour: '2-digit', 
            minute: '2-digit' 
        });
        
        div.innerHTML = `
            <div class="message-content">${this.escapeHtml(message.content)}</div>
            <div class="message-time">
                ${time}
                ${isSentByMe ? `<span class="message-status">${this.getMessageStatusIcon(message.status)}</span>` : ''}
            </div>
        `;
        
        return div;
    }

    async sendMessage() {
        const input = document.getElementById('message-input');
        const content = input.value.trim();
        
        if (!content || !this.activeChat) return;
        
        try {
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
            
            // Clear input
            input.value = '';
            
            // Reload messages
            this.loadMessages(this.activeChat);
        } catch (error) {
            console.error('Error sending message:', error);
            this.showError('Failed to send message');
        }
    }

    getMessageStatusIcon(status) {
        switch(status) {
            case 'sent': return '✓';
            case 'delivered': return '✓✓';
            case 'read': return '✓✓✓';
            default: return '';
        }
    }

    getChatName(chat) {
        if (chat.type === 'group' || chat.type === 'channel') {
            return chat.name || 'Unnamed Group';
        }

        // For direct chats, find the other participant
        if (chat.members && chat.members.length > 0) {
            const otherMember = chat.members.find(m => m.user_id !== this.user.id);
            if (otherMember) {
                // Priority: DisplayName > User Object Name > Phone
                return otherMember.display_name || (otherMember.user && otherMember.user.name) || otherMember.user_id;
            }
        }
        return 'Direct Chat-1';
    }

    renderChatList() {
        const chatList = document.getElementById('chat-list');
        chatList.innerHTML = '';

        // Sort: Pinned first, then by last activity
        const sortedChats = Array.from(this.chats.values()).sort((a, b) => {
            if (a.is_pinned !== b.is_pinned) return b.is_pinned ? 1 : -1;
            return new Date(b.last_activity) - new Date(a.last_activity);
        });

        sortedChats.forEach(chat => {
            const chatName = this.getChatName(chat); // Using the fix here
            const isActive = this.activeChat && this.activeChat.id === chat.id;
            
            const chatEl = document.createElement('div');
            chatEl.className = `chat-item ${isActive ? 'active' : ''}`;
            chatEl.innerHTML = `
                <div class="chat-avatar">${chatName.charAt(0).toUpperCase()}</div>
                <div class="chat-info">
                    <div class="chat-name">${chatName}</div>
                    <div class="chat-last-msg">${chat.last_message ? chat.last_message.content : 'No messages yet'}</div>
                </div>
                ${chat.unread_count > 0 ? `<div class="unread-badge">${chat.unread_count}</div>` : ''}
            `;
            chatEl.onclick = () => this.switchChat(chat.id);
            chatList.appendChild(chatEl);
        });
    }

    async handleCreateChat() {
        const chatNameInput = document.getElementById('new-chat-name'); // Ensure this ID exists in your HTML
        const selectedUserIds = Array.from(document.querySelectorAll('.user-checkbox:checked')).map(cb => cb.value);

        if (selectedUserIds.length === 0) return;

        const isGroup = selectedUserIds.length > 1 || (chatNameInput && chatNameInput.value);
        
        const payload = {
            type: isGroup ? 'group' : 'direct',
            user_ids: selectedUserIds,
            name: isGroup ? chatNameInput.value : null
        };

        try {
            const response = await fetch('/api/chats', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${this.token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            if (response.ok) {
                const data = await response.json();
                this.chats.set(data.chat.id, data.chat);
                this.closeModal('new-chat-modal');
                this.switchChat(data.chat.id);
            }
        } catch (err) {
            console.error("Failed to create chat", err);
        }
    }

    createChatElement(chat) {
        const div = document.createElement('div');
        div.className = 'chat-item';
        div.dataset.chatId = chat.id;
        
        // Get chat name with proper fallbacks
        let chatName = 'Direct Chat';
        let lastMessage = 'No messages yet';
        let timestamp = '';
        let unreadCount = '';
        let avatarText = 'D';
        
        // For direct chats, get the other user's name
        if (chat.type === 'direct') {
            // Try multiple ways to get participants
            let participants = chat.participants || chat.users || [];
            
            if (Array.isArray(participants) && participants.length > 0) {
                // Find the other participant (not current user)
                const otherUser = participants.find(p => p.id !== this.user?.id);
                
                if (otherUser) {
                    chatName = otherUser.name || otherUser.phone || 'Unknown User';
                    avatarText = chatName.charAt(0).toUpperCase();
                }
            }
        } else if (chat.type === 'group') {
            chatName = chat.name || 'Group Chat';
            avatarText = chatName.charAt(0).toUpperCase();
        }
        
        // Get last message info
        if (chat.last_message) {
            lastMessage = chat.last_message.content || '📎 Attachment';
            timestamp = this.formatTime(chat.last_message.sent_at);
        } else if (chat.last_activity) {
            timestamp = this.formatTime(chat.last_activity);
        } else if (chat.created_at) {
            timestamp = this.formatTime(chat.created_at);
        }
        
        // Get unread count
        if (chat.unread_count > 0) {
            unreadCount = `<div class="unread-count">${chat.unread_count}</div>`;
        }
        
        // Filter: Don't show chats with no messages AND no name (empty chats)
        // Only show if there's at least one message OR it's a named group
        const shouldShowChat = chat.last_message || chat.name || 
                            (chat.type === 'direct' && chatName !== 'Direct Chat');
        
        if (!shouldShowChat) {
            return null; // Don't render empty direct chats
        }
        
        div.innerHTML = `
            <div class="chat-avatar">
                <span>${avatarText}</span>
            </div>
            <div class="chat-info">
                <div class="chat-name">${chatName}</div>
                <div class="last-message">${lastMessage}</div>
                <div class="chat-meta">
                    <span class="timestamp">${timestamp}</span>
                    ${unreadCount}
                </div>
            </div>
        `;
        
        return div;
    }

    // Add these other missing helper methods:

    updateUserStatus(isOnline) {
        const statusElement = document.getElementById('user-status');
        if (statusElement) {
            statusElement.textContent = isOnline ? 'Online' : 'Offline';
        }
    }

    addMessageToUI(message, isSentByMe = true) {
        const container = document.getElementById('messages-container');
        const messageElement = this.createMessageElement(message, isSentByMe);
        container.appendChild(messageElement);
        this.scrollToBottom();
    }

    updateChatLastMessage(chatId, message) {
        // Update the chat list item with the last message
        const chatElement = document.querySelector(`.chat-item[data-chat-id="${chatId}"]`);
        if (chatElement) {
            const lastMessageEl = chatElement.querySelector('.last-message');
            const timestampEl = chatElement.querySelector('.timestamp');
            
            if (lastMessageEl) {
                lastMessageEl.textContent = message.content || '📎 Attachment';
            }
            if (timestampEl) {
                timestampEl.textContent = this.formatTime(message.sent_at);
            }
        }
    }

    updateChatUnreadCount(chatId) {
        // Increment unread count for a chat
        const chatElement = document.querySelector(`.chat-item[data-chat-id="${chatId}"]`);
        if (chatElement) {
            let unreadCount = chatElement.querySelector('.unread-count');
            if (!unreadCount) {
                const metaEl = chatElement.querySelector('.chat-meta');
                unreadCount = document.createElement('div');
                unreadCount.className = 'unread-count';
                metaEl.appendChild(unreadCount);
            }
            const currentCount = parseInt(unreadCount.textContent) || 0;
            unreadCount.textContent = currentCount + 1;
        }
    }

    async markMessageAsRead(messageId) {
        try {
            await fetch('/api/messages/status', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.token}`
                },
                body: JSON.stringify({
                    message_id: messageId,
                    status: 'read'
                })
            });
        } catch (error) {
            console.error('Error marking message as read:', error);
        }
    }

    async loadChat(chatId) {
        try {
            const response = await fetch(`/api/chats/${chatId}`, {
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
            
            if (!response.ok) throw new Error('Failed to load chat');
            
            const chat = await response.json();
            this.chats.set(chatId, chat);
        } catch (error) {
            console.error('Error loading chat:', error);
        }
    }

    handleTypingIndicator(data) {
        if (data.chat_id === this.activeChat && data.user_id !== this.user?.id) {
            const typingIndicator = document.getElementById('typing-indicator');
            if (data.is_typing) {
                typingIndicator.classList.add('active');
            } else {
                typingIndicator.classList.remove('active');
            }
        }
    }

    handlePresenceUpdate(data) {
        // Update user online status in UI
        const { user_id, is_online } = data;
        
        // Update in active chat if this is the other user
        if (this.activeChat) {
            const chat = this.chats.get(this.activeChat);
            if (chat && chat.type === 'direct') {
                const otherUser = chat.participants?.find(p => p.id === user_id) ||
                                chat.users?.find(p => p.id === user_id);
                if (otherUser) {
                    const chatStatus = document.getElementById('chat-status');
                    if (chatStatus) {
                        chatStatus.textContent = is_online ? 'Online' : 'Offline';
                    }
                }
            }
        }
        
        // Update in contacts list
        const contactElement = document.querySelector(`.chat-item[data-user-id="${user_id}"]`);
        if (contactElement) {
            const timestampEl = contactElement.querySelector('.timestamp');
            if (timestampEl) {
                timestampEl.textContent = is_online ? 'Online' : 'Offline';
            }
        }
    }

    handleStatusUpdate(data) {
        // Update message status in UI
        const { message_id, status } = data;
        const messageElement = document.querySelector(`.message[data-message-id="${message_id}"]`);
        if (messageElement) {
            const statusElement = messageElement.querySelector('.message-status');
            if (statusElement) {
                statusElement.textContent = this.getMessageStatusIcon(status);
            }
        }
    }

    handleChatUpdate(data) {
        // Handle chat updates (like new members, name changes, etc.)
        const { chat_id } = data;
        if (this.chats.has(chat_id)) {
            this.loadChat(chat_id); // Reload chat data
        }
    }

    handleTyping() {
        if (!this.activeChat || this.isTyping) return;
        
        this.isTyping = true;
        
        // Send typing indicator via WebSocket
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({
                type: 'typing',
                payload: {
                    chat_id: this.activeChat,
                    is_typing: true
                }
            }));
        }
        
        // Clear typing indicator after 3 seconds
        this.typingTimeout = setTimeout(() => {
            this.isTyping = false;
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                this.ws.send(JSON.stringify({
                    type: 'typing',
                    payload: {
                        chat_id: this.activeChat,
                        is_typing: false
                    }
                }));
            }
        }, 3000);
    }

    searchChats(query) {
        const chatItems = document.querySelectorAll('.chat-item');
        chatItems.forEach(item => {
            const chatName = item.querySelector('.chat-name').textContent.toLowerCase();
            const lastMessage = item.querySelector('.last-message').textContent.toLowerCase();
            
            if (chatName.includes(query.toLowerCase()) || lastMessage.includes(query.toLowerCase())) {
                item.style.display = 'flex';
            } else {
                item.style.display = 'none';
            }
        });
    }

    formatTime(dateString) {
        if (!dateString) return '';
        
        const date = new Date(dateString);
        const now = new Date();
        const diffMs = now - date;
        const diffMins = Math.floor(diffMs / 60000);
        const diffHours = Math.floor(diffMs / 3600000);
        const diffDays = Math.floor(diffMs / 86400000);
        
        if (diffMins < 1) return 'Just now';
        if (diffMins < 60) return `${diffMins}m ago`;
        if (diffHours < 24) return `${diffHours}h ago`;
        if (diffDays < 7) return `${diffDays}d ago`;
        
        return date.toLocaleDateString();
    }

    hideRightPanel() {
        document.querySelector('.right-panel').classList.remove('active');
    }

    showUserActions(userId) {
        // Show user action menu (profile view, etc.)
        const user = this.users.get(userId);
        if (user) {
            this.showToast(`Viewing ${user.name}'s profile`, 'info');
            // You can implement a proper user profile modal here
        }
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    scrollToBottom() {
        const container = document.getElementById('messages-container');
        container.scrollTop = container.scrollHeight;
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
