package main

import (
	"log"
	"net/http"

	"github.com/andrelks/objectvault/internal/config"
	"github.com/andrelks/objectvault/internal/db"
	"github.com/andrelks/objectvault/internal/handler"
	"github.com/andrelks/objectvault/internal/metadata"
	"github.com/andrelks/objectvault/internal/service"
	"github.com/andrelks/objectvault/internal/storage"
)

func main() {
	log.Println("Starting ObjectVault initialization...")

	// 1. Load config
	cfg := config.Load()

	// 2. Connect to Postgres database (no auto-create schema)
	dbConn, err := db.Connect(cfg.DBConnStr)
	if err != nil {
		log.Fatalf("Fatal error connecting to database: %v", err)
	}
	defer dbConn.Close()
	log.Println("Successfully connected to Postgres metadata database.")

	// 3. Initialize Storage Engine (ensures storage directory exists)
	storageEngine, err := storage.NewLocalDiskStorage(cfg.StorageDir)
	if err != nil {
		log.Fatalf("Fatal error initializing storage engine: %v", err)
	}
	log.Printf("Storage engine initialized. Payload directory: %s", cfg.StorageDir)

	// 4. Initialize Metadata Store
	metaStore := metadata.NewPostgresMetadataStore(dbConn)

	// 5. Initialize Services
	bucketService := service.NewBucketService(metaStore)
	objectService := service.NewObjectService(storageEngine, metaStore)

	// 6. Initialize Presentation Layer (Chi router/handlers)
	apiHandler := handler.NewAPIHandler(bucketService, objectService)
	router := apiHandler.Router()

	// 7. Start HTTP Server
	log.Printf("ObjectVault API listening on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}