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

## API Endpoints

Here is a list of available API endpoints for testing with Postman.

### Public Endpoints

#### `GET /products`
*   **Description**: Retrieves a list of all products. Supports filtering and pagination.
*   **Query Parameters**:
    *   `type` (string, optional): Filter products by type (e.g., "electronics", "clothing").
    *   `page` (integer, optional): Page number for pagination (default: 1).
    *   `limit` (integer, optional): Number of items per page (default: 10).
*   **Request Body**: None
*   **Responses**:
    *   `200 OK`:
        ```json
        [
            {
                "ID": 1,
                "Title": "Product A",
                "Description": "Description of Product A",
                "Type": "electronics",
                "Cost": 199.99,
                "Address": "123 Main St",
                "WhatsAppLink": "https://wa.me/1234567890",
                "ViewCount": 150,
                "AverageRating": 4.5,
                "ImageName": "https://signed-url-to-image.com/productA.jpg",
                "CreatedAt": "2023-01-01T12:00:00Z",
                "Comments": []
            }
        ]
        ```
    *   `500 Internal Server Error`: If there's an issue fetching products.

#### `GET /products/:id`
*   **Description**: Retrieves details for a single product by its ID.
*   **Path Parameters**:
    *   `id` (integer, required): The unique identifier of the product.
*   **Request Body**: None
*   **Responses**:
    *   `200 OK`:
        ```json
        {
            "ID": 1,
            "Title": "Product A",
            "Description": "Description of Product A",
            "Type": "electronics",
            "Cost": 199.99,
            "Address": "123 Main St",
            "WhatsAppLink": "https://wa.me/1234567890",
            "ViewCount": 150,
            "AverageRating": 4.5,
            "ImageName": "https://signed-url-to-image.com/productA.jpg",
            "CreatedAt": "2023-01-01T12:00:00Z",
            "Comments": [
                {"ID": 1, "UserID": 101, "ProductID": 1, "Rating": 5, "Text": "Great product!"}
            ]
        }
        ```
    *   `404 Not Found`: If the product with the given ID does not exist.
    *   `500 Internal Server Error`: If there's an issue fetching the product.

#### `POST /products/:id/view`
*   **Description**: Increments the view count for a specific product and updates its trending score.
*   **Path Parameters**:
    *   `id` (integer, required): The unique identifier of the product.
*   **Request Body**: None
*   **Responses**:
    *   `200 OK`: No body, just a success status.
    *   `404 Not Found`: If the product with the given ID does not exist.

#### `GET /products/trending`
*   **Description**: Retrieves a list of products currently trending based on view counts and a time-decay algorithm.
*   **Request Body**: None
*   **Responses**:
    *   `200 OK`:
        ```json
        [
            {
                "ID": 2,
                "Title": "Trending Product X",
                "Description": "Description of Trending Product X",
                "Type": "fashion",
                "Cost": 50.00,
                "Address": "456 Oak Ave",
                "WhatsAppLink": "https://wa.me/0987654321",
                "ViewCount": 300,
                "AverageRating": 4.8,
                "ImageName": "https://signed-url-to-image.com/productX.jpg",
                "CreatedAt": "2023-01-15T10:00:00Z",
                "Comments": []
            }
        ]
        ```
    *   `500 Internal Server Error`: If there's an issue fetching trending products.

### Authenticated Endpoints (User)
*All endpoints in this section require an `Authorization` header with a valid Firebase ID token: `Authorization: Bearer <firebase_token>`.*

#### `POST /auth/sync`
*   **Description**: Synchronizes user data from a Firebase ID token with the backend database. Creates a new user record if one doesn't exist for the given Firebase UID.
*   **Request Body**: None (Firebase token is extracted from the `Authorization` header).
*   **Responses**:
    *   `200 OK`:
        ```json
        {
            "user_id": 1,
            "profile": {
                "ID": 1,
                "FirebaseUID": "firebase_uid_123",
                "PhoneNumber": "+1234567890",
                "Email": "user@example.com",
                "Address": "",
                "Role": "buyer",
                "IsSeller": false,
                "CreatedAt": "2023-01-01T12:00:00Z"
            }
        }
        ```
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `500 Internal Server Error`: If there's an issue creating or fetching the user.

#### `POST /products/:id/comments`
*   **Description**: Allows an authenticated user to post a comment and rating on a product.
*   **Path Parameters**:
    *   `id` (integer, required): The unique identifier of the product.
*   **Request Body**:
    ```json
    {
        "rating": 4,
        "text": "This product is pretty good, fast delivery!"
    }
    ```
*   **Responses**:
    *   `201 Created`:
        ```json
        {
            "ID": 1,
            "UserID": 101,
            "ProductID": 1,
            "Rating": 4,
            "Text": "This product is pretty good, fast delivery!",
            "CreatedAt": "2023-01-20T14:30:00Z"
        }
        ```
    *   `400 Bad Request`: If the request body is invalid.
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `500 Internal Server Error`: If there's an issue creating the comment.

#### `POST /products/:id/report`
*   **Description**: Allows an authenticated user to report a product for inappropriate content or other issues.
*   **Path Parameters**:
    *   `id` (integer, required): The unique identifier of the product.
*   **Request Body**:
    ```json
    {
        "reason": "Misleading description and images."
    }
    ```
*   **Responses**:
    *   `201 Created`: No body, just a success status.
    *   `400 Bad Request`: If the request body is invalid.
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `500 Internal Server Error`: If there's an issue creating the report.

#### `GET /profile`
*   **Description**: Retrieves the authenticated user's profile information.
*   **Request Body**: None
*   **Responses**:
    *   `200 OK`:
        ```json
        {
            "ID": 1,
            "FirebaseUID": "firebase_uid_123",
            "PhoneNumber": "+1234567890",
            "Email": "user@example.com",
            "Address": "789 Pine St",
            "Role": "buyer",
            "IsSeller": false,
            "CreatedAt": "2023-01-01T12:00:00Z"
        }
        ```
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `404 Not Found`: If the user's profile cannot be found (should not happen after `/auth/sync`).

#### `PATCH /profile`
*   **Description**: Updates the authenticated user's profile information.
*   **Request Body**:
    ```json
    {
        "email": "new_email@example.com",
        "address": "New Address, City, Country"
    }
    ```
    (Fields are optional; only provided fields will be updated)
*   **Responses**:
    *   `200 OK`: Returns the updated user profile.
        ```json
        {
            "ID": 1,
            "FirebaseUID": "firebase_uid_123",
            "PhoneNumber": "+1234567890",
            "Email": "new_email@example.com",
            "Address": "New Address, City, Country",
            "Role": "buyer",
            "IsSeller": false,
            "CreatedAt": "2023-01-01T12:00:00Z"
        }
        ```
    *   `400 Bad Request`: If the request body is invalid.
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `404 Not Found`: If the user's profile cannot be found.

#### `POST /profile/favorites/:id`
*   **Description**: Toggles a product's favorite status for the authenticated user. If already favorited, it unfavorites; otherwise, it favorites.
*   **Path Parameters**:
    *   `id` (integer, required): The unique identifier of the product.
*   **Request Body**: None
*   **Responses**:
    *   `200 OK`:
        ```json
        {"status": "added"}
        ```
        or
        ```json
        {"status": "removed"}
        ```
    *   `400 Bad Request`: If the product ID is invalid.
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `500 Internal Server Error`: If there's an issue updating the favorite status.

#### `GET /profile/favorites`
*   **Description**: Retrieves a list of products that the authenticated user has marked as favorite.
*   **Request Body**: None
*   **Responses**:
    *   `200 OK`:
        ```json
        [
            {
                "ID": 3,
                "Title": "Favorite Item",
                "Description": "User's favorite product",
                "Type": "home",
                "Cost": 75.00,
                "Address": "101 Market St",
                "WhatsAppLink": "https://wa.me/1122334455",
                "ViewCount": 80,
                "AverageRating": 4.2,
                "ImageName": "https://signed-url-to-image.com/favorite.jpg",
                "CreatedAt": "2023-02-01T09:00:00Z",
                "Comments": []
            }
        ]
        ```
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `500 Internal Server Error`: If there's an issue fetching favorites.

#### `GET /profile/orders`
*   **Description**: Retrieves a list of all orders placed by the authenticated user.
*   **Request Body**: None
*   **Responses**:
    *   `200 OK`:
        ```json
        [
            {
                "ID": 1,
                "UserID": 1,
                "ProductID": 1,
                "Quantity": 2,
                "TotalCost": 399.98,
                "Status": "completed",
                "CreatedAt": "2023-01-10T11:00:00Z"
            }
        ]
        ```
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `500 Internal Server Error`: If there's an issue fetching orders.

#### `POST /seller/register`
*   **Description**: Allows an authenticated user to apply to become a seller.
*   **Request Body**: None
*   **Responses**:
    *   `200 OK`:
        ```json
        {"message": "Seller application submitted"}
        ```
    *   `400 Bad Request`: If the user is already a seller.
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `404 Not Found`: If the user's profile cannot be found.
    *   `500 Internal Server Error`: If there's an issue creating the seller application.

#### `GET /seller/upload-id-url`
*   **Description**: Provides a signed URL for the authenticated user to securely upload their ID card directly to Google Cloud Storage.
*   **Request Body**: None
*   **Responses**:
    *   `200 OK`:
        ```json
        {"upload_url": "https://signed-url-to-gcs-bucket/seller_ids/user_id.jpg?Signature=..."}
        ```
        The client should then use this URL to perform a `PUT` request with the image data.
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `500 Internal Server Error`: If there's an issue generating the signed URL.

### Authenticated Endpoints (Seller)
*All endpoints in this section require an `Authorization` header with a valid Firebase ID token: `Authorization: Bearer <firebase_token>`. Additionally, the authenticated user must be a registered seller.*

#### `GET /seller/products`
*   **Description**: Retrieves a list of all products owned by the authenticated seller.
*   **Request Body**: None
*   **Responses**:
    *   `200 OK`:
        ```json
        [
            {
                "ID": 10,
                "Title": "Seller's Product 1",
                "Description": "Description by seller",
                "Type": "handmade",
                "Cost": 25.00,
                "Address": "Seller's Address",
                "WhatsAppLink": "https://wa.me/sellerphone",
                "ViewCount": 50,
                "AverageRating": 4.0,
                "ImageName": "https://signed-url-to-image.com/seller_product1.jpg",
                "CreatedAt": "2023-03-01T10:00:00Z",
                "Comments": []
            }
        ]
        ```
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `403 Forbidden`: If the user is not a registered seller.
    *   `404 Not Found`: If the seller profile cannot be found.
    *   `500 Internal Server Error`: If there's an issue fetching products.

#### `POST /seller/products`
*   **Description**: Creates a new product for the authenticated seller.
*   **Request Body**:
    ```json
    {
        "name": "New Handmade Craft",
        "description": "A beautifully crafted item.",
        "price": 45.99,
        "type": "crafts",
        "address": "123 Craft Lane",
        "image_url": "https://example.com/new_craft_image.jpg"
    }
    ```
*   **Responses**:
    *   `201 Created`: Returns the newly created product.
        ```json
        {
            "ID": 11,
            "SellerID": 1,
            "Name": "New Handmade Craft",
            "Description": "A beautifully crafted item.",
            "Price": 45.99,
            "Type": "crafts",
            "Address": "123 Craft Lane",
            "ImageURL": "https://example.com/new_craft_image.jpg",
            "CreatedAt": "2023-03-05T11:00:00Z"
        }
        ```
    *   `400 Bad Request`: If the request body is invalid.
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `403 Forbidden`: If the user is not a registered seller.
    *   `404 Not Found`: If the seller profile cannot be found.
    *   `500 Internal Server Error`: If there's an issue creating the product.

#### `PUT /seller/products/:id`
*   **Description**: Updates an existing product owned by the authenticated seller.
*   **Path Parameters**:
    *   `id` (integer, required): The unique identifier of the product to update.
*   **Request Body**:
    ```json
    {
        "name": "Updated Handmade Craft",
        "description": "An even more beautifully crafted item.",
        "price": 49.99,
        "type": "crafts",
        "address": "456 Artisan Way",
        "image_url": "https://example.com/updated_craft_image.jpg"
    }
    ```
    (All fields are required for a PUT, or use PATCH for partial updates)
*   **Responses**:
    *   `200 OK`: Returns the updated product.
        ```json
        {
            "ID": 11,
            "SellerID": 1,
            "Name": "Updated Handmade Craft",
            "Description": "An even more beautifully crafted item.",
            "Price": 49.99,
            "Type": "crafts",
            "Address": "456 Artisan Way",
            "ImageURL": "https://example.com/updated_craft_image.jpg",
            "CreatedAt": "2023-03-05T11:00:00Z"
        }
        ```
    *   `400 Bad Request`: If the request body is invalid.
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `403 Forbidden`: If the user is not a registered seller.
    *   `404 Not Found`: If the product is not found or not owned by the seller.
    *   `500 Internal Server Error`: If there's an issue updating the product.

#### `DELETE /seller/products/:id`
*   **Description**: Deletes a product owned by the authenticated seller.
*   **Path Parameters**:
    *   `id` (integer, required): The unique identifier of the product to delete.
*   **Request Body**: None
*   **Responses**:
    *   `200 OK`:
        ```json
        {"message": "Product deleted"}
        ```
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `403 Forbidden`: If the user is not a registered seller.
    *   `404 Not Found`: If the product is not found or not owned by the seller.
    *   `500 Internal Server Error`: If there's an issue deleting the product.

#### `GET /seller/profile`
*   **Description**: Retrieves the authenticated seller's profile information, including product count.
*   **Request Body**: None
*   **Responses**:
    *   `200 OK`:
        ```json
        {
            "name": "seller@example.com",
            "status": "approved",
            "total_products": 5
        }
        ```
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `403 Forbidden`: If the user is not a registered seller.
    *   `404 Not Found`: If the seller profile cannot be found.

#### `PATCH /seller/profile`
*   **Description**: Updates the authenticated seller's profile information. Currently supports updating the profile picture (though the actual upload is handled via a signed URL).
*   **Request Body**:
    ```json
    {
        "profile_picture": "https://example.com/new_seller_profile.jpg"
    }
    ```
*   **Responses**:
    *   `200 OK`:
        ```json
        {"message": "Profile updated"}
        ```
    *   `400 Bad Request`: If the request body is invalid.
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `403 Forbidden`: If the user is not a registered seller.
    *   `404 Not Found`: If the seller profile cannot be found.

#### `GET /seller/chats`
*   **Description**: Retrieves a list of all chat conversations associated with the authenticated seller.
*   **Request Body**: None
*   **Responses**:
    *   `200 OK`:
        ```json
        [
            {
                "ID": 1,
                "CreatedAt": "2023-04-01T10:00:00Z",
                "UpdatedAt": "2023-04-01T10:00:00Z",
                "DeletedAt": null,
                "SellerID": 1,
                "BuyerID": 2,
                "Messages": []
            }
        ]
        ```
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `403 Forbidden`: If the user is not a registered seller.
    *   `404 Not Found`: If the seller profile cannot be found.
    *   `500 Internal Server Error`: If there's an issue fetching chats.

#### `GET /seller/chats/:chat_id/messages`
*   **Description**: Retrieves all messages within a specific chat conversation for the authenticated seller.
*   **Path Parameters**:
    *   `chat_id` (integer, required): The unique identifier of the chat conversation.
*   **Request Body**: None
*   **Responses**:
    *   `200 OK`:
        ```json
        [
            {
                "ID": 1,
                "CreatedAt": "2023-04-01T10:01:00Z",
                "UpdatedAt": "2023-04-01T10:01:00Z",
                "DeletedAt": null,
                "ChatID": 1,
                "SenderID": 2,
                "Content": "Hello, is this product available?"
            },
            {
                "ID": 2,
                "CreatedAt": "2023-04-01T10:02:00Z",
                "UpdatedAt": "2023-04-01T10:02:00Z",
                "DeletedAt": null,
                "ChatID": 1,
                "SenderID": 1,
                "Content": "Yes, it is! How can I help you?"
            }
        ]
        ```
    *   `401 Unauthorized`: If the `Authorization` header is missing or the token is invalid.
    *   `403 Forbidden`: If the user is not a registered seller or does not have access to this chat.
    *   `500 Internal Server Error`: If there's an issue fetching messages.
