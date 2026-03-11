# Domain Template — Complete Boilerplate

Replace `Order` / `order` with your actual domain name throughout.

---

## app/entities/order.go

```go
package entities

import "gorm.io/gorm"

type Order struct {
	gorm.Model
	ID     int64  `gorm:"primaryKey;autoIncrement"`
	UserID int64  `gorm:"not null;index"`
	Status string `gorm:"not null;default:'pending'"`
	Total  float64
}
```

---

## app/repository/order/repository.go

```go
package repository

import (
	"go-base-project/app/entities"

	"gorm.io/gorm"
)

type OrderRepository interface {
	Create(order entities.Order) error
	GetByID(id int64) (entities.Order, error)
	Update(order entities.Order) error
	GetAll() ([]entities.Order, error)
	GetByUserID(userID int64) ([]entities.Order, error)
}

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(order entities.Order) error {
	return r.db.Create(&order).Error
}

func (r *orderRepository) GetByID(id int64) (entities.Order, error) {
	var order entities.Order
	err := r.db.First(&order, id).Error
	return order, err
}

func (r *orderRepository) Update(order entities.Order) error {
	return r.db.Save(&order).Error
}

func (r *orderRepository) GetAll() ([]entities.Order, error) {
	var orders []entities.Order
	err := r.db.Find(&orders).Error
	return orders, err
}

func (r *orderRepository) GetByUserID(userID int64) ([]entities.Order, error) {
	var orders []entities.Order
	err := r.db.Where("user_id = ?", userID).Find(&orders).Error
	return orders, err
}
```

---

## app/services/order/service.go

```go
package service

import (
	"go-base-project/app/entities"
)

type OrderService interface {
	ProcessOrder(order entities.Order) (entities.Order, error)
}

type orderService struct{}

func NewOrderService() OrderService {
	return &orderService{}
}

func (s *orderService) ProcessOrder(order entities.Order) (entities.Order, error) {
	// Domain logic or external calls (e.g. payment gateway, inventory check)
	order.Status = "processed"
	return order, nil
}
```

---

## app/usecase/order/usecase.go

```go
package usecase

import (
	"go-base-project/app/entities"
)

// UseCase-owned interfaces — never import repository or service packages here
type OrderRepository interface {
	Create(order entities.Order) error
	GetByID(id int64) (entities.Order, error)
	Update(order entities.Order) error
	GetAll() ([]entities.Order, error)
	GetByUserID(userID int64) ([]entities.Order, error)
}

type OrderService interface {
	ProcessOrder(order entities.Order) (entities.Order, error)
}

type OrderUseCase struct {
	Repo    OrderRepository
	Service OrderService
}

func NewOrderUseCase(r OrderRepository, s OrderService) *OrderUseCase {
	return &OrderUseCase{
		Repo:    r,
		Service: s,
	}
}

func (u *OrderUseCase) CreateOrder(userID int64, total float64) error {
	o := entities.Order{UserID: userID, Total: total, Status: "pending"}
	return u.Repo.Create(o)
}

func (u *OrderUseCase) GetOrder(id int64) (entities.Order, error) {
	return u.Repo.GetByID(id)
}

func (u *OrderUseCase) GetAllOrders() ([]entities.Order, error) {
	return u.Repo.GetAll()
}

func (u *OrderUseCase) ProcessOrder(id int64) error {
	o, err := u.Repo.GetByID(id)
	if err != nil {
		return err
	}

	o, err = u.Service.ProcessOrder(o)
	if err != nil {
		return err
	}

	return u.Repo.Update(o)
}
```

---

## app/handler/web/order/dto.go

```go
package handler

type CreateOrderRequest struct {
	UserID int64   `json:"user_id"`
	Total  float64 `json:"total"`
}

type OrderResponse struct {
	ID     int64   `json:"id"`
	UserID int64   `json:"user_id"`
	Status string  `json:"status"`
	Total  float64 `json:"total"`
}
```

---

## app/handler/web/order/handler.go

```go
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-base-project/app/entities"
	"go-base-project/internal/customerror"
	"go-base-project/internal/handler"

	"github.com/gorilla/mux"
)

// Handler-owned UseCase interface — never import the usecase package
type UseCase interface {
	CreateOrder(userID int64, total float64) error
	GetOrder(id int64) (entities.Order, error)
	GetAllOrders() ([]entities.Order, error)
}

type OrderHandler struct {
	usecase UseCase
}

func NewOrderHandler(uc UseCase) *OrderHandler {
	return &OrderHandler{usecase: uc}
}

func (h *OrderHandler) Handlers() []handler.Configuration {
	return []handler.Configuration{
		{Pattern: "/orders", Action: h.CreateOrder, Method: "POST"},
		{Pattern: "/orders", Action: h.GetAllOrders, Method: "GET"},
		{Pattern: "/orders/{id}", Action: h.GetOrder, Method: "GET"},
	}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, customerror.CustomError{Code: 400, Message: "invalid request body"}.Error(), http.StatusBadRequest)
		return
	}

	if err := h.usecase.CreateOrder(req.UserID, req.Total); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, customerror.CustomError{Code: 400, Message: "invalid id"}.Error(), http.StatusBadRequest)
		return
	}

	order, err := h.usecase.GetOrder(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(OrderResponse{
		ID:     order.ID,
		UserID: order.UserID,
		Status: order.Status,
		Total:  order.Total,
	})
}

func (h *OrderHandler) GetAllOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.usecase.GetAllOrders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := make([]OrderResponse, len(orders))
	for i, o := range orders {
		resp[i] = OrderResponse{ID: o.ID, UserID: o.UserID, Status: o.Status, Total: o.Total}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
```

---

## app/handler/tasks/order/handler.go

```go
package handler

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
)

const TaskProcessOrder = "order:process"

// Task-owned UseCase interface
type OrderUseCase interface {
	ProcessOrder(id int64) error
}

type ProcessOrderPayload struct {
	OrderID int64 `json:"order_id"`
}

type OrderProcessor struct {
	usecase OrderUseCase
}

func NewOrderProcessor(uc OrderUseCase) *OrderProcessor {
	return &OrderProcessor{usecase: uc}
}

func (p *OrderProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload ProcessOrderPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}
	return p.usecase.ProcessOrder(payload.OrderID)
}
```

---

## app/workers/order/worker.go

```go
package worker

import (
	"encoding/json"

	taskHandler "go-base-project/app/handler/tasks/order"

	"github.com/hibiken/asynq"
)

type OrderWorker struct {
	client *asynq.Client
}

func NewOrderWorker(client *asynq.Client) *OrderWorker {
	return &OrderWorker{client: client}
}

func (w *OrderWorker) EnqueueProcessOrder(orderID int64) error {
	payload, err := json.Marshal(taskHandler.ProcessOrderPayload{OrderID: orderID})
	if err != nil {
		return err
	}

	task := asynq.NewTask(taskHandler.TaskProcessOrder, payload)
	_, err = w.client.Enqueue(task,
		asynq.MaxRetry(3),
		asynq.Queue("default"),
	)
	return err
}
```

---

## cmd/api/modules/order.go

```go
package modules

import (
	taskHandler "go-base-project/app/handler/tasks/order"
	webHandler "go-base-project/app/handler/web/order"
	repository "go-base-project/app/repository/order"
	service "go-base-project/app/services/order"
	usecase "go-base-project/app/usecase/order"
	worker "go-base-project/app/workers/order"

	"go.uber.org/fx"
)

var OrderModule = fx.Module("order",
	fx.Provide(
		repository.NewOrderRepository,
		service.NewOrderService,
		usecase.NewOrderUseCase,
		worker.NewOrderWorker,
		taskHandler.NewOrderProcessor,
		webHandler.NewOrderHandler,
		func(s repository.OrderRepository) usecase.OrderRepository { return s },
		func(s service.OrderService) usecase.OrderService { return s },
		func(s *usecase.OrderUseCase) taskHandler.OrderUseCase { return s },
		func(s *usecase.OrderUseCase) webHandler.UseCase { return s },
	),
)
```

---

## app/usecase/order/usecase_test.go

```go
package usecase_test

import (
	"testing"

	"go-base-project/app/entities"
	. "go-base-project/app/usecase/order"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks (inline, no separate mocks/ dir) ---

type mockOrderRepo struct{ mock.Mock }

func (m *mockOrderRepo) Create(o entities.Order) error {
	return m.Called(o).Error(0)
}
func (m *mockOrderRepo) GetByID(id int64) (entities.Order, error) {
	args := m.Called(id)
	return args.Get(0).(entities.Order), args.Error(1)
}
func (m *mockOrderRepo) Update(o entities.Order) error {
	return m.Called(o).Error(0)
}
func (m *mockOrderRepo) GetAll() ([]entities.Order, error) {
	args := m.Called()
	return args.Get(0).([]entities.Order), args.Error(1)
}
func (m *mockOrderRepo) GetByUserID(userID int64) ([]entities.Order, error) {
	args := m.Called(userID)
	return args.Get(0).([]entities.Order), args.Error(1)
}

type mockOrderService struct{ mock.Mock }

func (m *mockOrderService) ProcessOrder(o entities.Order) (entities.Order, error) {
	args := m.Called(o)
	return args.Get(0).(entities.Order), args.Error(1)
}

// --- Tests ---

func TestOrderUseCase_CreateOrder(t *testing.T) {
	repo := &mockOrderRepo{}
	svc := &mockOrderService{}
	uc := NewOrderUseCase(repo, svc)

	repo.On("Create", mock.AnythingOfType("entities.Order")).Return(nil)

	err := uc.CreateOrder(1, 99.99)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestOrderUseCase_ProcessOrder(t *testing.T) {
	repo := &mockOrderRepo{}
	svc := &mockOrderService{}
	uc := NewOrderUseCase(repo, svc)

	order := entities.Order{ID: 1, UserID: 1, Status: "pending", Total: 50.0}
	processed := entities.Order{ID: 1, UserID: 1, Status: "processed", Total: 50.0}

	repo.On("GetByID", int64(1)).Return(order, nil)
	svc.On("ProcessOrder", order).Return(processed, nil)
	repo.On("Update", processed).Return(nil)

	err := uc.ProcessOrder(1)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
	svc.AssertExpectations(t)
}
```

---

## app/repository/order/repository_test.go

```go
package repository_test

import (
	"testing"

	"go-base-project/app/entities"
	. "go-base-project/app/repository/order"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	db.AutoMigrate(&entities.Order{})
	return db
}

func TestOrderRepository_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOrderRepository(db)

	order := entities.Order{UserID: 1, Total: 100.0, Status: "pending"}
	err := repo.Create(order)
	assert.NoError(t, err)

	fetched, err := repo.GetByID(1)
	assert.NoError(t, err)
	assert.Equal(t, float64(100.0), fetched.Total)
}
```
