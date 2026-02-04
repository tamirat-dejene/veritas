# Veritas Microservices Project

This repository contains the microservices architecture for the Veritas project, including Go and Python services orchestrated via Docker Compose.

## Architecture

The system consists of the following microservices:

### Go Services (Port 8080 internal)
- **API Gateway**: Entry point for the frontend/clients.
- **Authentication Service**: Manages user identity and access control.
- **Enterprise Service**: Handles enterprise management and multi-tenancy.
- **Exam Management Service**: Core logic for exam creation and scheduling.
- **Candidate Management Service**: Manages candidate registration and exam interfaces.
- **Payment Service**: Handles subscriptions and payments.

### Python Services (Port 8000 internal)
- **Proctoring Service**: AI-based monitoring (FastAPI).
- **Grading Service**: Automated grading using ML.
- **Reporting Service**: Dashboard and analytics generation.
- **Face Verification Service**: Auxiliary identity verification.

## Getting Started

### Prerequisites
- Docker and Docker Compose installed.

### Running the Project

To build and start all services:

```bash
docker-compose up --build
```

### Accessing Services

| Service | Local Host Port |
|---------|----------------|
| API Gateway | 8080 |
| Auth Service | 8081 |
| Enterprise Service | 8082 |
| Exam Service | 8083 |
| Candidate Service | 8084 |
| Payment Service | 8085 |
| Proctoring Service | 8086 |
| Grading Service | 8087 |
| Reporting Service | 8088 |
| Face Verification | 8089 |
| Prometheus | 9090 |
| Grafana | 3000 |

## Development

Each service is located in the `services/` directory.

- Go services use `go.mod` for dependency management.
- Python services use `requirements.txt`.

### Database
A shared Postgres instance is available on port 5432.
Default credentials: user/password
Database: veritas_db