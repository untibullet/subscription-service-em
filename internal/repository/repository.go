package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/untibullet/subscription-service-em/internal/models"
)

// SubscriptionRepository определяет методы работы с подписками
type SubscriptionRepository interface {
	Create(ctx context.Context, sub *models.Subscription) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error)
	Update(ctx context.Context, sub *models.Subscription) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter models.SubscriptionFilter) ([]*models.Subscription, error)
	CalculateCost(ctx context.Context, filter models.CostFilter) (int, error)
}
