package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"behappy-delivery-order-guard/internal/config"
	"behappy-delivery-order-guard/internal/ingestion"
	"behappy-delivery-order-guard/internal/printer"
	"behappy-delivery-order-guard/internal/watcher"

	"go.uber.org/zap"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	logger, err := config.InitLogger(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			log.Printf("Failed to sync logger: %v", err)
		}
	}(logger)

	logger.Info("Starting Delivery Order Guard System",
		zap.String("version", "1.0.0"),
		zap.String("env", cfg.Environment),
	)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 初始化 Redis 客户端
	redisClient, err := config.InitRedis(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize Redis", zap.Error(err))
	}
	defer redisClient.Close()

	// 创建订单通道
	orderChan := make(chan *ingestion.Order, cfg.OrderChannelBuffer)

	// 创建 WaitGroup 用于等待所有 goroutine 完成
	var wg sync.WaitGroup

	// 初始化订单接收器
	ingester := ingestion.NewIngester(cfg, logger, orderChan)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ingester.Start(ctx)
	}()

	// 初始化打印队列（Worker Pool）
	printerPool := printer.NewPrinterPool(cfg, logger, redisClient, orderChan)
	wg.Add(1)
	go func() {
		defer wg.Done()
		printerPool.Start(ctx)
	}()

	// 初始化超时监控器
	timeoutWatcher := watcher.NewTimeoutWatcher(cfg, logger, redisClient)
	wg.Add(1)
	go func() {
		defer wg.Done()
		timeoutWatcher.Start(ctx)
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	logger.Info("System is running. Press Ctrl+C to stop.")
	<-sigChan

	logger.Info("Shutting down...")
	cancel()

	// 等待所有 goroutine 完成
	wg.Wait()
	logger.Info("System stopped gracefully")
}
