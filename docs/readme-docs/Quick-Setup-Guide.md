## Quick Setup Guide
### Step 1: Check Prerequisites
First, verify you have the required software installed:

```bash
# Check Go version (should be 1.21 or higher)
go version

# Check if Docker is installed (optional, but recommended)
docker --version
docker-compose --version

# Check if PostgreSQL is installed
psql --version

# Check if Redis is installed
redis-cli --version
```
If you don't have these installed, here are quick install commands:

#### For Ubuntu/Debian:

```bash
# Install Go
sudo apt update
sudo apt install -y golang-go

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Install Docker Compose
sudo apt install -y docker-compose

# Install PostgreSQL and Redis
sudo apt install -y postgresql postgresql-contrib redis-server
```

#### For macOS:

```bash
# Install Homebrew if not installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install packages
brew install go
brew install docker docker-compose
brew install postgresql redis
```

#### For Windows:
- Download Go from: https://golang.org/dl/
- Download Docker Desktop: https://www.docker.com/products/docker-desktop
- Download PostgreSQL: https://www.postgresql.org/download/windows/
- Download Redis: https://github.com/tporadowski/redis/releases

### Step 2: Clone and Setup the Project
```bash
# Clone the repository
git clone <repository-url>
cd chitchat

# Initialize Go module
go mod init github.com/msniranjan18/chit-chat
go mod tidy
```

### Step 3: Setup Environment Configuration
```bash
# Copy environment file
cp .env.example .env

# Edit the .env file with your configuration
# For local development, you can use these minimal settings:
cat > .env << 'EOF'
# Server Configuration
PORT=8080
ENV=development

# Database Configuration
DATABASE_URL=postgres://postgres:password@localhost:5432/chitchat?sslmode=disable

# Redis Configuration
REDIS_URL=redis://localhost:6379

# JWT Configuration
JWT_SECRET=your-secret-key-for-development-change-in-production
EOF
```
#### Generate a Strong Secret
Using OpenSSL (Linux/Mac/WSL)
```bash
# Generate a 64-character random secret
openssl rand -base64 64 | tr -d '\n'

# Or 32-character (256-bit) - most common
openssl rand -base64 32

# Generate and copy to clipboard (Mac)
openssl rand -base64 32 | pbcopy

# Generate and copy to clipboard (Linux)
openssl rand -base64 32 | xclip -selection clipboard
```

### Step 4: Start Dependencies
#### Option A: Using Docker (Recommended for beginners)
```bash
# Start only PostgreSQL and Redis with Docker
docker-compose up -d postgres redis

# Check if services are running
docker-compose ps

# You should see:
# postgres   Up   5432/tcp
# redis      Up   6379/tcp
```

#### Option B: Manual Setup
Start PostgreSQL:

```bash
# Start PostgreSQL service
sudo service postgresql start  # Ubuntu/Debian
# or
brew services start postgresql  # macOS

# Create database and user
sudo -u postgres psql -c "CREATE DATABASE chitchat;"
sudo -u postgres psql -c "CREATE USER chitchat_user WITH PASSWORD 'password';"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE chitchat TO chitchat_user;"

# Update .env with:
# DATABASE_URL=postgres://chitchat_user:password@localhost:5432/chitchat?sslmode=disable
```

Start Redis:

```bash
# Start Redis service
sudo service redis-server start  # Ubuntu/Debian
# or
brew services start redis  # macOS
# or
redis-server  # Run in foreground
```

### Step 5: Run the Application
```bash
# Install dependencies
go mod download

# Build and run the application
go run main.go
```

You should see output similar to:

```text
Initializing storage...
Successfully connected to PostgreSQL and Redis
Initializing database schema...
Initializing authentication...
Initializing WebSocket hub...
Setting up routes...
ChitChat server starting on http://localhost:8080
Server is ready to accept connections
```

### Step 6: Test the Application
1.Open your browser: http://localhost:8080
2.You should see the authentication screen:
- Enter a 10-digit Indian phone number (e.g., 9876543210)
- Enter your name
- Click "Continue"

If successful, you'll be redirected to the main chat interface.

### Step 7: API Testing
You can also test the API directly:

```bash
# Test authentication
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"phone":"9876543210","name":"Test User"}'

# Test WebSocket connection (using wscat)
npm install -g wscat

# First get a token from the register response, then:
wscat -c "ws://localhost:8080/ws?token=YOUR_JWT_TOKEN"
```


## Troubleshooting Common Issues
### Issue 1: "Failed to connect to storage"
Error:
Failed to connect to storage: failed to connect to PostgreSQL: dial tcp [::1]:5432: connect: connection refused

Solution:
```bash
# Make sure PostgreSQL is running
sudo service postgresql status

# If not running, start it
sudo service postgresql start

# Check PostgreSQL is listening
sudo netstat -tulpn | grep 5432
```

### Issue 2: "Redis connection failed"
Error:
Redis connection failed: dial tcp 127.0.0.1:6379: connect: connection refused

Solution:
```bash
# Start Redis
sudo service redis-server start

# Or run manually
redis-server

# Test Redis connection
redis-cli ping
# Should respond: PONG
```

### Issue 3: "Permission denied" for database
Error:
pq: permission denied for database "chitchat"

Solution:
```bash
# Grant permissions
sudo -u postgres psql -c "ALTER USER chitchat_user WITH SUPERUSER;"
# Or recreate with proper permissions
sudo -u postgres psql -c "DROP DATABASE IF EXISTS chitchat;"
sudo -u postgres psql -c "CREATE DATABASE chitchat;"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE chitchat TO chitchat_user;"
```

### Issue 4: "Port already in use"
Error:
listen tcp :8080: bind: address already in use

Solution:
```bash
# Find what's using port 8080
sudo lsof -i :8080

# Kill the process
sudo kill -9 <PID>

# Or use a different port in .env
# PORT=8081
```

### Issue 5: "Module not found" errors
Error:
cannot find module providing package github.com/lib/pq
Solution:

```bash
# Clean module cache and download again
go clean -modcache
go mod download
go mod tidy

# Make sure you're in the correct directory
pwd  # Should show /path/to/chitchat
```

### Issue 6: "JWT not initialized"
Error:
JWT not initialized

Solution:
Make sure your .env file has the JWT_SECRET variable set. You can add it:

```bash
echo 'JWT_SECRET=your-secret-key-for-development-change-in-production' >> .env
```

## Docker Full Stack Setup
If you want to run everything with Docker:

```bash
# Create a docker-compose.override.yml for development
cat > docker-compose.override.yml << 'EOF'
version: '3.8'

services:
  app:
    build:
      context: .
      dockerfile: Dockerfile.dev
    ports:
      - "8080:8080"
    volumes:
      - .:/app
      - go-mod-cache:/go/pkg/mod
    environment:
      - DATABASE_URL=postgres://postgres:password@postgres:5432/chitchat?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=development-secret-key
    command: ["air", "-c", ".air.toml"]
    develop:
      watch:
        - action: sync
          path: .
          target: /app
        - action: rebuild
          path: go.mod

volumes:
  go-mod-cache:
EOF

# Start everything
docker-compose up

# The application will be available at http://localhost:8080
```

## Testing with Multiple Users
- To test the chat functionality properly, you can:
- Open two browser windows (or use incognito mode)
- Register two different users with different phone numbers
- Search for each other using phone numbers
- Start a chat and test messaging

Example test users:
- User 1: Phone: 9876543210, Name: Alice
- User 2: Phone: 9876543211, Name: Bob

## Debugging Tips
### Enable Debug Logging
Add to your .env file:

```env
LOG_LEVEL=debug
```

### Check Database Schema
```bash
# Connect to PostgreSQL
psql -U postgres -d chitchat

# List tables
\dt

# Check users table
SELECT * FROM users;

# Check chats table
SELECT * FROM chats;

# Exit
\q
```

### Monitor Redis
```bash
# Connect to Redis
redis-cli

# Monitor all commands
MONITOR

# Check keys
KEYS *

# Exit
quit
```

### View Application Logs
```bash
# If using Docker
docker-compose logs -f app

# If running directly
go run main.go 2>&1 | tee chitchat.log
```

## Production Deployment (Simplified)
For a quick production-like setup:

```bash
# Build the application
go build -ldflags="-s -w" -o chitchat

# Create production .env
cat > .env.production << 'EOF'
PORT=8080
ENV=production
DATABASE_URL=postgres://user:strongpassword@localhost:5432/chitchat?sslmode=require
REDIS_URL=redis://localhost:6379
JWT_SECRET=strong-random-secret-at-least-32-characters
EOF

# Run with production config
export $(cat .env.production | xargs) && ./chitchat
```

## Health Check Endpoints
Once running, test these endpoints:

Health Check: http://localhost:8080/health

API Status: http://localhost:8080/api/auth/verify (requires auth)

WebSocket Test: Connect via wscat as shown above

### If you're still having issues:

Check the logs - Look for any error messages

Verify all services are running:
```bash
# Check PostgreSQL
psql -U postgres -c "SELECT 1;"  # Should return 1

# Check Redis
redis-cli ping  # Should return PONG

# Check Go version
go version
```
Try the Docker method - It's more isolated and less prone to system-specific issues

Create an issue with your error logs and system information

## ✅ Success Checklist
- When everything is working correctly, you should be able to:
- Access http://localhost:8080 in browser
- Register a new user with phone number
- See the chat interface after registration
- Search for other users
- Create a chat with another user
- Send and receive messages in real-time
- See message status (sent/delivered/read)
- See typing indicators
- See online/offline status





### 1. Install golangci-lint:
```bash
# Install globally
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Create basic config
cat > .golangci.yml << 'EOF'
linters:
  enable:
    - errcheck
    - govet
    - gofmt
EOF
```

### 2. Add to your workflow:
```bash
# In Makefile, add:
dev:
    air

lint:
    golangci-lint run ./...
```

### Benefits Summary
- .air.toml: 10x faster development iteration
- .golangci.yml: Consistent, bug-free code
- Dockerfile.dev: Reproducible development environments

These files are best practices in modern Go development and will significantly improve your development experience and code quality!

### File Summary

-----------------------------------------------------------------------------------
File	        |   Purpose		        | DevOnly | Prod  |   When Created
-----------------------------------------------------------------------------------
.air.toml		| Hot reload config	    |   ✅	 | ❌	|   Early development
Dockerfile.dev	| Dev container	        |   ✅	 | ❌	|   Team/docker setup
.golangci.yml	| Code quality	        |   ✅	 | ✅(CI)|   Project start
Dockerfile	    | Production container	|   ❌	 | ✅	|   Deployment ready
Makefile	    | Task automation	    |   ✅	 | ✅	|   Project start
.env	        | Environment vars	    |   ✅	 | ✅	|   Project start