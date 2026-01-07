package printer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"behappy-delivery-order-guard/internal/config"
	"behappy-delivery-order-guard/internal/ingestion"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// PrinterPool 打印队列（Worker Pool 模式）
type PrinterPool struct {
	cfg        *config.Config
	logger     *zap.Logger
	redis      *redis.Client
	orderChan  <-chan *ingestion.Order
	workerChan chan *ingestion.Order
	wg         sync.WaitGroup
}

// NewPrinterPool 创建打印队列
func NewPrinterPool(cfg *config.Config, logger *zap.Logger, redis *redis.Client, orderChan <-chan *ingestion.Order) *PrinterPool {
	return &PrinterPool{
		cfg:        cfg,
		logger:     logger,
		redis:      redis,
		orderChan:  orderChan,
		workerChan: make(chan *ingestion.Order, cfg.PrinterWorkers),
	}
}

// Start 启动打印队列
func (p *PrinterPool) Start(ctx context.Context) {
	p.logger.Info("Printer pool started",
		zap.Int("workers", p.cfg.PrinterWorkers),
	)

	// 启动 Worker Pool
	for i := 0; i < p.cfg.PrinterWorkers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	// 从订单通道接收订单并分发到 Worker
	go p.distribute(ctx)

	// 等待所有 worker 完成
	p.wg.Wait()
	p.logger.Info("Printer pool stopped")
}

// distribute 分发订单到 Worker
func (p *PrinterPool) distribute(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			close(p.workerChan)
			return
		case order := <-p.orderChan:
			select {
			case p.workerChan <- order:
			case <-ctx.Done():
				close(p.workerChan)
				return
			}
		}
	}
}

// worker 工作协程，处理打印任务
func (p *PrinterPool) worker(ctx context.Context, id int) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case order, ok := <-p.workerChan:
			if !ok {
				return
			}
			p.processOrder(ctx, order, id)
		}
	}
}

// processOrder 处理单个订单
func (p *PrinterPool) processOrder(ctx context.Context, order *ingestion.Order, workerID int) {
	startTime := time.Now()

	p.logger.Info("Processing order for printing",
		zap.Int("worker_id", workerID),
		zap.String("order_id", order.ID),
		zap.String("platform", order.Platform),
	)

	// 模拟打印过程（实际场景中这里会调用打印机 API）
	time.Sleep(500 * time.Millisecond)

	// 将订单保存到 Redis（用于超时监控）
	orderJSON, err := json.Marshal(order)
	if err != nil {
		p.logger.Error("Failed to marshal order", zap.Error(err))
		return
	}

	orderKey := fmt.Sprintf("order:%s", order.ID)
	if err := p.redis.Set(ctx, orderKey, orderJSON, p.cfg.OrderTimeout*2).Err(); err != nil {
		p.logger.Error("Failed to save order to Redis", zap.Error(err))
		return
	}

	// 设置订单状态为 pending（等待接单）
	statusKey := fmt.Sprintf("order:status:%s", order.ID)
	if err := p.redis.Set(ctx, statusKey, "pending", p.cfg.OrderTimeout*2).Err(); err != nil {
		p.logger.Error("Failed to save order status", zap.Error(err))
		return
	}

	// 记录订单创建时间（用于超时监控）
	createTimeKey := fmt.Sprintf("order:created:%s", order.ID)
	if err := p.redis.Set(ctx, createTimeKey, order.CreatedAt.Unix(), p.cfg.OrderTimeout*2).Err(); err != nil {
		p.logger.Error("Failed to save order create time", zap.Error(err))
		return
	}

	duration := time.Since(startTime)
	p.logger.Info("Order printed successfully",
		zap.Int("worker_id", workerID),
		zap.String("order_id", order.ID),
		zap.Duration("duration", duration),
	)
}
