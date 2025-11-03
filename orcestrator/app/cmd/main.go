package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"orchestrator/app/config"
	"orchestrator/app/usecase"
	"orchestrator/internal/infrastructure/llm"
	"orchestrator/internal/infrastructure/metrics"
	"orchestrator/internal/infrastructure/store/filesystem"
	mongorepo "orchestrator/internal/infrastructure/store/mongodb"
	"orchestrator/internal/infrastructure/transport"
	"orchestrator/internal/infrastructure/validator"
)

func main() {
	// logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// load config
	cfg := loadConfig()

	// Connect to MongoDB
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	mongoDBName := getEnv("MONGO_DB", "orchestrator")

	mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer mongoCancel()
	mongoClient, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		logger.Error("mongo connect failed", "err", err)
		log.Fatalf("mongo connect: %v", err)
	}
	if err := mongoClient.Ping(mongoCtx, nil); err != nil {
		logger.Error("mongo ping failed", "err", err)
		log.Fatalf("mongo ping: %v", err)
	}
	logger.Info("connected to mongo", "uri", mongoURI)
	db := mongoClient.Database(mongoDBName)

	// Repositories
	jobRepo := mongorepo.NewMongoJobRepo(db)
	configRepo := mongorepo.NewMongoConfigRepo(db)
	configFileRepo, err := filesystem.NewFileRepository("./deployments")
	if err != nil {
		log.Printf("err init file repo: %s", err)
		return
	}
	// Usecases / services
	jobSvc := usecase.NewJobService(jobRepo, configRepo, usecase.NewTerraformDeployer())
	configFileSvc := usecase.NewConfigService(configRepo)

	// LLM client
	llmClient := llm.NewAmveraGenerator(
		cfg.LLM.APIKey,
		cfg.LLM.BaseURL,
		cfg.LLM.Model,
	)

	configGenerator := usecase.NewConfigGeneratorService(
		jobRepo,
		configRepo,
		configFileRepo,
		llmClient,
		*validator.NewTerraformAnalyzer(), // static validator
		nil,                               // sandbox validator
		nil,                               // security validator
		logger,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configGenerator.Start(ctx) // фоновый воркер

	// terraform deployer
	// Transport (HTTP handlers)
	handler := transport.NewOrchestratorHandler(
		jobSvc,
		configFileSvc,
		logger,
	)

	// Router and server
	r := mux.NewRouter()
	handler.RegisterRoutes(r)
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)(r)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      corsHandler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		logger.Info("starting metrics server on :2112")
		metrics.StartMetricsServer()
	}()

	// Start HTTP server

	go func() {
		logger.Info("starting HTTP server", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server failed", "err", err)
			cancel()
		}
	}()

	// OS signal handling for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-stop:
		logger.Info("shutdown signal received")
	case <-ctx.Done():
		logger.Info("context cancelled")
	}

	// Shutdown sequence
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	logger.Info("shutting down http server")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown error", "err", err)
	}

	logger.Info("disconnecting mongo")
	if err := mongoClient.Disconnect(shutdownCtx); err != nil {
		logger.Error("mongo disconnect error", "err", err)
	}

	logger.Info("service stopped")
}

func loadConfig() *config.Config {
	cfg := &config.Config{
		Server: config.HTTPServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         8080,
			ReadTimeout:  30 * time.Minute,
			WriteTimeout: 30 * time.Minute,
		},
		LLM: config.LLMConfig{
			APIKey:    getEnv("AMVERA_API_KEY", ""),
			BaseURL:   getEnv("AMVERA_BASE_URL", "https://kong-proxy.yc.amvera.ru/api/v1/models/gpt"),
			Model:     getEnv("AMVERA_MODEL", "gpt-5"),
			MaxTokens: 4000,
			Timeout:   60 * time.Minute,
		},
		Mongo: config.MongoConfig{
			URI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
			Database: getEnv("MONGO_DB", "orchestrator"),
		},
		FileRepo: config.FileRepoConfig{
			ConfigDir: getEnv("CONFIG_DIR", "./deployments"),
		},
	}

	if cfg.LLM.APIKey == "" {
		log.Fatal("AMVERA_API_KEY env variable is required")
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
