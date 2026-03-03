# Architecture Diagram

## Mermaid Diagram (render at mermaid.live or in GitHub)

```mermaid
flowchart TB
    Client["HTTP Client<br/>(curl, Postman, etc.)"]

    subgraph Docker["Docker Compose Network"]
        subgraph PythonService["Python Service · FastAPI · :8000"]
            direction TB
            PAuth["Auth Middleware"]
            PLog["Logging Middleware"]
            PRouter["Routers"]
            PRepo["Repository Layer"]
            PAuth --> PLog --> PRouter --> PRepo
        end

        subgraph GoService["Go Service · Gin · :8080"]
            direction TB
            GAuth["Auth Middleware"]
            GLog["Logging Middleware"]
            GHandler["Handlers"]
            GRepo["Repository Layer"]
            GAuth --> GLog --> GHandler --> GRepo
        end

        subgraph Storage["Data Storage"]
            PDB[("sensors-python.db")]
            GDB[("sensors-go.db")]
            Seed["sensors.json"]
        end
    end

    Client -->|"Bearer token"| PAuth
    Client -->|"Bearer token"| GAuth

    PRepo --> PDB
    GRepo --> GDB
    Seed -.->|"seed on startup"| PDB
    Seed -.->|"seed on startup"| GDB
```

## ASCII Diagram (for terminals/plain text)

```
                         HTTP Client
                    (curl, Postman, etc.)
                    │                   │
                    │ Bearer token      │ Bearer token
                    ▼                   ▼
╔═══════════════════════════════════════════════════════╗
║              DOCKER COMPOSE NETWORK                   ║
║                                                       ║
║  ┌─────────────────────┐   ┌─────────────────────┐   ║
║  │ Python · FastAPI     │   │ Go · Gin            │   ║
║  │ :8000                │   │ :8080               │   ║
║  │                      │   │                     │   ║
║  │  ┌────────────────┐  │   │  ┌────────────────┐ │   ║
║  │  │ Auth Middleware │  │   │  │ Auth Middleware │ │   ║
║  │  └───────┬────────┘  │   │  └───────┬────────┘ │   ║
║  │          ▼           │   │          ▼          │   ║
║  │  ┌────────────────┐  │   │  ┌────────────────┐ │   ║
║  │  │ Log Middleware  │  │   │  │ Log Middleware  │ │   ║
║  │  └───────┬────────┘  │   │  └───────┬────────┘ │   ║
║  │          ▼           │   │          ▼          │   ║
║  │  ┌────────────────┐  │   │  ┌────────────────┐ │   ║
║  │  │ Routers        │  │   │  │ Handlers       │ │   ║
║  │  └───────┬────────┘  │   │  └───────┬────────┘ │   ║
║  │          ▼           │   │          ▼          │   ║
║  │  ┌────────────────┐  │   │  ┌────────────────┐ │   ║
║  │  │ Repository     │  │   │  │ Repository     │ │   ║
║  │  └───────┬────────┘  │   │  └───────┬────────┘ │   ║
║  └──────────┼──────────┘   └──────────┼──────────┘   ║
║             ▼                         ▼              ║
║  ┌─────────────────────────────────────────────────┐ ║
║  │                 DATA STORAGE                     │ ║
║  │  ┌──────────────┐ ┌────────────┐ ┌───────────┐  │ ║
║  │  │sensors-python│ │ sensors-go │ │sensors.json│  │ ║
║  │  │    .db       │ │    .db     │ │ (seed)     │  │ ║
║  │  └──────────────┘ └────────────┘ └───────────┘  │ ║
║  └─────────────────────────────────────────────────┘ ║
╚═══════════════════════════════════════════════════════╝
```

## Key Components

| Component | Description |
|-----------|-------------|
| **Docker Compose Network** | Isolates services; all requests require Bearer token |
| **Auth Middleware** | Validates `Authorization: Bearer <token>` header on every request |
| **Logging Middleware** | Adds correlation ID (`X-Correlation-ID`) for request tracing |
| **Routers/Handlers** | Route definitions and business logic for CRUD operations |
| **Repository Layer** | Data access abstraction (injected via DI) |
| **SQLite Databases** | Separate database per service to avoid conflicts |
| **Seed Data** | `sensors.json` loaded on startup if database is empty |

## Request Flow

The sequence diagram below shows how a typical authenticated request flows through the middleware chain. Both services follow this identical pattern.

```mermaid
sequenceDiagram
    actor Client
    participant Auth as Auth Middleware
    participant Log as Logging Middleware
    participant Router as Router / Handler
    participant Repo as Repository
    participant DB as SQLite

    Client->>+Auth: GET /sensors<br/>Authorization: Bearer <token>
    Auth->>Auth: Validate token
    alt Invalid or missing token
        Auth-->>Client: 401 Unauthorized
    end
    Auth->>+Log: Forward request
    Log->>Log: Generate correlation ID
    Log->>+Router: Forward request
    Router->>+Repo: GetAll()
    Repo->>+DB: SELECT * FROM sensors
    DB-->>-Repo: rows
    Repo-->>-Router: []Sensor
    Router-->>-Log: 200 OK + JSON
    Log->>Log: Log request + duration
    Log-->>-Auth: Response
    Auth-->>-Client: 200 OK + JSON body
```

### Steps

1. Client sends HTTP request with `Authorization: Bearer <token>` header
2. Auth Middleware validates token → returns 401 if invalid
3. Logging Middleware generates/extracts correlation ID, logs request
4. Router/Handler processes request, calls Repository
5. Repository executes SQL against SQLite database
6. Response returned with correlation ID in logs
