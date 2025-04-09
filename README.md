# Movie VS Backend

A GoLang API application using Gin framework and MongoDB for a movie battle game.

## Prerequisites

- Go 1.21 or higher
- MongoDB running locally on port 27017
- Git

## Setup

1. Clone the repository
2. Install dependencies:
```bash
go mod tidy
```

3. Make sure MongoDB is running locally on port 27017

4. Start the server:
```bash
go run .
```

The server will start on port 8080 by default.

## API Endpoints

### Public Endpoints

- `POST /api/register` - Register a new user
- `POST /api/login` - Login and get JWT token
- `POST /api/logout` - Logout (client-side)

### Protected Endpoints (Requires JWT Token)

- `GET /api/battle` - Get a pair of movies for battle
- `GET /api/leaderboard` - Get top 20 users
- `POST /api/battle` - Submit battle winner

## Authentication

Include the JWT token in the Authorization header for protected endpoints:
```
Authorization: Bearer <your-token>
```
# MovieVs_Back_End
