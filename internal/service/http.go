package service

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/untibullet/subscription-service-em/internal/models"
	"github.com/untibullet/subscription-service-em/internal/repository"
	"go.uber.org/zap"
)

type HTTPService struct {
	repo repository.SubscriptionRepository
	log  *zap.Logger
}

func NewHTTPService(repo repository.SubscriptionRepository, log *zap.Logger) *HTTPService {
	return &HTTPService{repo: repo, log: log}
}

func (s *HTTPService) RegisterRoutes(e *echo.Echo) {
	g := e.Group("/api/v1/subscriptions")
	g.POST("", s.Create)
	g.GET("/:id", s.GetByID)
	g.PUT("/:id", s.Update)
	g.DELETE("/:id", s.Delete)
	g.GET("", s.List)
	g.GET("/cost", s.CalculateCost)
}

// DTOs

// swagger:model CreateRequest
type createReq struct {
	ServiceName string    `json:"service_name" validate:"required,min=1,max=255"`
	Price       int       `json:"price" validate:"required,gt=0"`
	UserID      uuid.UUID `json:"user_id" validate:"required"`
	StartDate   string    `json:"start_date" validate:"required"` // MM-YYYY
	EndDate     *string   `json:"end_date,omitempty"`             // MM-YYYY
}

// swagger:model UpdateRequest
type updateReq struct {
	ServiceName *string `json:"service_name,omitempty" validate:"omitempty,min=1,max=255"`
	Price       *int    `json:"price,omitempty" validate:"omitempty,gt=0"`
	StartDate   *string `json:"start_date,omitempty"` // MM-YYYY
	EndDate     *string `json:"end_date,omitempty"`   // MM-YYYY
}

// swagger:model listResp
type listResp struct {
	Data  []*models.Subscription `json:"data"`
	Total int                   `json:"total"`
}

// swagger:model costResp
type costResp struct {
	Total int `json:"total"`
}

// Helpers

func parseMonth(s string) (time.Time, error) {
	// ожидаем формат "MM-YYYY" -> приводим к 01-MM-YYYY
	t, err := time.Parse("01-2006", s)
	if err != nil {
		return time.Time{}, err
	}
	// нормализуем к первому числу
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

// Handlers

// @Summary Создать новую подписку
// @Description Создаёт новую подписку на сервис
// @ID create-subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param input body createReq true "Данные подписки"
// @Success 201 {object} models.Subscription "Созданная подписка"
// @Failure 400 {object} echo.Map "Неверный запрос"
// @Failure 500 {object} echo.Map "Внутренняя ошибка сервера"
// @Router /api/v1/subscriptions [post]
func (s *HTTPService) Create(c echo.Context) error {
	var req createReq
	if err := c.Bind(&req); err != nil {
		s.log.Warn("bind error", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	start, err := parseMonth(req.StartDate)
	if err != nil {
		s.log.Warn("invalid start_date", zap.String("value", req.StartDate), zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid start_date"})
	}

	var endPtr *time.Time
	if req.EndDate != nil {
		end, err := parseMonth(*req.EndDate)
		if err != nil {
			s.log.Warn("invalid end_date", zap.String("value", *req.EndDate), zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid end_date"})
		}
		endPtr = &end
	}

	now := time.Now().UTC()
	sub := models.Subscription{
		ID:          uuid.New(),
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      req.UserID,
		StartDate:   start,
		EndDate:     endPtr,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(c.Request().Context(), &sub); err != nil {
		s.log.Error("create failed", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create"})
	}

	return c.JSON(http.StatusCreated, sub)
}

// @Summary Получить подписку по ID
// @Description Возвращает информацию о подписке по её уникальному идентификатору
// @ID get-subscription-by-id
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path string true "UUID идентификатор подписки"
// @Success 200 {object} models.Subscription "Информация о подписке"
// @Failure 400 {object} echo.Map "Неверный формат ID"
// @Failure 404 {object} echo.Map "Подписка не найдена"
// @Failure 500 {object} echo.Map "Внутренняя ошибка сервера"
// @Router /api/v1/subscriptions/{id} [get]
func (s *HTTPService) GetByID(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		s.log.Warn("invalid id", zap.String("id", idStr), zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}

	sub, err := s.repo.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
		}
		s.log.Error("get failed", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to get"})
	}

	return c.JSON(http.StatusOK, sub)
}

// @Summary Обновить подписку
// @Description Обновляет существующую подписку по её ID
// @ID update-subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path string true "UUID идентификатор подписки"
// @Param input body updateReq true "Данные для обновления"
// @Success 200 {object} models.Subscription "Обновлённая подписка"
// @Failure 400 {object} echo.Map "Неверный запрос"
// @Failure 404 {object} echo.Map "Подписка не найдена"
// @Failure 500 {object} echo.Map "Внутренняя ошибка сервера"
// @Router /api/v1/subscriptions/{id} [put]
func (s *HTTPService) Update(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		s.log.Warn("invalid id", zap.String("id", idStr), zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}

	var req updateReq
	if err := c.Bind(&req); err != nil {
		s.log.Warn("bind error", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	// читаем текущую запись
	sub, err := s.repo.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
		}
		s.log.Error("get for update failed", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to get"})
	}

	// применяем изменения
	if req.ServiceName != nil {
		sub.ServiceName = *req.ServiceName
	}
	if req.Price != nil {
		sub.Price = *req.Price
	}
	if req.StartDate != nil {
		start, err := parseMonth(*req.StartDate)
		if err != nil {
			s.log.Warn("invalid start_date", zap.String("value", *req.StartDate), zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid start_date"})
		}
		sub.StartDate = start
	}
	if req.EndDate != nil {
		if *req.EndDate == "" {
			sub.EndDate = nil
		} else {
			end, err := parseMonth(*req.EndDate)
			if err != nil {
				s.log.Warn("invalid end_date", zap.String("value", *req.EndDate), zap.Error(err))
				return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid end_date"})
			}
			sub.EndDate = &end
		}
	}
	sub.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(c.Request().Context(), sub); err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
		}
		s.log.Error("update failed", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update"})
	}

	return c.JSON(http.StatusOK, sub)
}

// @Summary Удалить подписку
// @Description Удаляет подписку по её ID
// @ID delete-subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path string true "UUID идентификатор подписки"
// @Success 204 "Подписка удалена"
// @Failure 400 {object} echo.Map "Неверный формат ID"
// @Failure 404 {object} echo.Map "Подписка не найдена"
// @Failure 500 {object} echo.Map "Внутренняя ошибка сервера"
// @Router /api/v1/subscriptions/{id} [delete]
func (s *HTTPService) Delete(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		s.log.Warn("invalid id", zap.String("id", idStr), zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}

	if err := s.repo.Delete(c.Request().Context(), id); err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
		}
		s.log.Error("delete failed", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to delete"})
	}

	return c.NoContent(http.StatusNoContent)
}

// @Summary Список подписок
// @Description Возвращает список подписок с фильтрацией и пагинацией
// @ID list-subscriptions
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param user_id query string false "UUID пользователя"
// @Param service_name query string false "Название сервиса"
// @Param limit query int false "Количество элементов (макс. 500)" default(50)
// @Param offset query int false "Смещение" default(0)
// @Success 200 {object} listResp "Список подписок"
// @Failure 400 {object} echo.Map "Неверный запрос"
// @Failure 500 {object} echo.Map "Внутренняя ошибка сервера"
// @Router /api/v1/subscriptions [get]
func (s *HTTPService) List(c echo.Context) error {
	var (
		userIDPtr   *uuid.UUID
		serviceName *string
	)

	if v := c.QueryParam("user_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			s.log.Warn("invalid user_id", zap.String("value", v), zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user_id"})
		}
		userIDPtr = &id
	}
	if v := c.QueryParam("service_name"); v != "" {
		serviceName = &v
	}

	limit := 50
	offset := 0
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	filter := models.SubscriptionFilter{
		UserID:      userIDPtr,
		ServiceName: serviceName,
		Limit:       limit,
		Offset:      offset,
	}

	items, err := s.repo.List(c.Request().Context(), filter)
	if err != nil {
		s.log.Error("list failed", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to list"})
	}

	resp := listResp{
		Data:  items,
		Total: len(items),
	}
	return c.JSON(http.StatusOK, resp)
}

// @Summary Рассчитать стоимость подписок за период
// @Description Возвращает суммарную стоимость подписок пользователя или сервиса за указанный период
// @ID calculate-cost
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param start_period query string true "Начало периода (формат MM-YYYY)"
// @Param end_period query string true "Конец периода (формат MM-YYYY)"
// @Param user_id query string false "UUID пользователя"
// @Param service_name query string false "Название сервиса"
// @Success 200 {object} costResp "Суммарная стоимость"
// @Failure 400 {object} echo.Map "Неверный запрос"
// @Failure 500 {object} echo.Map "Внутренняя ошибка сервера"
// @Router /api/v1/subscriptions/cost [get]
func (s *HTTPService) CalculateCost(c echo.Context) error {
	var (
		userIDPtr   *uuid.UUID
		serviceName *string
	)

	startStr := c.QueryParam("start_period")
	endStr := c.QueryParam("end_period")
	if startStr == "" || endStr == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "start_period and end_period are required"})
	}

	start, err := parseMonth(startStr)
	if err != nil {
		s.log.Warn("invalid start_period", zap.String("value", startStr), zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid start_period"})
	}
	end, err := parseMonth(endStr)
	if err != nil {
		s.log.Warn("invalid end_period", zap.String("value", endStr), zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid end_period"})
	}

	if v := c.QueryParam("user_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			s.log.Warn("invalid user_id", zap.String("value", v), zap.Error(err))
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user_id"})
		}
		userIDPtr = &id
	}
	if v := c.QueryParam("service_name"); v != "" {
		serviceName = &v
	}

	filter := models.CostFilter{
		UserID:      userIDPtr,
		ServiceName: serviceName,
		StartPeriod: start,
		EndPeriod:   end,
	}

	total, err := s.repo.CalculateCost(c.Request().Context(), filter)
	if err != nil {
		s.log.Error("calculate cost failed", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to calculate"})
	}

	return c.JSON(http.StatusOK, costResp{Total: total})
}