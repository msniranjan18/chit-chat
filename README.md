# ChitChat - WhatsApp-like Messaging Application

![ChitChat Logo](https://img.shields.io/badge/ChitChat-Messaging-blue)
![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)
![License](https://img.shields.io/badge/License-MIT-green)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen)

ChitChat is a feature-rich, real-time messaging application built with Go that replicates core WhatsApp/Telegram functionality. It supports one-to-one chats, group chats, message receipts, typing indicators, and more.
Swagger documentation generated in docs/
Access it at: http://localhost:8080/swagger/index.html

## 🌟 Features

### Core Features
- **One-to-One Chats**: Secure direct messaging between two users
- **Group Chats**: Create groups with multiple participants and admin controls
- **Message Status**: Real-time message status tracking (Sent ✓, Delivered ✓✓, Read ✓✓)
- **Typing Indicators**: See when others are typing
- **Online Presence**: Real-time online/offline status
- **Message History**: Scrollable chat history with pagination
- **Contact Management**: Add, remove, and manage contacts
- **Search**: Search users, chats, and messages

### Advanced Features
- **Message Editing**: Edit sent messages (sender only)
- **Message Deletion**: Delete messages with soft deletion
- **Message Replies**: Reply to specific messages
- **Message Forwarding**: Forward messages to other chats
- **File Attachments**: Support for images, videos, documents (text-only in Phase 1)
- **Multi-Device Support**: Multiple active sessions per user
- **Read Receipts**: Track who has read your messages
- **Chat Archiving**: Archive inactive chats
- **Chat Muting**: Mute notifications for specific chats

### Technical Features
- **Real-time Communication**: WebSocket-based instant messaging
- **Scalable Architecture**: Redis Pub/Sub for multi-instance support
- **Database Persistence**: PostgreSQL for reliable data storage
- **Redis Caching**: Performance optimization with Redis caching
- **JWT Authentication**: Secure token-based authentication
- **RESTful API**: Well-structured API endpoints
- **Responsive UI**: Mobile-friendly web interface
- **Docker Support**: Easy deployment with Docker

## 🏗️ Architecture

### System Architecture
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Client    │◄──►│   Go Server     │◄──►│   PostgreSQL    │
│   (Browser)     │    │   (ChitChat)      │    │   (Primary DB)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │             WebSocket │ HTTP                  │
         │                       │                       │
         │               ┌───────▼─────────┐             │
         └──────────────►│   Redis         │◄────────────┘
                         │   (Cache/PubSub)│
                         └─────────────────┘
```


### Data Flow
1. **Client** ↔ **Go Server** via WebSocket (real-time) and HTTP (REST API)
2. **Go Server** ↔ **PostgreSQL** for persistent data storage
3. **Go Server** ↔ **Redis** for caching and Pub/Sub messaging
4. **Multiple Go Servers** ↔ **Redis Pub/Sub** for horizontal scaling

## Installation

### Prerequisites
- Go 1.21 or higher
- PostgreSQL 15 or higher
- Redis 7 or higher
- Node.js (for frontend assets - optional)

### Quick Start with Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/yourusername/chit-chat.git
cd chit-chat

# Copy environment file
cp .env.example .env

# Edit .env file with your configuration
nano .env

# Start services with Docker Compose
docker-compose up -d

# Access the application
open http://localhost:8080
```

### Manual Installation
#### 1. Database Setup
```sql
-- Create database
CREATE DATABASE chitchat;

-- Create user (optional)
CREATE USER chitchat_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE chitchat TO chitchat_user;
```

#### 2. Redis Setup
```bash
# Install Redis
# On Ubuntu/Debian:
sudo apt-get install redis-server

# On macOS:
brew install redis

# Start Redis
redis-server
```

#### 3. Application Setup
```bash
# Clone the repository
git clone https://github.com/yourusername/chit-chat.git
cd chit-chat

# Install dependencies
go mod download

# Copy environment file
cp .env.example .env

# Configure environment variables
# Edit .env with your database and Redis credentials

# Run database migrations (automatically on first run)
go run main.go

# Start the server
go run main.go

# Or build and run
go build -o chitchat
./chitchat
```

## Configuration
### Environment Variables
Create a .env file in the root directory:
```
# Server Configuration
PORT=8080
ENV=development

# Database Configuration
DATABASE_URL=postgres://postgres:password@localhost:5432/chitchat?sslmode=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_MAX_IDLE_TIME=5m

# Redis Configuration
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT Configuration
JWT_SECRET=your-secret-key-change-in-production
JWT_EXPIRATION=168h  # 7 days

# WebSocket Configuration
WS_READ_BUFFER_SIZE=1024
WS_WRITE_BUFFER_SIZE=1024
WS_WRITE_WAIT=10s
WS_PONG_WAIT=60s
WS_PING_PERIOD=54s
WS_MAX_MESSAGE_SIZE=10485760  # 10MB

# Rate Limiting
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_BURST=100
```

## API Documentation
### Authentication
#### Register/Login
```http
POST /api/auth/register
Content-Type: application/json

{
  "phone": "9876543210",
  "name": "John Doe"
}
```
Response:
```json
{
  "token": "jwt_token_here",
  "user": {
    "id": "uuid",
    "phone": "9876543210",
    "name": "John Doe",
    "status": "Hey there! I am using ChitChat",
    "last_seen": "2024-01-15T10:30:00Z",
    "created_at": "2024-01-15T10:30:00Z"
  },
  "expires_at": "2024-01-22T10:30:00Z"
}
```

#### Verify Token
```http
GET /api/auth/verify
Authorization: Bearer <jwt_token>
```

### Chats
#### Get User Chats
```http
GET /api/chats
Authorization: Bearer <jwt_token>
```

#### Create Chat
```http
POST /api/chats
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "type": "direct",
  "user_ids": ["user_uuid_here"]
}

// For group chat
{
  "type": "group",
  "name": "Family Group",
  "user_ids": ["user1_uuid", "user2_uuid"]
}
```

#### Get Chat Details
```http
GET /api/chats/{chat_id}
Authorization: Bearer <jwt_token>
```

### Messages
#### Send Message
```http
POST /api/messages
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "chat_id": "chat_uuid",
  "content": "Hello World!",
  "content_type": "text"
}
```

#### Get Messages
```http
GET /api/messages?chat_id={chat_id}&offset=0&limit=50
Authorization: Bearer <jwt_token>
```

#### Update Message Status
```http
POST /api/messages/status
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "message_id": "msg_uuid",
  "status": "read"  // or "delivered"
}
```

### Users
#### Search Users
```http
GET /api/users/search?q=john
Authorization: Bearer <jwt_token>
```

#### Get Contacts
```http
GET /api/contacts
Authorization: Bearer <jwt_token>
```

### WebSocket Connection
#### Connect to WebSocket endpoint:
```javascript
const ws = new WebSocket(`ws://localhost:8080/ws?token=${jwt_token}`);
```

#### WebSocket Message Format:
```json
{
  "type": "message",
  "room_id": "chat_id",
  "sender": "user_id",
  "payload": {
    "content": "Hello!",
    "content_type": "text"
  }
}
```

#### Message Types:
- **message:** New chat message

- **typing:** Typing indicator

- **presence:** User online/offline status

- **status_update:** Message status update

- **chat_update:** Chat information update

## Running the Application
### Development Mode
```bash
# With hot reload (using air)
air

# Or with go run
go run main.go
```

### Production Mode
```bash
# Build the application
go build -ldflags="-s -w" -o chitchat

# Run with production environment
export ENV=production
./chitchat
```

### Using PM2 (Process Manager)
```bash
# Install PM2
npm install -g pm2

# Start application
pm2 start chitchat --name chitchat

# Monitor
pm2 monit

# Logs
pm2 logs chitchat
```

## Docker Deployment
### Single Container
```bash
# Build Docker image
docker build -t chitchat .

# Run container
docker run -d \
  -p 8080:8080 \
  --name chitchat \
  -e DATABASE_URL=postgres://user:pass@host:5432/db \
  -e REDIS_URL=redis://host:6379 \
  chitchat
```

### Docker Compose (Full Stack)
```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chitchat
spec:
  replicas: 3
  selector:
    matchLabels:
      app: chitchat
  template:
    metadata:
      labels:
        app: chitchat
    spec:
      containers:
      - name: chitchat
        image: chitchat:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: chitchat-secrets
              key: database-url
        - name: REDIS_URL
          value: "redis://redis-service:6379"
```

## Development
### Code Style
```bash
# Format code
go fmt ./...

# Run vet
go vet ./...

# Run tests
go test ./...
```

### Adding New Features
1. Add new model in **pkg/models/**
2. Add storage methods in **pkg/store/**
3. Add handlers in **pkg/handlers/**
4. Add routes in **pkg/routes/routes.go**
5. Update frontend in **static/app.js**

### Database Migrations
The application automatically creates tables on first run. For production migrations, consider using a migration tool like [golang-migrate](https://github.com/golang-migrate/migrate).

## Monitoring and Logging
### Health Check
```http
GET /health
```
Response:

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "services": {
    "database": "connected",
    "redis": "connected",
    "websocket": "running"
  }
}
```

### Logging Levels
- **INFO:** General application events
- **WARN:** Non-critical issues
- **ERROR:** Critical errors
- **DEBUG:** Detailed debugging information (development only)

## Security
### Implemented Security Measures
1. **JWT Authentication:** Token-based authentication with expiration
2. **Passwordless:** Phone number based authentication (Indian market focus)
3. **Input Validation:** All user inputs are validated
4. **SQL Injection Prevention:** Parameterized queries
5. **XSS Protection:** Output escaping in templates
6. **CORS Configuration:** Restricted origins in production
7. **Rate Limiting:** Prevents brute force attacks
8. **HTTPS Enforcement:** In production environments

### Security Best Practices
- Use strong JWT secret in production
- Enable HTTPS with valid SSL certificate
- Regular security updates
- Database backup strategy
- Monitoring for suspicious activities


## Scaling
### Horizontal Scaling
- Multiple Instances: Run multiple ChitChat instances behind a load balancer
- Redis Pub/Sub: Handles inter-instance communication
- Database Connection Pooling: Efficient database connections
- Caching Strategy: Redis caching for frequent queries

### Performance Optimization
- Connection Pooling: For both PostgreSQL and Redis
- Message Batching: Debounced database writes
- Caching: Frequently accessed data in Redis
- WebSocket Optimization: Efficient message broadcasting

## Deployment Checklist
- Update .env with production values
- Set ENV=production
- Use strong JWT_SECRET
- Configure HTTPS (nginx/caddy reverse proxy)
- Set up database backups
- Configure monitoring (Prometheus/Grafana)
- Set up logging (ELK stack)
- Configure firewall rules
- Set up CI/CD pipeline
- Load testing
- Backup strategy

## License
This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments
- Built with [Go](https://golang.org/)
- Database: [PostgreSQL](https://www.postgresql.org/)
- Cache/PubSub: [Redis](https://redis.io/)
- WebSocket: [gorilla/websocket](https://github.com/gorilla/websocket)
- JWT: [golang-jwt/jwt](https://github.com/golang-jwt/jwt)

## Support
Issues: GitHub Issues

## Roadmap
### Phase 2 (Planned)
- Voice and video calls (WebRTC)
- End-to-end encryption
- Message reactions (emojis)
- Stories/Status updates
- Voice messages
- Payment integration (UPI)
- Bot API
- Channel support


### Phase 3
- Mobile apps (React Native/Flutter)
- Desktop app (Electron)
- Sticker marketplace
- Advanced group permissions
- Message scheduling
- AI-powered features

#### Happy chatting!
