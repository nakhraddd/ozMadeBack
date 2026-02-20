# OzMade Backend

This repository contains the backend services for the OzMade application, built with Go. It follows Clean Architecture principles to ensure a modular, scalable, and maintainable codebase.

## Architecture Overview

The architecture is designed to be clean, SOLID, and scalable, separating concerns into distinct layers. This makes the system easier to develop, test, and maintain.

*   **Domain Layer (`internal/models`)**: Contains the core business models and entities of the application (e.g., `User`, `Product`, `Order`). This layer is the innermost part of the architecture and has no dependencies on any other layer.
*   **Application Layer (`internal/service`)**: Contains the business logic of the application. Services orchestrate the flow of data between the domain layer and the data persistence layer.
*   **Interface/Adapter Layer (`internal/handler`, `internal/repository`)**: This layer acts as a bridge between the application and the outside world.
    *   `handler`: Handles incoming HTTP requests, validates input, and calls the appropriate services.
    *   `repository`: Implements the data access interfaces defined by the application layer, providing a concrete implementation for data sources like PostgreSQL and Redis.
*   **Infrastructure Layer (`cmd`, `pkg`, `api`)**: The outermost layer, responsible for initializing and starting the application, as well as containing external-facing components.
    *   `cmd`: The main entry point of the application.
    *   `pkg`: Shared libraries and helpers for external services like databases, caches, and cloud storage.
    *   `api`: API definitions, such as OpenAPI specifications.

## Directory Structure

```
.
├── api/                # API contracts (OpenAPI, gRPC protos)
│   └── openapi.yaml
├── cmd/                # Main application entrypoints
│   └── ozmade/
│       └── main.go
├── internal/           # Private application logic
│   ├── auth/           # Authentication handlers
│   ├── chat/           # Chat handlers and logic
│   ├── config/         # Configuration loading
│   ├── middleware/     # Request middleware (e.g., auth checks)
│   ├── models/         # Core business models (User, Product, etc.)
│   ├── order/          # Order handlers and logic
│   ├── product/        # Product handlers and logic
│   ├── repository/     # Data access implementations (Postgres, Redis)
│   ├── router/         # API routing
│   ├── service/        # Business logic services
│   ├── user/           # User handlers and logic
│   └── worker/         # Background workers (e.g., recommendation algorithm)
├── pkg/                # Shared, reusable libraries
│   ├── database/       # Database connections (PostgreSQL)
│   ├── realtime/       # Real-time communication (FCM, WebSockets)
│   ├── redis/          # Redis cache connection
│   └── storage/        # Cloud storage clients (GCS)
├── .dockerignore       # Files to ignore in Docker builds
├── .gitignore          # Files to ignore in Git
├── Dockerfile          # Container definition for the application
├── go.mod              # Go module dependencies
├── Makefile            # Helper commands for building and running
└── README.md           # This file
```

## Infrastructure

The backend is designed to run on a modern cloud infrastructure, leveraging the following services:

*   **Server**: Go application deployed on **Google Cloud Run**, fronted by **Google Cloud API Gateway** for traffic management and security.
*   **Database**:
    *   **Cloud SQL (PostgreSQL)**: Primary relational database for users, products, and orders.
    *   **Cloud Memorystore (Redis)**: Caching for OTPs and the "Most Viewed" product recommendation scores.
    *   **Firestore (Optional)**: Recommended for real-time chat history synchronization with the Android client.
*   **Storage**:
    *   **Google Cloud Storage (GCS)**: Used for storing product images (public bucket) and seller ID cards (private, restricted bucket).
*   **Real-time**:
    *   **Firebase Cloud Messaging (FCM)**: For sending push notifications to the Android client.
    *   **WebSockets**: For active, in-app chat sessions.

## Key Features

### Authentication
Authentication is handled via Firebase Phone Auth. The client sends a Firebase-issued token to the backend, which is then verified using the Firebase Admin SDK to create a user session.

### Seller Verification
Sellers upload their ID cards using a secure **GCP Signed URL** provided by the backend. This allows the client to upload directly to a private GCS bucket without the file passing through the backend server.

### "Most Viewed" Recommendation Algorithm
A background worker (`internal/worker`) periodically calculates a time-decayed score for products based on their view count. The top-ranked products are cached in a Redis Sorted Set for fast retrieval on the main page.

### Chat System
The chat system uses WebSockets for real-time messaging and FCM for push notifications and unread counts.

## Getting Started

### Prerequisites
*   Go (version 1.22 or later)
*   Docker (optional, for containerized deployment)

### Running the Application
To run the application locally, use the `Makefile`:
```sh
make run
```

### Building the Application
To build the application binary:
```sh
make build
```
The binary will be located in the `./out/` directory.

### Docker
To build and run the Docker container:
```sh
docker build -t ozmade-backend .
docker run -p 8080:8080 ozmade-backend
```
