// ChatManager.js - Contains all chat-related logic and API calls
class ChatManager {
    constructor(appInstance) {
        this.app = appInstance
    }

    // Chat Operations
    async loadChats() {
        try {
            const response = await fetch('/api/chats', {
                headers: { 'Authorization': `Bearer ${this.app.token}` }
            });
            
            if (!response.ok) throw new Error('Failed to load chats');
            
            const data = await response.json();
            
            this.app.chats.clear();
            data.chats.forEach(chat => {
                this.app.chats.set(chat.id, chat);
            });
            
            this.app.uiManager.renderChatList(data.chats);
        } catch (error) {
            console.error('Error loading chats:', error);
            this.app.uiManager.showError('Failed to load chats');
        }
    }

    async loadChat(chatId) {
        try {
            const response = await fetch(`/api/chats/${chatId}`, {
                headers: { 'Authorization': `Bearer ${this.app.token}` }
            });
            
            if (!response.ok) throw new Error('Failed to load chat');
            
            const chat = await response.json();
            this.app.chats.set(chatId, chat);
        } catch (error) {
            console.error('Error loading chat:', error);
        }
    }

    selectChat(chatId) {
        this.app.activeChat = chatId;
        const chat = this.app.chats.get(chatId);
        
        if (!chat) {
            console.error('Chat not found:', chatId);
            return;
        }
        
        document.getElementById('empty-chat').classList.add('hidden');
        document.getElementById('active-chat').classList.remove('hidden');
        
        const chatTitle = document.getElementById('chat-title');
        const chatStatus = document.getElementById('chat-status');
        
        let participants = chat.participants || chat.users || [];
        
        if (chat.type === 'direct') {
            const otherUser = participants.find(p => p.id !== this.app.user.id);
            
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
        
        this.loadMessages(chatId);
        
        document.querySelectorAll('.chat-item').forEach(item => {
            item.classList.toggle('active', item.dataset.chatId === chatId);
        });
    }

    async loadMessages(chatId) {
        try {
            const response = await fetch(`/api/messages?chat_id=${chatId}`, {
                headers: { 'Authorization': `Bearer ${this.app.token}` }
            });
            
            if (!response.ok) throw new Error('Failed to load messages');
            
            const data = await response.json();
            this.app.uiManager.renderMessages(data.messages);
        } catch (error) {
            console.error('Error loading messages:', error);
        }
    }

    async sendMessage() {
        const input = document.getElementById('message-input');
        const content = input.value.trim();
        
        if (!content || !this.app.activeChat) return;
        
        try {
            const response = await fetch('/api/messages', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.app.token}`
                },
                body: JSON.stringify({
                    chat_id: this.app.activeChat,
                    content: content,
                    content_type: 'text'
                })
            });
            
            if (!response.ok) throw new Error('Failed to send message');
            
            input.value = '';
            this.loadMessages(this.app.activeChat);
        } catch (error) {
            console.error('Error sending message:', error);
            this.app.uiManager.showError('Failed to send message');
        }
    }

    async markMessageAsRead(messageId) {
        try {
            await fetch('/api/messages/status', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.app.token}`
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

    // User Operations
    async loadUsers() {
        try {
            const response = await fetch('/api/users/search?q=', {
                headers: { 'Authorization': `Bearer ${this.app.token}` }
            });
            
            if (!response.ok) throw new Error('Failed to load users');
            
            const users = await response.json();
            
            // Check if users is an array
            if (Array.isArray(users)) {
                this.app.uiManager.renderUserList(users);
                
                users.forEach(user => {
                    this.app.users.set(user.id, user);
                });
            } else {
                this.app.uiManager.renderUserList([]);
            }
        } catch (error) {
            console.error('Error loading users:', error);
            this.app.uiManager.renderUserList([]);
        }
    }

    async searchUsers(query) {
        try {
            const response = await fetch(`/api/users/search?q=${encodeURIComponent(query)}`, {
                headers: { 'Authorization': `Bearer ${this.app.token}` }
            });
            
            if (!response.ok) throw new Error('Search failed');
            
            const users = await response.json();
            this.app.uiManager.renderUserList(users);
        } catch (error) {
            console.error('Search error:', error);
        }
    }

    // Contact Operations
    async addContact(userId) {
        try {
            const response = await fetch('/api/contacts', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.app.token}`
                },
                body: JSON.stringify({ user_id: userId })
            });
            
            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`Failed to add contact: ${response.status} - ${errorText}`);
            }
            
            this.app.uiManager.showToast('Contact added successfully');
            await this.loadContacts();
            
            const user = this.app.users.get(userId);
            if (user) {
                this.app.contacts.set(userId, user);
                // Check if users exists before rendering
                const usersArray = Array.from(this.app.users.values());
                if (usersArray && usersArray.length > 0) {
                    this.app.uiManager.renderUserList(usersArray);
                }
            }
        } catch (error) {
            console.error('Error adding contact:', error);
            this.app.uiManager.showError('Failed to add contact');
        }
    }

    async removeContact(userId) {
        try {
            const response = await fetch(`/api/contacts/${userId}`, {
                method: 'DELETE',
                headers: { 'Authorization': `Bearer ${this.app.token}` }
            });
            
            if (!response.ok) throw new Error('Failed to remove contact');
            
            this.app.uiManager.showToast('Contact removed');
            this.app.contacts.delete(userId);
            this.loadContacts();
            
            this.app.uiManager.renderUserList(Array.from(this.app.users.values()));
        } catch (error) {
            console.error('Error removing contact:', error);
            this.app.uiManager.showError('Failed to remove contact');
        }
    }

    async loadContacts() {
        try {
            const response = await fetch('/api/contacts', {
                headers: { 'Authorization': `Bearer ${this.app.token}` }
            });
            
            if (!response.ok) {
                // If 404 or empty response, treat as no contacts
                if (response.status === 404) {
                    this.app.contacts.clear();
                    return;
                }
                throw new Error('Failed to load contacts');
            }
            
            const contacts = await response.json();
            this.app.contacts.clear();
            
            // Handle null or undefined response
            if (contacts && Array.isArray(contacts)) {
                contacts.forEach(user => {
                    this.app.contacts.set(user.id, user);
                });
            } else {
                // If contacts is null/undefined, just clear the map
                this.app.contacts.clear();
            }
        } catch (error) {
            console.error('Error loading contacts:', error);
            // Don't show error for empty contacts, just clear
            if (error.message !== 'Failed to load contacts') {
                this.app.uiManager.showError('Failed to load contacts');
            }
            this.app.contacts.clear();
        }
    }

    // Chat Creation
    async startChatWithUser(userId) {
        try {
            const existingChat = Array.from(this.app.chats.values()).find(chat => {
                return chat.type === 'direct' && 
                    chat.participants && 
                    chat.participants.some(p => p.id === userId);
            });
            
            if (existingChat) {
                this.selectChat(existingChat.id);
            } else {
                const response = await fetch('/api/chats', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${this.app.token}`
                    },
                    body: JSON.stringify({
                        type: 'direct',
                        user_ids: [userId]
                    })
                });
                
                if (!response.ok) throw new Error('Failed to create chat');
                
                const data = await response.json();
                const chat = data.chat;
                
                this.app.chats.set(chat.id, chat);
                this.loadChats();
                this.selectChat(chat.id);
            }
            
            this.app.uiManager.hideModal();
        } catch (error) {
            console.error('Error starting chat:', error);
            this.app.uiManager.showError('Failed to start chat');
        }
    }

    async handleCreateChat() {
        const chatNameInput = document.getElementById('new-chat-name');
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
                    'Authorization': `Bearer ${this.app.token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            if (response.ok) {
                const data = await response.json();
                this.app.chats.set(data.chat.id, data.chat);
                this.app.uiManager.hideModal();
                this.selectChat(data.chat.id);
            }
        } catch (err) {
            console.error("Failed to create chat", err);
        }
    }

    // Message Handling
    handleIncomingMessage(message) {
        const chatId = message.chat_id;
        if (!this.app.chats.has(chatId)) {
            this.loadChat(chatId);
        }
        
        if (this.app.activeChat === chatId) {
            this.app.uiManager.addMessageToUI(message, false);
            this.app.uiManager.scrollToBottom();
            this.markMessageAsRead(message.id);
        } else {
            this.app.uiManager.updateChatUnreadCount(chatId);
        }
        
        this.app.uiManager.updateChatLastMessage(chatId, message);
    }

    // Helper Methods
    getChatName(chat) {
        if (chat.type === 'group' || chat.type === 'channel') {
            return chat.name || 'Unnamed Group';
        }

        if (chat.members && chat.members.length > 0) {
            const otherMember = chat.members.find(m => m.user_id !== this.app.user.id);
            if (otherMember) {
                return otherMember.display_name || (otherMember.user && otherMember.user.name) || otherMember.user_id;
            }
        }
        return 'Direct Chat-1';
    }

    handleChatUpdate(data) {
        const { chat_id } = data;
        if (this.app.chats.has(chat_id)) {
            this.loadChat(chat_id);
        }
    }
}

// Export for use in other modules
export { ChatManager };

window.ChatManager = ChatManager;
