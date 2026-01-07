package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"behappy-delivery-order-guard/internal/config"

	"go.uber.org/zap"
)

// Ingester 订单接收器
type Ingester struct {
	cfg       *config.Config
	logger    *zap.Logger
	orderChan chan<- *Order
}

// NewIngester 创建订单接收器
func NewIngester(cfg *config.Config, logger *zap.Logger, orderChan chan<- *Order) *Ingester {
	return &Ingester{
		cfg:       cfg,
		logger:    logger,
		orderChan: orderChan,
	}
}

// Start 启动订单接收器，模拟从不同平台接收订单
func (i *Ingester) Start(ctx context.Context) {
	i.logger.Info("Order ingester started")

	// 为每个启用的平台启动一个 goroutine
	for _, platform := range i.cfg.Platforms {
		if !platform.Enabled {
			continue
		}

		switch platform.Name {
		case "meituan":
			go i.simulateMeituanOrders(ctx, platform.Interval)
		case "eleme":
			go i.simulateElemeOrders(ctx, platform.Interval)
		default:
			i.logger.Warn("Unknown platform", zap.String("platform", platform.Name))
		}
	}

	// 等待上下文取消
	<-ctx.Done()
	i.logger.Info("Order ingester stopped")
}

// simulateMeituanOrders 模拟美团订单
func (i *Ingester) simulateMeituanOrders(ctx context.Context, interval int) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			order := i.generateMeituanOrder()
			orderJSON, _ := json.Marshal(order)
			parsedOrder, err := ParseMeituanOrder(orderJSON)
			if err != nil {
				i.logger.Error("Failed to parse Meituan order", zap.Error(err))
				continue
			}

			select {
			case i.orderChan <- parsedOrder:
				i.logger.Info("Received Meituan order",
					zap.String("order_id", parsedOrder.ID),
					zap.String("order_no", parsedOrder.OrderNo),
					zap.Float64("amount", parsedOrder.TotalAmount),
				)
			case <-ctx.Done():
				return
			}
		}
	}
}

// simulateElemeOrders 模拟饿了么订单
func (i *Ingester) simulateElemeOrders(ctx context.Context, interval int) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			order := i.generateElemeOrder()
			orderJSON, _ := json.Marshal(order)
			parsedOrder, err := ParseElemeOrder(orderJSON)
			if err != nil {
				i.logger.Error("Failed to parse Eleme order", zap.Error(err))
				continue
			}

			select {
			case i.orderChan <- parsedOrder:
				i.logger.Info("Received Eleme order",
					zap.String("order_id", parsedOrder.ID),
					zap.String("order_no", parsedOrder.OrderNo),
					zap.Float64("amount", parsedOrder.TotalAmount),
				)
			case <-ctx.Done():
				return
			}
		}
	}
}

// generateMeituanOrder 生成模拟的美团订单
func (i *Ingester) generateMeituanOrder() MeituanOrder {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	orderID := fmt.Sprintf("MT%d", time.Now().UnixNano())
	orderNumber := fmt.Sprintf("MT%08d", r.Intn(99999999))

	items := []MeituanItem{
		{GoodsName: "宫保鸡丁", Quantity: 1, Price: 28.00, Remark: "不要花生"},
		{GoodsName: "麻婆豆腐", Quantity: 1, Price: 18.00, Remark: ""},
		{GoodsName: "白米饭", Quantity: 2, Price: 3.00, Remark: ""},
	}

	totalPrice := 0.0
	for _, item := range items {
		totalPrice += item.Price * float64(item.Quantity)
	}

	return MeituanOrder{
		OrderID:      orderID,
		OrderNumber:  orderNumber,
		BuyerName:    "张先生",
		BuyerPhone:   "13800138000",
		BuyerAddress: "北京市朝阳区xxx街道xxx号",
		GoodsList:    items,
		TotalPrice:   totalPrice,
		CreateTime:   time.Now().Format("2006-01-02 15:04:05"),
	}
}

// generateElemeOrder 生成模拟的饿了么订单
func (i *Ingester) generateElemeOrder() ElemeOrder {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	orderID := fmt.Sprintf("EL%d", time.Now().UnixNano())
	orderSeq := fmt.Sprintf("EL%08d", r.Intn(99999999))

	items := []ElemeItem{
		{Name: "红烧肉", Quantity: 1, Price: 38.00, Memo: "少油"},
		{Name: "清炒时蔬", Quantity: 1, Price: 15.00, Memo: ""},
		{Name: "米饭", Quantity: 2, Price: 2.00, Memo: ""},
	}

	totalFee := 0.0
	for _, item := range items {
		totalFee += item.Price * float64(item.Quantity)
	}

	return ElemeOrder{
		OrderID:    orderID,
		OrderSeq:   orderSeq,
		UserName:   "李女士",
		UserPhone:  "13900139000",
		Address:    "上海市浦东新区xxx路xxx号",
		ItemList:   items,
		TotalFee:   totalFee,
		CreateTime: time.Now().Unix(),
	}
}
