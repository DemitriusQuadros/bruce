# Advanced Patterns Reference

## Pagination

Always paginate list endpoints. Never return unbounded slices.

```go
// In usecase
type PaginationParams struct {
    Page     int
    PageSize int
}

type PaginatedResult[T any] struct {
    Items      []T
    TotalCount int64
    Page       int
    PageSize   int
    TotalPages int
}

func (u *OrderUseCase) GetOrdersPaginated(params PaginationParams) (PaginatedResult[entities.Order], error) {
    return u.Repo.GetPaginated(params.Page, params.PageSize)
}

// In repository
func (r *orderRepository) GetPaginated(page, pageSize int) (PaginatedResult[entities.Order], error) {
    var orders []entities.Order
    var total int64

    offset := (page - 1) * pageSize

    r.db.Model(&entities.Order{}).Count(&total)
    err := r.db.Offset(offset).Limit(pageSize).Find(&orders).Error

    totalPages := int(total) / pageSize
    if int(total)%pageSize != 0 {
        totalPages++
    }

    return PaginatedResult[entities.Order]{
        Items:      orders,
        TotalCount: total,
        Page:       page,
        PageSize:   pageSize,
        TotalPages: totalPages,
    }, err
}

// In HTTP handler — parse from query params
page, _ := strconv.Atoi(r.URL.Query().Get("page"))
if page < 1 { page = 1 }
pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
if pageSize < 1 || pageSize > 100 { pageSize = 20 }
```

---

## Database Transactions

Use GORM transactions for multi-step writes that must be atomic.

```go
// In repository — expose transaction support
func (r *orderRepository) CreateWithItems(order entities.Order, items []entities.OrderItem) error {
    return r.db.Transaction(func(tx *gorm.DB) error {
        if err := tx.Create(&order).Error; err != nil {
            return err  // Triggers automatic rollback
        }
        for i := range items {
            items[i].OrderID = order.ID
        }
        return tx.Create(&items).Error
    })
}
```

---

## Soft Delete

GORM's `gorm.Model` includes `DeletedAt` for soft delete out of the box.

```go
// Soft delete (sets DeletedAt, not removed from DB)
r.db.Delete(&order)

// Hard delete
r.db.Unscoped().Delete(&order)

// Query only non-deleted (default behavior with gorm.Model)
r.db.Find(&orders)  // Automatically adds WHERE deleted_at IS NULL

// Query including soft-deleted
r.db.Unscoped().Find(&orders)
```

If you don't want soft delete, use a plain struct without `gorm.Model`:
```go
type Order struct {
    ID        int64     `gorm:"primaryKey;autoIncrement"`
    CreatedAt time.Time
    UpdatedAt time.Time
    // No DeletedAt — hard deletes only
}
```

---

## Custom Error Handling

Use `customerror.CustomError` so HTTP handlers can respond with the right status code.

```go
import "go-base-project/internal/customerror"

// In usecase or service
func (u *OrderUseCase) GetOrder(id int64) (entities.Order, error) {
    order, err := u.Repo.GetByID(id)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return entities.Order{}, customerror.CustomError{
                Code:    404,
                Message: "order not found",
            }
        }
        return entities.Order{}, err
    }
    return order, nil
}

// In HTTP handler — check and unwrap
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
    order, err := h.usecase.GetOrder(id)
    if err != nil {
        var ce customerror.CustomError
        if errors.As(err, &ce) {
            http.Error(w, ce.Message, ce.Code)
            return
        }
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }
    // ...
}
```

---

## Prometheus Metrics

Instrument new endpoints using the existing `MetricsCollector`.

```go
// In HTTP handler (injected via FX)
type OrderHandler struct {
    usecase  UseCase
    metrics  *metrics.MetricsCollector
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
    h.metrics.IncCounter("order_create_requests_total")
    // ... handler logic
    h.metrics.IncCounter("order_create_success_total")
}
```

For timing:
```go
start := time.Now()
// ... operation
elapsed := time.Since(start).Seconds()
h.metrics.ObserveHistogram("order_process_duration_seconds", elapsed)
```

---

## Asynq Task with Options

```go
func (w *OrderWorker) EnqueueProcessOrder(orderID int64, priority string) error {
    payload, _ := json.Marshal(ProcessOrderPayload{OrderID: orderID})
    task := asynq.NewTask(TaskProcessOrder, payload)

    _, err := w.client.Enqueue(task,
        asynq.MaxRetry(5),
        asynq.Timeout(30*time.Second),
        asynq.Queue(priority),           // "critical", "default", "low"
        asynq.Deadline(time.Now().Add(1*time.Hour)),
    )
    return err
}
```

---

## Context Propagation

Always accept and pass `context.Context` through long-running operations.

```go
// In repository — pass context to GORM
func (r *orderRepository) GetByID(ctx context.Context, id int64) (entities.Order, error) {
    var order entities.Order
    err := r.db.WithContext(ctx).First(&order, id).Error
    return order, err
}

// In HTTP handler — pass request context down
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    order, err := h.usecase.GetOrder(ctx, id)
    // ...
}
```

---

## FX Lifecycle Hooks

Use `fx.Lifecycle` for startup/shutdown logic (e.g., running migrations, closing connections).

```go
func NewDatabase(lc fx.Lifecycle, cfg *configuration.Config) (*gorm.DB, error) {
    db, err := gorm.Open(...)
    if err != nil {
        return nil, err
    }

    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            return db.AutoMigrate(&entities.Order{})
        },
        OnStop: func(ctx context.Context) error {
            sqlDB, _ := db.DB()
            return sqlDB.Close()
        },
    })

    return db, nil
}
```

---

## Benchmark Tests

```go
func BenchmarkOrderUseCase_CreateOrder(b *testing.B) {
    repo := &mockOrderRepo{}
    svc := &mockOrderService{}
    uc := NewOrderUseCase(repo, svc)

    repo.On("Create", mock.Anything).Return(nil)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        uc.CreateOrder(1, 99.99)
    }
}
```

Run with: `go test -bench=BenchmarkOrderUseCase -benchmem ./app/usecase/order/...`

---

## GORM Query Optimization

```go
// SELECT only needed fields
r.db.Select("id", "status", "total").Find(&orders)

// Preload associations (avoid N+1)
r.db.Preload("Items").Find(&orders)

// Batch operations
r.db.CreateInBatches(items, 100)

// Raw SQL when GORM is awkward
r.db.Raw("SELECT id, SUM(total) as revenue FROM orders WHERE status = ? GROUP BY id", "completed").Scan(&result)

// Add composite index via migration
r.db.Exec("CREATE INDEX IF NOT EXISTS idx_orders_user_status ON orders(user_id, status)")
```
