Usage Examples
Basic Development Workflow
bash
# 1. Setup project
make setup

# 2. Start development with hot reload
make dev

# 3. Or run without hot reload
make run

# 4. Run tests
make test

# 5. Run linter
make lint

# 6. Format code
make fmt
Docker Development
bash
# Start all services with Docker Compose
make docker-compose-up

# View logs
make docker-compose-logs

# Stop services
make docker-compose-down
Build and Release
bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Create a release
make release
Database Operations
bash
# Start databases
make db-start

# Connect to PostgreSQL
make db-psql

# Connect to Redis
make db-redis

# Run migrations
make db-migrate
Quality Assurance
bash
# Run all quality checks
make qa

# Run security scan
make security-scan

# Run CI pipeline
make ci
Common Make Commands
Command	Description
make help	Show all available commands
make setup	Complete project setup
make dev	Start development with hot reload
make test	Run all tests
make lint	Run linter
make build	Build application
make docker-compose-up	Start all services with Docker
make clean	Clean build artifacts
make monitor	Monitor system resources
make health	Check system health
