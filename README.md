# Object Storage Simulator

A lightweight object storage service inspired by Amazon S3, built to explore distributed systems, storage architecture, data consistency, and cloud infrastructure concepts.

## Features

* Object upload and download API
* Simple bucket management
* Object versioning
* Local storage engine with indexed metadata
* Token-based authentication
* Data consistency controls
* Optional caching layer
* Checksum and integrity validation
* Multi-node replication (planned)
* Observability and metrics (planned)

## Architecture Goals

This project aims to simulate the core components of a modern cloud object storage platform while remaining simple enough for educational and portfolio purposes.

The main objectives are:

* Understand how object storage systems work internally
* Explore metadata management strategies
* Implement consistency and durability mechanisms
* Experiment with replication and fault tolerance
* Practice scalable backend architecture patterns

## Tech Stack

* PHP 8.x
* Laravel
* PostgreSQL
* Redis
* Docker
* REST API

## Roadmap

### Phase 1

* Authentication
* Bucket creation
* Object upload
* Object download
* Metadata storage

### Phase 2

* Object versioning
* File integrity validation
* Redis caching
* Metrics collection

### Phase 3

* Multi-node replication
* Consistency strategies
* Failure recovery
* Distributed storage simulation

## Why This Project?

Cloud object storage services are a foundational component of modern infrastructure platforms. This project is designed to provide hands-on experience with concepts commonly found in systems such as Amazon S3, Google Cloud Storage, and Azure Blob Storage.

The focus is on learning storage internals, distributed system design, and infrastructure engineering principles rather than building a production-ready replacement.
