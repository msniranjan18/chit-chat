// UIManager.js - Contains all UI-related functions
class UIManager {
    constructor(appInstance) {
        this.app = appInstance;
    }
    // UI Display Functions
    showChatScreen() {
        document.getElementById('auth-screen').classList.add('hidden');
        document.getElementById('chat-screen').classList.remove('hidden');
        
        if (this.app.user) {
            document.getElementById('user-name').textContent = this.app.user.name;
            document.getElementById('user-status').textContent = 'Online';
            document.getElementById('user-avatar').innerHTML = `<span>${this.app.user.name.charAt(0).toUpperCase()}</span>`;
        }
    }

    showNewChatModal() {
        document.getElementById('new-chat-modal').classList.remove('hidden');
        this.app.loadUsers();
    }

    hideModal() {
        document.getElementById('new-chat-modal').classList.add('hidden');
        document.getElementById('search-user').value = '';
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

    showError(message) {
        this.showToast(message, 'error');
    }

    updateUserStatus(isOnline) {
        const statusElement = document.getElementById('user-status');
        if (statusElement) {
            statusElement.textContent = isOnline ? 'Online' : 'Offline';
        }
    }

    hideRightPanel() {
        document.querySelector('.right-panel').classList.remove('active');
    }

    showUserActions(userId) {
        const user = this.app.users.get(userId);
        if (user) {
            this.showToast(`Viewing ${user.name}'s profile`, 'info');
        }
    }

    scrollToBottom() {
        const container = document.getElementById('messages-container');
        container.scrollTop = container.scrollHeight;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
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

    // UI Element Creation Functions
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
        
        div.addEventListener('click', async (e) => {
            e.stopPropagation();
            await this.app.startChatWithUser(user.id);
        });
        
        return div;
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

    getMessageStatusIcon(status) {
        switch(status) {
            case 'sent': return '✓';
            case 'delivered': return '✓✓';
            case 'read': return '✓✓✓';
            default: return '';
        }
    }

    createChatElement(chat) {
        const div = document.createElement('div');
        div.className = 'chat-item';
        div.dataset.chatId = chat.id;
        
        let chatName = 'Direct Chat';
        let lastMessage = 'No messages yet';
        let timestamp = '';
        let unreadCount = '';
        let avatarText = 'D';
        
        if (chat.type === 'direct') {
            let participants = chat.participants || chat.users || [];
            
            if (Array.isArray(participants) && participants.length > 0) {
                const otherUser = participants.find(p => p.id !== this.app.user?.id);
                
                if (otherUser) {
                    chatName = otherUser.name || otherUser.phone || 'Unknown User';
                    avatarText = chatName.charAt(0).toUpperCase();
                }
            }
        } else if (chat.type === 'group') {
            chatName = chat.name || 'Group Chat';
            avatarText = chatName.charAt(0).toUpperCase();
        }
        
        if (chat.last_message) {
            lastMessage = chat.last_message.content || '📎 Attachment';
            timestamp = this.formatTime(chat.last_message.sent_at);
        } else if (chat.last_activity) {
            timestamp = this.formatTime(chat.last_activity);
        } else if (chat.created_at) {
            timestamp = this.formatTime(chat.created_at);
        }
        
        if (chat.unread_count > 0) {
            unreadCount = `<div class="unread-count">${chat.unread_count}</div>`;
        }
        
        const shouldShowChat = chat.last_message || chat.name || 
                            (chat.type === 'direct' && chatName !== 'Direct Chat');
        
        if (!shouldShowChat) {
            return null;
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

    // UI Rendering Functions
    renderUserList(users) {
        const container = document.getElementById('user-list');
        container.innerHTML = '';

        if (users.length === 0) {
            container.innerHTML = '<p class="no-users">No users found</p>';
            return;
        }

        users.forEach(user => {
            if (user.id === this.app.user.id) return;
            
            const isContact = this.app.contacts.has(user.id);
            const element = this.createUserElement(user, isContact);
            container.appendChild(element);
        });
    }

    renderContactsList() {
        const container = document.getElementById('chat-list');
        container.innerHTML = '';
        
        // Check if contacts is defined and has size
        if (!this.app.contacts || this.app.contacts.size === 0) {
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
        
        // Convert to array safely
        const contactsArray = Array.from(this.app.contacts.values());
        contactsArray.forEach(user => {
            const element = this.createContactElement(user);
            container.appendChild(element);
        });
    }

    renderMessages(messages) {
        const container = document.getElementById('messages-container');
        container.innerHTML = '';
        
        if (messages.length === 0) {
            container.innerHTML = '<p class="no-messages">No messages yet. Start the conversation!</p>';
            return;
        }
        
        messages.forEach(message => {
            const isSentByMe = message.sender_id === this.app.user.id;
            const messageElement = this.createMessageElement(message, isSentByMe);
            container.appendChild(messageElement);
        });
        
        this.scrollToBottom();
    }

    renderChatList(chats) {
        const container = document.getElementById('chat-list');
        container.innerHTML = '';

        const sortedChats = chats.sort((a, b) => {
            if (a.is_pinned !== b.is_pinned) return b.is_pinned ? 1 : -1;
            return new Date(b.last_activity) - new Date(a.last_activity);
        });

        sortedChats.forEach(chat => {
            const chatElement = this.createChatElement(chat);
            if (chatElement) {
                container.appendChild(chatElement);
            }
        });
    }

    // UI Update Functions
    addMessageToUI(message, isSentByMe = true) {
        const container = document.getElementById('messages-container');
        const messageElement = this.createMessageElement(message, isSentByMe);
        container.appendChild(messageElement);
        this.scrollToBottom();
    }

    updateChatLastMessage(chatId, message) {
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

    // Search Functions
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

    // Menu and Tab Functions
    switchTab(tab) {
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.tab === tab);
        });

        if (tab !== 'chats') {
            this.app.activeChat = null;
            document.getElementById('empty-chat').classList.remove('hidden');
            document.getElementById('active-chat').classList.add('hidden');
        }

        switch (tab) {
            case 'chats':
                this.app.loadChats();
                document.querySelector('.chat-list').style.display = 'block';
                break;
            case 'groups':
                this.loadGroups();
                break;
            case 'contacts':
                this.app.loadContacts();
                this.renderContactsList();
                break;
            case 'settings':
                this.loadSettings();
                break;
        }
    }

    showMenu(button) {
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
            
            document.getElementById('logout-btn').addEventListener('click', (e) => {
                e.preventDefault();
                this.app.handleLogout();
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
        
        const rect = button.getBoundingClientRect();
        menu.style.top = `${rect.bottom + 5}px`;
        menu.style.left = `${rect.left}px`;
        menu.classList.add('show');
        
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

    showProfile() {
        if (!this.app.user) return;
        
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
                            <span>${this.app.user.name.charAt(0).toUpperCase()}</span>
                        </div>
                        <h2>${this.app.user.name}</h2>
                        <p>${this.app.user.phone}</p>
                    </div>
                    <div class="profile-info">
                        <div class="info-item">
                            <label>Status</label>
                            <p>${this.app.user.status || 'Hey there! I am using ChitChat'}</p>
                        </div>
                        <div class="info-item">
                            <label>Last Seen</label>
                            <p>${this.formatTime(this.app.user.last_seen)}</p>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        document.body.appendChild(modal);
        modal.classList.remove('hidden');
        
        modal.querySelector('.close-modal').addEventListener('click', () => {
            modal.remove();
        });
        
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.remove();
            }
        });
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
        
        const button = document.querySelector('.chat-header-actions .icon-btn:last-child');
        const rect = button.getBoundingClientRect();
        menu.style.top = `${rect.bottom + 5}px`;
        menu.style.right = `${window.innerWidth - rect.right}px`;
        menu.classList.add('show');
        
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

    loadGroups() {
        const container = document.getElementById('chat-list');
        container.innerHTML = `
            <div class="empty-chat-list">
                <p>Groups feature coming soon!</p>
                <button class="btn-secondary" onclick="chitchat.uiManager.showToast('Create group feature coming soon!')">
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
}

// Export for use in other modules
export { UIManager };

window.UIManager = UIManager;
