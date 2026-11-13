# GopherSocial API

<div style="margin-bottom:20px">
  <a href="#"><img alt="go-version" src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go"/></a>
  <a href="#"><img alt="postgres-version" src="https://img.shields.io/badge/PostgreSQL-16.3-336791?style=flat-square&logo=postgresql"/></a>
  <a href="#"><img alt="redis-version" src="https://img.shields.io/badge/Redis-7-DC382D?style=flat-square&logo=redis"/></a>
  <a href="#"><img alt="docker-version" src="https://img.shields.io/badge/Docker-Compose-2496ED?style=flat-square&logo=docker"/></a>
</div>

> **A highly scalable, production-ready social media backend built in Go, engineered for high-throughput user feeds, strict concurrency controls, and resilient caching.**

# Table of Contents

- [The Goals of This Project](#the-goals-of-this-project)
- [Technologies & Libraries](#technologies--libraries)
- [Core Architecture](#core-architecture)
- [Key Engineering Features](#key-engineering-features)
- [TODO](#todo)
- [How to Run](#how-to-run)

## The Goals of This Project

- Architect a modular REST API in Go using `chi` and explicit interface boundaries, maintaining strict separation between HTTP transport, business logic, and data access.
- Implement a thread-safe, fixed-window rate limiter using `sync.Mutex` and background eviction goroutines to protect endpoints while preventing unbounded memory growth.
- Reduce PostgreSQL read overhead on authenticated routes via a Redis cache-aside layer designed with automatic database fallbacks during cache outages.
- Guarantee reliable, eventually consistent background task execution using the Transactional Outbox pattern within ACID database transactions.
- Maintain consistent feed query latency at scale by replacing traditional `OFFSET` queries with cursor-based (keyset) pagination.
- Standardize the local development environment by orchestrating PostgreSQL and Redis via Docker Compose.

## Technologies & Libraries

- ✔️ **Go** - Core backend runtime and concurrency.
- ✔️ **go-chi/chi** - Lightweight, idiomatic HTTP router.
- ✔️ **PostgreSQL** - Primary relational database for persistent, transactional storage.
- ✔️ **Redis** - In-memory store for user identity caching.
- ✔️ **golang-jwt** - Stateless JWT generation and signature verification.
- ✔️ **uber-go/zap** - Structured, leveled logging.
- ✔️ **Docker Compose** - Container orchestration for local PostgreSQL and Redis instances.

## Core Architecture

Instead of distributing into microservices prematurely, this system utilizes a modular monolith architecture separated by strict domain boundaries:

- `API & Middleware Layer`: The central entry point. Handles request parsing, fixed-window rate limiting, JWT validation, and context injection.
- **Cache Layer:** Sits between authentication middleware and persistent storage, serving user profiles from Redis using a cache-aside pattern with automatic fallback to PostgreSQL on cache misses or connection errors.
- `Storage Layer (Postgres)`: Manages the relational domain schema (Users, Posts, Comments, Followers). Interfaces isolate the SQL logic from the HTTP handlers.
- **Background Workers:** Asynchronous goroutines that poll the outbox table to process side effects (such as email notifications) independently of the HTTP request lifecycle.

## Key Engineering Features

1. **Concurrency-Safe Rate Limiting:** Built a fixed-window IP rate limiter using `sync.Mutex` and a background cleanup goroutine to evict stale client buckets, protecting endpoints while keeping memory usage bounded.
2. **Fail-Open Identity Caching:** Integrated a Redis cache-aside layer into the JWT authentication middleware to reduce PostgreSQL read load, routing requests directly to the primary database if Redis becomes unreachable.
3. **Transactional Outbox Processing:** Coupled primary database writes and outbox event records within single ACID transactions so asynchronous workers can reliably dispatch notifications across service restarts.

## TODO

1. **Bookmark Feature:** Add a `post_bookmarks` table, storage methods, and a `POST /v1/posts/{postID}/bookmark` endpoint using authenticated user context.
2. **API Containerization:** Package the Go service with a multi-stage `Dockerfile` and integrate it into `docker-compose.yaml`.
3. **Deployment:** Deploy the containerized stack to a cloud VM.

## How to Run

### Prerequisites

Ensure you have **Go (1.26+)**, **Docker**, and **Docker Compose** installed.

### 1. Boot the Infrastructure (PostgreSQL & Redis)

Navigate to the project root and start the database and cache in detached mode using Docker:

```bash
docker compose up -d
```

### 2. Verify Infrastructure Services

Ensure both the postgres-db and redis-cache containers are running successfully:

```bash
docker ps
```

### 3. Start the API Gateway

Download the Go dependencies and start the local HTTP server:

```bash
go mod tidy
go run ./cmd/api
```

### 4. Test the API

The API will be available at http://localhost:8080. You can test the rate limiter by aggressively hitting a public or protected endpoint:

```bash
curl -i http://localhost:8080/v1/health
```
