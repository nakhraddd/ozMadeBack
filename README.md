# OzMade Backend

This repository contains the backend services for the OzMade application, built with Go. It follows Clean Architecture principles to ensure a modular, scalable, and maintainable codebase.

## Architecture Overview

The architecture is designed to be clean, SOLID, and scalable, separating concerns into distinct layers. This makes the system easier to develop, test, and maintain.

*   **Domain Layer (`internal/models`)**: Contains the core business models and entities of the application (e.g., `User`, `Product`, `Order`).
*   **Application Layer (`internal/service`)**: Contains the business logic of the application. Services orchestrate the flow of data between the domain layer and the data persistence layer.
*   **Interface/Adapter Layer (`internal/handlers`, `internal/repository`)**: This layer acts as a bridge between the application and the outside world.
    *   `handlers`: Handles incoming HTTP requests, validates input, and calls the appropriate services.
    *   `repository`: Implements the data access interfaces defined by the application layer, providing a concrete implementation for data sources like PostgreSQL and Redis.
*   **Infrastructure Layer (`cmd`, `pkg`)**: The outermost layer, responsible for initializing and starting the application, as well as containing external-facing components.
    *   `cmd`: The main entry point of the application.
    *   `pkg`: Shared libraries and helpers for external services like databases, caches, and cloud storage.

## Directory Structure

```
.
├── cmd/                # Main application entrypoints
│   └── ozmade/
│       └── main.go
├── internal/           # Private application logic
│   ├── auth/           # Authentication logic and middleware
│   ├── database/       # DB connection and migrations
│   ├── handlers/       # API route handlers
│   ├── middleware/     # Request middleware (e.g., auth checks)
│   ├── models/         # Core business models (User, Product, etc.)
│   ├── repository/     # Data access implementations (Postgres, Redis)
│   ├── routes/         # API routing definitions
│   ├── service/        # Domain-specific business logic
│   └── services/       # Infrastructure-level services (Search, GCS, etc.)
├── pkg/                # Shared, reusable libraries
│   ├── database/       # DB helpers
│   ├── realtime/       # FCM and WebSockets
│   ├── redis/          # Redis helpers
│   └── storage/        # Cloud storage helpers (GCS)
├── config/             # Environment configuration
├── README.md           # This file
```

## Infrastructure

*   **Server**: Go application using **Gin Web Framework**.
*   **Database**: **PostgreSQL** (GORM) for primary storage, **Redis** for caching and trending scores.
*   **Search**: Custom search index with support for full-text queries and price filtering.
*   **Storage**: **Google Cloud Storage (GCS)** with V4 Signed URLs for secure uploads and protected assets.
*   **Real-time**: **WebSockets** for live chat and **FCM (Firebase Cloud Messaging)** for push notifications.

## Database Schema

### Users (`users`)
*   `id`, `firebase_uid`, `phone_number`, `email`, `name`, `address`, `role`, `is_seller`, `fcm_token`

### Sellers (`sellers`)
*   `id`, `user_id`, `status`, `id_card`, `pickup_enabled`, `pickup_address`, `pickup_time`, `free_delivery_enabled`, `delivery_center_lat/lng/radius`, `intercity_enabled`

### Products (`products`)
*   `id`, `seller_id`, `title`, `description`, `cost`, `view_count`, `average_rating`, `image_name`, `images` (JSON), `weight`, `height/width/depth_cm`, `composition`, `youtube_url`, `categories` (JSON)

### Orders (`orders`)
*   `id`, `user_id`, `product_id`, `quantity`, `total_cost`, `status`, `delivery_type`, `shipping_address_text`, `confirm_code`

### Chats (`chats`)
*   `id`, `seller_id`, `buyer_id`, `product_id`, `deleted_by_buyer`, `deleted_by_seller`

## Key API Endpoints

### Public
*   `GET /categories` - List categories
*   `GET /ads` - Banners/Ads
*   `GET /products` - List products
*   `GET /products/:id` - Product details
*   `GET /products/:id/reviews` - Product reviews & ratings summary
*   `GET /products/search` - Full-text search
*   `GET /products/trending` - Trending products
*   `GET /sellers/:id` - Seller public profile & products
*   `GET /sellers/:id/reviews` - Aggregate reviews for a seller

### Buyer (Authenticated)
*   `POST /auth/sync` - Sync Firebase user
*   `GET /profile` - User profile
*   `POST /products/:id/comments` - Post review/rating
*   `POST /profile/favorites/:id` - Toggle favorite
*   `GET /orders` - Buyer orders
*   `POST /orders` - Create order
*   `POST /chats` - Initiate chat
*   `GET /chats` - List chats
*   `DELETE /chats/:id` - Hide chat

### Seller (Authenticated)
*   `POST /seller/register` - Apply for seller account
*   `GET /seller/products` - Manage products
*   `GET /seller/orders` - Manage seller orders
*   `GET /seller/delivery` - Shipping settings
*   `GET /seller/upload-product-photo-url` - Signed URL for product photos

## Real-time Notifications
The backend uses **WebSockets** for active sessions and **FCM** for background delivery. Notification types:
*   `chat_message`: New message in a conversation.
*   `order_update`: Changes in order status.

## Image Handling
All images are stored in GCS. The backend generates **V4 Signed URLs** with a short expiry (15m) for all GET requests to ensure security.
*   Path: `oz-made/products/{filename}`
*   Path: `oz-made/seller_ids/{filename}`
