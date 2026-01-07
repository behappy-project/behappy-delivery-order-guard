package ingestion

import (
	"encoding/json"
	"time"
)

// Order 统一的订单结构体
type Order struct {
	ID          string     `json:"id"`
	Platform    string     `json:"platform"`              // 平台名称：meituan, eleme
	OrderNo     string     `json:"order_no"`              // 平台订单号
	Customer    Customer   `json:"customer"`              // 客户信息
	Items       []Item     `json:"items"`                 // 订单项
	TotalAmount float64    `json:"total_amount"`          // 总金额
	Status      string     `json:"status"`                // 订单状态：pending, accepted, preparing, ready
	CreatedAt   time.Time  `json:"created_at"`            // 订单创建时间
	AcceptedAt  *time.Time `json:"accepted_at,omitempty"` // 接单时间
	ReadyAt     *time.Time `json:"ready_at,omitempty"`    // 出餐时间
}

// Customer 客户信息
type Customer struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}

// Item 订单项
type Item struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
	Note     string  `json:"note,omitempty"`
}

// MeituanOrder 美团订单格式
type MeituanOrder struct {
	OrderID      string        `json:"order_id"`
	OrderNumber  string        `json:"order_number"`
	BuyerName    string        `json:"buyer_name"`
	BuyerPhone   string        `json:"buyer_phone"`
	BuyerAddress string        `json:"buyer_address"`
	GoodsList    []MeituanItem `json:"goods_list"`
	TotalPrice   float64       `json:"total_price"`
	CreateTime   string        `json:"create_time"`
}

// MeituanItem 美团订单项
type MeituanItem struct {
	GoodsName string  `json:"goods_name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
	Remark    string  `json:"remark"`
}

// ElemeOrder 饿了么订单格式
type ElemeOrder struct {
	OrderID    string      `json:"orderId"`
	OrderSeq   string      `json:"orderSeq"`
	UserName   string      `json:"userName"`
	UserPhone  string      `json:"userPhone"`
	Address    string      `json:"address"`
	ItemList   []ElemeItem `json:"itemList"`
	TotalFee   float64     `json:"totalFee"`
	CreateTime int64       `json:"createTime"`
}

// ElemeItem 饿了么订单项
type ElemeItem struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
	Memo     string  `json:"memo"`
}

// ParseMeituanOrder 将美团订单转换为统一格式
func ParseMeituanOrder(data []byte) (*Order, error) {
	var meituanOrder MeituanOrder
	if err := json.Unmarshal(data, &meituanOrder); err != nil {
		return nil, err
	}

	// 解析时间
	createTime, err := time.Parse("2006-01-02 15:04:05", meituanOrder.CreateTime)
	if err != nil {
		createTime = time.Now()
	}

	// 转换订单项
	items := make([]Item, len(meituanOrder.GoodsList))
	for i, goods := range meituanOrder.GoodsList {
		items[i] = Item{
			Name:     goods.GoodsName,
			Quantity: goods.Quantity,
			Price:    goods.Price,
			Note:     goods.Remark,
		}
	}

	return &Order{
		ID:       meituanOrder.OrderID,
		Platform: "meituan",
		OrderNo:  meituanOrder.OrderNumber,
		Customer: Customer{
			Name:    meituanOrder.BuyerName,
			Phone:   meituanOrder.BuyerPhone,
			Address: meituanOrder.BuyerAddress,
		},
		Items:       items,
		TotalAmount: meituanOrder.TotalPrice,
		Status:      "pending",
		CreatedAt:   createTime,
	}, nil
}

// ParseElemeOrder 将饿了么订单转换为统一格式
func ParseElemeOrder(data []byte) (*Order, error) {
	var elemeOrder ElemeOrder
	if err := json.Unmarshal(data, &elemeOrder); err != nil {
		return nil, err
	}

	// 解析时间（饿了么使用时间戳）
	createTime := time.Unix(elemeOrder.CreateTime, 0)

	// 转换订单项
	items := make([]Item, len(elemeOrder.ItemList))
	for i, item := range elemeOrder.ItemList {
		items[i] = Item{
			Name:     item.Name,
			Quantity: item.Quantity,
			Price:    item.Price,
			Note:     item.Memo,
		}
	}

	return &Order{
		ID:       elemeOrder.OrderID,
		Platform: "eleme",
		OrderNo:  elemeOrder.OrderSeq,
		Customer: Customer{
			Name:    elemeOrder.UserName,
			Phone:   elemeOrder.UserPhone,
			Address: elemeOrder.Address,
		},
		Items:       items,
		TotalAmount: elemeOrder.TotalFee,
		Status:      "pending",
		CreatedAt:   createTime,
	}, nil
}
