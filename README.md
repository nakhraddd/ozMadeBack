# OzMade Backend

This repository contains the backend services for the OzMade application, built with Go. It follows Clean Architecture principles to ensure a modular, scalable, and maintainable codebase.

## Architecture Overview

The architecture is designed to be clean, SOLID, and scalable, separating concerns into distinct layers. This makes the system easier to develop, test, and maintain.

*   **Domain Layer (`internal/models`)**: Contains the core business models and entities of the application (e.g., `User`, `Product`, `Order`). This layer is the innermost part of the architecture and has no dependencies on any other layer.
*   **Application Layer (`internal/service`, `internal/services`)**: Contains the business logic of the application. Services orchestrate the flow of data between the domain layer and the data persistence layer.
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
│       └── main.go     # Application bootstrap
├── internal/           # Private application logic
│   ├── auth/           # Firebase Authentication logic and middleware
│   ├── database/       # Database connection initialization and migrations
│   ├── handlers/       # HTTP route handlers (Controllers)
│   ├── middleware/     # Request middleware (Logging, Auth, CORS)
│   ├── models/         # GORM models and business entities
│   ├── repository/     # Data access layer (Postgres, Redis)
│   ├── routes/         # Router setup and group definitions
│   ├── service/        # Domain-specific business services
│   └── services/       # Infrastructure services (GCS, Search, Trending)
├── pkg/                # Shared, reusable libraries
│   ├── database/       # DB driver helpers
│   ├── realtime/       # WebSockets and FCM logic
│   ├── redis/          # Redis client and cache helpers
│   └── storage/        # Google Cloud Storage client
├── config/             # Environment variables and configuration
├── go.mod              # Dependency management
└── README.md           # This file
```

## Infrastructure

The backend is designed to run on modern cloud infrastructure, leveraging the following services:

*   **Server**: Go application built with the **Gin Web Framework**. Optimized for high concurrency and low latency.
*   **Database**:
    *   **PostgreSQL**: Primary relational database for all persistent data (Users, Products, Orders, Reviews).
    *   **Redis**: High-speed caching for OTPs and the real-time "Trending" product rankings.
*   **Search**: Custom-built search index with support for full-text matching, category filtering, and price range queries.
*   **Storage**: **Google Cloud Storage (GCS)** for persistent file storage.
    *   `oz-made/products/`: Public-facing product images.
    *   `oz-made/seller_ids/`: Private seller verification documents.
*   **Real-time Communication**:
    *   **WebSockets**: Bi-directional communication for instant chat messaging.
    *   **Firebase Cloud Messaging (FCM)**: Push notifications for orders and messages when the app is in the background.

## Database Schema

### Users (`users`)
Stores core account data for all application users.
*   `id`: Unique identifier (uint)
*   `firebase_uid`: Unique ID from Firebase Auth (string)
*   `phone_number`: Primary contact (string)
*   `email`: User email (string)
*   `name`: Display name (string)
*   `role`: "buyer", "seller", or "admin"
*   `is_seller`: Quick flag for seller status
*   `fcm_token`: Device token for push notifications

### Sellers (`sellers`)
Extended profile for users who sell products.
*   `id`: Primary identifier (different from UserID)
*   `user_id`: Reference to `users.id`
*   `status`: Application status ("pending", "approved", "rejected")
*   `id_card`: Path to the uploaded verification image
*   **Delivery Settings**:
    *   `pickup_enabled`: If customers can collect in person.
    *   `pickup_address`: The physical collection point.
    *   `free_delivery_enabled`: If the seller offers local delivery.
    *   `delivery_radius_km`: The maximum range for free delivery.
    *   `intercity_enabled`: If shipping between cities is supported.

### Products (`products`)
Full product catalog data.
*   `id`: Primary identifier
*   `seller_id`: Reference to `sellers.id`
*   `title`: Product name
*   `description`: Full markdown-capable description
*   `cost`: Price in local currency
*   `view_count`: Popularity metric
*   `average_rating`: Mean of all user ratings
*   `image_name`: Main image filename (signed on retrieval)
*   `images`: JSON array of additional image filenames
*   `dimensions`: `weight`, `height_cm`, `width_cm`, `depth_cm`
*   `categories`: JSON array of category tags (e.g., "handmade", "food")

### Orders (`orders`)
Transaction records between buyers and sellers.
*   `id`: Primary identifier
*   `status`: Lifecycle state ("PENDING_SELLER", "CONFIRMED", "READY_OR_SHIPPED", "COMPLETED", "CANCELLED")
*   `delivery_type`: "PICKUP", "MY_DELIVERY", or "INTERCITY"
*   `confirm_code`: 4-digit security code for order hand-off

### Chats & Messages
*   **Chats**: Tracks conversations per product between a specific buyer and seller. Supports "Soft Delete" (hiding for one party).
*   **Messages**: Individual chat entries with timestamps and sender roles.

### Comments (`comments`)
User reviews and ratings for products.
*   `product_id`: The target product.
*   `rating`: Numerical score (1.0 to 5.0).
*   `text`: Optional written review.

## Key Features

### 1. Advanced Image Security
The system uses **V4 Signed URLs** for all image access. Raw filenames are stored in the database (e.g., `1_123.jpg`), and the backend automatically:
1.  Prepends the correct directory (`products/` or `seller_ids/`).
2.  Generates a time-limited signature (15 minutes).
3.  Returns a secure URL that only works for that specific session.

### 2. Search & Discovery
*   **Trending Algorithm**: A background worker periodically calculates scores based on views and age (time-decay) to keep the "Trending" section fresh.
*   **Full-Text Search**: Users can search across titles, descriptions, and categories simultaneously.

### 3. Comprehensive Reviews
*   **Product Reviews**: Detailed breakdown of ratings per product.
*   **Seller Reputation**: Aggregate scoring that sums up a seller's performance across all their listings.

---

## API Documentation

### Public Endpoints (No Auth Required)

#### `GET /products`
Retrieves a paginated list of products.
*   **Query Params**: `page` (int), `limit` (int), `type` (string).
*   **Behavior**: Automatically resolves all `ImageName` and `Images` paths to signed URLs.

#### `GET /products/:id`
Retrieves full details for a product.
*   **Response**: Includes `Delivery` settings and `Seller` basic info.

#### `GET /products/:id/reviews`
Retrieves the rating summary and list of reviews for a product.
```json
{
  "summary": {
    "product_id": 2,
    "average_rating": 4.8,
    "ratings_count": 150,
    "reviews_count": 120
  },
  "reviews": [
    { "id": 1, "user_name": "Alex", "rating": 5.0, "text": "Great!", "created_at": "..." }
  ]
}
```

#### `GET /sellers/:id`
Returns a seller's public profile, including their active listings and delivery capabilities.

#### `GET /sellers/:id/reviews`
Aggregates all ratings from all products owned by the seller.

#### `GET /products/search`
Filters the entire catalog. Supports `q`, `min_cost`, `max_cost`, `category`, and `type`.

---

### Buyer Endpoints (Auth Token Required)

#### `POST /auth/sync`
Call this after Firebase Login to ensure the backend has the latest user profile.

#### `POST /products/:id/comments`
Submit a rating and review.
*   **Body**: `{ "rating": 5, "text": "Amazing!" }`
*   **Note**: Triggers background calculation of the product's average rating.

#### `POST /orders`
Creates a new order. Requires `product_id`, `quantity`, and `delivery_type`.

#### `DELETE /chats/:chat_id`
Soft-deletes a chat. The conversation will no longer appear in your list, but remains for the other person.

---

### Seller Endpoints (Auth + Seller Role Required)

#### `POST /seller/products`
Creates a new listing.
*   **Important**: Use the `fileUrl` from the upload-url endpoint as the `image_url`.

#### `GET /seller/upload-product-photo-url`
Returns a signed `PUT` URL.
1.  Request the URL.
2.  Perform a `PUT` from the client directly to GCS.
3.  Send the `fileUrl` back to the server in the product creation request.

#### `PATCH /seller/delivery`
Updates how your products are shipped or picked up.

#### `POST /seller/orders/:id/confirm`
Changes order status to `CONFIRMED` and generates a `confirm_code` for the buyer.
