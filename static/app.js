// Import the classes
import { UIManager } from './UIManager.js';
import { ChatManager } from './ChatManager.js';
import { WebSocketManager } from './WebSocketManager.js';

// app.js - Main entry point with minimal code
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
        
        // Initialize managers
        this.uiManager = new UIManager(this);
        this.chatManager = new ChatManager(this);
        this.wsManager = new WebSocketManager(this);
        
        // Pass app reference to managers
        // this.uiManager.app = this;
        // this.chatManager.app = this;
        // this.wsManager.app = this;
        
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
        document.getElementById('start-chat-btn').addEventListener('click', () => this.uiManager.showNewChatModal());
        document.getElementById('new-chat-btn').addEventListener('click', () => this.uiManager.showNewChatModal());
        document.getElementById('send-btn').addEventListener('click', () => this.chatManager.sendMessage());
        document.getElementById('message-input').addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.chatManager.sendMessage();
            } else {
                this.handleTyping();
            }
        });
        
        // Search
        document.getElementById('chat-search').addEventListener('input', (e) => this.uiManager.searchChats(e.target.value));
        document.getElementById('search-user').addEventListener('input', (e) => this.chatManager.searchUsers(e.target.value));

        // Tabs - FIXED
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                this.uiManager.switchTab(e.currentTarget.dataset.tab);
            });
        });

        // Modals
        document.querySelector('.close-modal').addEventListener('click', () => this.uiManager.hideModal());
        document.querySelector('.close-panel').addEventListener('click', () => this.uiManager.hideRightPanel());

        // Chat list clicks
        document.getElementById('chat-list').addEventListener('click', (e) => {
            const chatItem = e.target.closest('.chat-item');
            if (chatItem && chatItem.dataset.chatId) {
                this.chatManager.selectChat(chatItem.dataset.chatId);
            }
        });

        // User list clicks (in modal) - Add contact functionality
        document.getElementById('user-list').addEventListener('click', (e) => {
            // Check if click is on an ADD button
            if (e.target.classList.contains('add-contact-btn')) {
                e.preventDefault();
                e.stopPropagation();
                const userItem = e.target.closest('.user-item');
                if (userItem && userItem.dataset.userId) {
                    this.chatManager.addContact(userItem.dataset.userId);
                }
                return;
            }
            
            // Check if click is on a START CHAT button
            if (e.target.classList.contains('start-chat-btn')) {
                e.preventDefault();
                e.stopPropagation();
                const userItem = e.target.closest('.user-item');
                if (userItem && userItem.dataset.userId) {
                    this.chatManager.startChatWithUser(userItem.dataset.userId);
                }
                return;
            }
            
            // Check if click is on a REMOVE button
            if (e.target.classList.contains('remove-contact-btn')) {
                e.preventDefault();
                e.stopPropagation();
                const userItem = e.target.closest('.user-item');
                if (userItem && userItem.dataset.userId) {
                    this.chatManager.removeContact(userItem.dataset.userId);
                }
                return;
            }
            
            // If click is anywhere else on the user item (but not on buttons), show user actions
            const userItem = e.target.closest('.user-item');
            if (userItem && userItem.dataset.userId && 
                !e.target.classList.contains('btn-primary') && 
                !e.target.classList.contains('btn-secondary') && 
                !e.target.classList.contains('btn-text')) {
                this.uiManager.showUserActions(userItem.dataset.userId);
            }
        });

        // Menu button - FIXED
        document.getElementById('menu-btn').addEventListener('click', (e) => this.uiManager.showMenu(e.currentTarget));
        
        // Voice and video call buttons (placeholder)
        document.querySelectorAll('.chat-header-actions .icon-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const icon = e.currentTarget.querySelector('i').className;
                if (icon.includes('phone')) {
                    this.uiManager.showToast('Voice call feature coming soon!');
                } else if (icon.includes('video')) {
                    this.uiManager.showToast('Video call feature coming soon!');
                } else if (icon.includes('ellipsis')) {
                    this.uiManager.showChatOptions();
                }
            });
        });

        // Input action buttons
        document.querySelectorAll('.input-actions .icon-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const icon = e.currentTarget.querySelector('i').className;
                if (icon.includes('paperclip')) {
                    this.uiManager.showToast('File attachment coming soon!');
                } else if (icon.includes('smile')) {
                    this.uiManager.showToast('Emoji picker coming soon!');
                }
            });
        });
    }

    async handleLogin() {
        const phone = document.getElementById('phone').value.trim();
        const name = document.getElementById('name').value.trim();

        if (!phone || phone.length !== 10) {
            this.uiManager.showError('Please enter a valid 10-digit phone number');
            return;
        }

        if (!name) {
            this.uiManager.showError('Please enter your name');
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
            
            this.uiManager.showChatScreen();
            this.wsManager.connectWebSocket();
            this.chatManager.loadChats();
            this.chatManager.loadContacts();
        } catch (error) {
            console.error('Login error:', error);
            this.uiManager.showError('Login failed. Please try again.');
        }
    }

    checkAuth() {
        const token = localStorage.getItem('chitchat_token');
        const userStr = localStorage.getItem('chitchat_user');
        
        if (token && userStr) {
            this.validateToken(token).then(isValid => {
                if (isValid) {
                    this.token = token;
                    this.user = JSON.parse(userStr);
                    this.uiManager.showChatScreen();
                    this.wsManager.connectWebSocket();
                    this.chatManager.loadChats();
                    this.chatManager.loadContacts();
                } else {
                    this.clearAuthStorage();
                    this.uiManager.showToast('Session expired. Please login again.', 'error');
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

    handleTyping() {
        if (!this.activeChat || this.isTyping) return;
        
        this.isTyping = true;
        
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({
                type: 'typing',
                payload: {
                    chat_id: this.activeChat,
                    is_typing: true
                }
            }));
        }
        
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

    async handleLogout() {
        try {
            await fetch('/api/auth/logout', {
                method: 'POST',
                headers: { 'Authorization': `Bearer ${this.token}` }
            });
        } catch (error) {
            console.error('Logout error:', error);
        }
        
        localStorage.removeItem('chitchat_token');
        localStorage.removeItem('chitchat_user');
        
        this.token = null;
        this.user = null;
        this.ws?.close();
        this.ws = null;
        
        document.getElementById('chat-screen').classList.add('hidden');
        document.getElementById('auth-screen').classList.remove('hidden');
        
        document.getElementById('phone').value = '';
        document.getElementById('name').value = '';
        
        this.uiManager.showToast('Logged out successfully');
    }
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.chitchat = new ChitChat();
});

// Export for potential use elsewhere
export { ChitChat };