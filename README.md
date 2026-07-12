# Authentication Service

A production-ready authentication service built with Go, designed for SaaS applications and portfolio demonstrations.

## Features

- 🔐 **JWT Authentication** - Secure access and refresh token-based authentication
- 🔄 **Refresh Token Rotation** - Automatic token rotation for enhanced security
- 👥 **Role-Based Access Control (RBAC)** - Support for user, moderator, and admin roles
- 🗄️ **PostgreSQL Persistence** - Robust data storage with GORM ORM
- ⚡ **Redis Caching** - Token blacklisting, rate limiting, and session management
- 📝 **Structured Logging** - Production-ready logging with Zap
- 🐳 **Dockerized Deployment** - Easy setup with Docker Compose
- 📊 **Swagger Documentation** - Auto-generated API documentation
- ✅ **Comprehensive Testing** - Unit and integration tests with 80%+ coverage target
- 🔒 **Security First** - bcrypt hashing, CORS, rate limiting, account lockout

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.24+ |
| HTTP Framework | Gin |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| ORM | GORM |
| Authentication | JWT + Refresh Tokens |
| Password Hashing | bcrypt |
| Logging | Zap |
| Configuration | Viper |
| Testing | Go test + Testify |
| Containers | Docker, Docker Compose |
| CI | GitHub Actions |

## Quick Start

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- Make (optional)

### Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/kaushlender/auth-service.git
cd auth-service

# Start all services
make docker-up

# Or manually
docker-compose up -d --build
```

The API will be available at `http://localhost:8080`.

### Local Development

```bash
# Start dependencies
docker-compose up -d postgres redis

# Install dependencies
make deps

# Run the application
make run
```

## API Endpoints

### Authentication

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|:------------:|
| POST | `/api/v1/auth/register` | Register a new user | No |
| POST | `/api/v1/auth/login` | Login with credentials | No |
| POST | `/api/v1/auth/refresh` | Refresh access token | No |
| POST | `/api/v1/auth/logout` | Logout and revoke tokens | Yes |

### Users

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|:------------:|
| GET | `/api/v1/users/me` | Get current user profile | Yes |

### System

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check endpoint |

## API Usage Examples

### Register a User

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com",
    "password": "SecurePass123!"
  }'
```

### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "SecurePass123!"
  }'
```

### Access Protected Endpoint

```bash
curl http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer <access_token>"
```

### Refresh Token

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "<refresh_token>"
  }'
```

## Project Structure

```
auth-service/
├── cmd/
│   └── server/          # Application entry point
├── internal/
│   ├── cache/           # Redis caching layer
│   ├── config/          # Configuration management
│   ├── handler/         # HTTP handlers
│   ├── logger/          # Structured logging
│   ├── middleware/       # HTTP middleware
│   ├── model/           # Data models
│   ├── repository/      # Database repositories
│   ├── service/         # Business logic
│   ├── token/           # JWT management
│   └── validator/       # Input validation
├── migrations/          # Database migrations
├── docs/               # Documentation
├── tests/              # Test files
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

## Configuration

Configuration is managed via environment variables (prefixed with `AUTH_`) or a `config.yaml` file.

### Environment Variables

See `.env.example` for all available configuration options.

Key configuration variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `AUTH_SERVER_PORT` | Server port | `8080` |
| `AUTH_DATABASE_HOST` | PostgreSQL host | `localhost` |
| `AUTH_DATABASE_PORT` | PostgreSQL port | `5432` |
| `AUTH_REDIS_HOST` | Redis host | `localhost` |
| `AUTH_REDIS_PORT` | Redis port | `6379` |
| `AUTH_JWT_ACCESS_SECRET` | JWT access token secret | (change in production) |
| `AUTH_JWT_REFRESH_SECRET` | JWT refresh token secret | (change in production) |
| `AUTH_APP_ENVIRONMENT` | Environment (development/production) | `development` |

## Development

### Available Make Commands

```bash
make build          # Build the binary
make run            # Run the application
make test           # Run tests
make test-coverage  # Run tests with coverage report
make lint           # Run linter
make deps           # Download dependencies
make docker-up      # Start all services
make docker-down    # Stop all services
make swagger        # Generate Swagger docs
```

## Security Features

- ✅ bcrypt password hashing (cost factor 12)
- ✅ JWT signing with strong secrets
- ✅ Refresh token rotation
- ✅ Token blacklisting on logout
- ✅ Rate limiting (60 requests/minute)
- ✅ Account lockout after 5 failed attempts
- ✅ CORS configuration
- ✅ Security headers (CSP, HSTS, XSS protection)
- ✅ Request validation
- ✅ SQL injection protection (GORM)
- ✅ Audit logging

## Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

## Deployment

### Docker Deployment

```bash
# Build and start services
docker-compose up -d --build

# View logs
docker-compose logs -f api
```

### Production Considerations

1. Change JWT secrets to strong random values
2. Enable HTTPS with a reverse proxy (nginx, Traefik)
3. Set `AUTH_APP_ENVIRONMENT=production`
4. Configure proper CORS origins
5. Set up database backups
6. Configure monitoring and alerting
7. Use a secrets manager for sensitive data

## License

MIT

## Author

Kaushlendra Kumar Verma