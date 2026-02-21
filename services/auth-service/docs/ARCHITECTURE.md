# Architecture

The Auth Service follows **Clean Architecture** principles, ensuring separation of concerns, testability, and independence from frameworks and external tools.

## Layers

### 1. Domain Layer (`internal/domain`)
The core of the application. Contains business entities and logic.
- **Entities**: `User`, `RefreshToken`.
- **Ports**: Interfaces that the infrastructure layer must implement (`UserRepository`, `TokenService`).
- **Logic**: Sentinel errors and role permissions.

### 2. Application Layer (`internal/usecase`)
Orchestrates the flow of data between the domain and external layers.
- **Use Cases**: `LoginUseCase`, `RefreshUseCase`, `LogoutUseCase`.
- **Responsibilities**: Validating business rules, calling repository ports, and coordinating token generation.

### 3. Infrastructure/Interface Adapter Layer
Bridges the application to external tools.

#### Repository (`internal/repository/postgres`)
Implementation of persistence ports using PostgreSQL.
- SQL query management.
- Data mapping between DB rows and domain entities.

#### Token Service (`internal/infrastructure/token`)
Logic for security tokens.
- **JWT Service**: Signed HMAC-SHA256 access tokens.
- **Refresh Service**: Secure random token generation and hashing.

#### Transport (`internal/handler`, `internal/router`)
HTTP presentation layer.
- **Gin Web Framework**: Handles routing and middleware.
- **Handlers**: Translates HTTP requests to use case calls and formats JSON responses.

## Request Flow

1. **Client** sends request to **Gin Router**.
2. **Middleware** (RequestID, Recovery) processes the request.
3. **Handler** parses input and calls the appropriate **Use Case**.
4. **Use Case** performs logic, interacting with **Domain** entities and **Repository** ports.
5. **Repository** interacts with the **PostgreSQL** database.
6. **Use Case** returns results to **Handler**.
7. **Handler** formats the final JSON response for the **Client**.
