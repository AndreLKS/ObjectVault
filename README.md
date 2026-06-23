# ObjectVault

An S3-inspired object storage platform built in Go to explore distributed systems, storage architecture, data consistency, and cloud infrastructure concepts.

## Features

* Object upload and download API
* Bucket management
* Object versioning
* Local storage engine with indexed metadata
* Token-based authentication
* Data consistency controls
* Redis caching layer
* Checksum and integrity validation
* Multi-node replication
* Metrics and observability
* Distributed storage simulation

## Architecture Goals

ObjectVault aims to simulate the core components of a modern cloud object storage service while remaining approachable for learning and experimentation.

The primary goals are:

* Understand how object storage systems work internally
* Design scalable storage architectures
* Implement consistency and durability mechanisms
* Explore metadata indexing strategies
* Experiment with replication and fault tolerance
* Practice cloud-native engineering patterns
* Learn observability and monitoring techniques

## Tech Stack

### Core

* Go 1.25+
* Chi Router
* PostgreSQL
* Redis
* Docker

### Observability

* Prometheus
* Grafana
* OpenTelemetry

### Testing

* Go Testing Package
* Testcontainers

## High-Level Architecture

```text
Client
   |
REST API
   |
+----------------------+
| ObjectVault API      |
+----------------------+
      |
      +---- Metadata Layer (PostgreSQL)
      |
      +---- Cache Layer (Redis)
      |
      +---- Storage Layer (Filesystem)
      |
      +---- Replication Layer
```

## Roadmap

### Phase 1 — Core Storage

* Object upload
* Object download
* Object listing
* Metadata persistence
* Docker environment
* Automated tests

### Phase 2 — Storage Features

* Bucket management
* Object versioning
* Authentication
* Checksum validation
* Integrity verification

### Phase 3 — Performance

* Redis caching
* Cache invalidation strategies
* Performance benchmarks

### Phase 4 — Distributed Systems

* Multi-node replication
* Event-driven synchronization
* Failure recovery
* Consistency strategies

### Phase 5 — Observability

* Structured logging
* Prometheus metrics
* Grafana dashboards
* OpenTelemetry tracing

## Why This Project?

Modern cloud platforms rely heavily on object storage systems for durability, scalability, and performance.

ObjectVault is designed as a learning-oriented infrastructure project focused on understanding the engineering challenges behind systems such as Amazon S3, Google Cloud Storage, MinIO, and Azure Blob Storage.

The goal is not to replace existing storage solutions, but to gain practical experience with distributed systems, cloud-native architecture, data durability, and backend infrastructure engineering.
