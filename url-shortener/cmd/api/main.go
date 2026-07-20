package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlexTihonow/url-shortener/internal/cache"
	"github.com/AlexTihonow/url-shortener/internal/config"
	"github.com/AlexTihonow/url-shortener/internal/events"
	"github.com/AlexTihonow/url-shortener/internal/handler"
	"github.com/AlexTihonow/url-shortener/internal/logger"
	"github.com/AlexTihonow/url-shortener/internal/repository"
	"github.com/AlexTihonow/url-shortener/internal/service"
)

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(rootCtx, cfg.PostgresDSN)
	if err != nil {
		log.Error("postgres connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := repository.New(pool)
	rcache := cache.New(cfg.RedisAddr, cfg.RedisPass, cfg.CacheTTL)
	defer rcache.Close()

	producer := events.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic, repo, log)
	consumer := events.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, repo, log)
	go consumer.Run(rootCtx)

	svc := service.New(repo, rcache, producer)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	handler.New(svc, cfg.BaseURL).Register(router)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("http server listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server", "err", err)
			stop()
		}
	}()

	<-rootCtx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	_ = consumer.Close()
	_ = producer.Close()
	log.Info("bye")
}
