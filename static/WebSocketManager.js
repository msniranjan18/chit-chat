// WebSocketManager.js - Contains WebSocket related functions
class WebSocketManager {
    constructor(appInstance) {
        this.app = appInstance
    }

    connectWebSocket() {
        if (!this.app.token) return;

        const wsUrl = `ws://${window.location.host}/ws?token=${this.app.token}`;
        this.app.ws = new WebSocket(wsUrl);

        this.app.ws.onopen = () => {
            console.log('WebSocket connected');
            this.app.uiManager.updateUserStatus(true);
        };

        this.app.ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                this.handleWebSocketMessage(data);
            } catch (error) {
                console.error('Error parsing WebSocket message:', error);
            }
        };

        this.app.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        this.app.ws.onclose = () => {
            console.log('WebSocket disconnected');
            this.app.uiManager.updateUserStatus(false);
            
            setTimeout(() => this.connectWebSocket(), 5000);
        };
    }

    handleWebSocketMessage(data) {
        switch (data.type) {
            case 'message':
                this.app.chatManager.handleIncomingMessage(data.payload);
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
                this.app.chatManager.handleChatUpdate(data.payload);
                break;
        }
    }

    handleTypingIndicator(data) {
        if (data.chat_id === this.app.activeChat && data.user_id !== this.app.user?.id) {
            const typingIndicator = document.getElementById('typing-indicator');
            if (data.is_typing) {
                typingIndicator.classList.add('active');
            } else {
                typingIndicator.classList.remove('active');
            }
        }
    }

    handlePresenceUpdate(data) {
        const { user_id, is_online } = data;
        
        if (this.app.activeChat) {
            const chat = this.app.chats.get(this.app.activeChat);
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
        
        const contactElement = document.querySelector(`.chat-item[data-user-id="${user_id}"]`);
        if (contactElement) {
            const timestampEl = contactElement.querySelector('.timestamp');
            if (timestampEl) {
                timestampEl.textContent = is_online ? 'Online' : 'Offline';
            }
        }
    }

    handleStatusUpdate(data) {
        const { message_id, status } = data;
        const messageElement = document.querySelector(`.message[data-message-id="${message_id}"]`);
        if (messageElement) {
            const statusElement = messageElement.querySelector('.message-status');
            if (statusElement) {
                statusElement.textContent = this.app.uiManager.getMessageStatusIcon(status);
            }
        }
    }
}

// Export for use in other modules
export { WebSocketManager };

window.WebSocketManager = WebSocketManager;
