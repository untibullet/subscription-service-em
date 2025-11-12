package models

import (
	"time"

	"github.com/google/uuid"
)

// Subscription представляет подписку пользователя
// swagger:model Subscription
type Subscription struct {
	ID          uuid.UUID  `json:"id"`
	ServiceName string     `json:"service_name"`
	Price       int        `json:"price"`
	UserID      uuid.UUID  `json:"user_id"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateSubscriptionDTO - входные данные для создания подписки
// swagger:model CreateSubscriptionDTO
type CreateSubscriptionDTO struct {
	ServiceName string    `json:"service_name" validate:"required,min=1,max=255"`
	Price       int       `json:"price" validate:"required,gt=0"`
	UserID      uuid.UUID `json:"user_id" validate:"required"`
	StartDate   string    `json:"start_date" validate:"required"` // формат: MM-YYYY
	EndDate     *string   `json:"end_date,omitempty"`             // формат: MM-YYYY
}

// UpdateSubscriptionDTO - входные данные для обновления подписки
// swagger:model UpdateSubscriptionDTO
type UpdateSubscriptionDTO struct {
	ServiceName *string    `json:"service_name,omitempty" validate:"omitempty,min=1,max=255"`
	Price       *int       `json:"price,omitempty" validate:"omitempty,gt=0"`
	StartDate   *string    `json:"start_date,omitempty"` // формат: MM-YYYY
	EndDate     *string    `json:"end_date,omitempty"`   // формат: MM-YYYY
}

// SubscriptionFilter - фильтры для выборки подписок
// swagger:model SubscriptionFilter
type SubscriptionFilter struct {
	UserID      *uuid.UUID
	ServiceName *string
	Limit       int
	Offset      int
}

// CostFilter - фильтры для подсчета стоимости
// swagger:model CostFilter
type CostFilter struct {
	UserID      *uuid.UUID
	ServiceName *string
	StartPeriod time.Time
	EndPeriod   time.Time
}