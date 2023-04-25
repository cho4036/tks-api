package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/openinfradev/tks-api/internal/middleware/auth/request"

	"github.com/google/uuid"
	"github.com/openinfradev/tks-api/internal/repository"
	"github.com/openinfradev/tks-api/pkg/domain"
	"github.com/openinfradev/tks-api/pkg/httpErrors"
	"github.com/openinfradev/tks-api/pkg/log"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type IAlertUsecase interface {
	Get(alertId uuid.UUID) (domain.Alert, error)
	GetByName(organizationId string, name string) (domain.Alert, error)
	Fetch(organizationId string) ([]domain.Alert, error)
	Create(ctx context.Context, dto domain.Alert) (err error)
	Update(ctx context.Context, dto domain.Alert) error
	Delete(ctx context.Context, dto domain.Alert) error

	CreateAlertAction(ctx context.Context, dto domain.AlertAction) (alertActionId uuid.UUID, err error)
}

type AlertUsecase struct {
	repo        repository.IAlertRepository
	clusterRepo repository.IClusterRepository
}

func NewAlertUsecase(r repository.Repository) IAlertUsecase {
	return &AlertUsecase{
		repo:        r.Alert,
		clusterRepo: r.Cluster,
	}
}

func (u *AlertUsecase) Create(ctx context.Context, dto domain.Alert) (err error) {
	alertId, err := u.repo.Create(dto)
	if err != nil {
		return httpErrors.NewInternalServerError(err)
	}
	log.Info("newly created alertId:", alertId)

	return nil
}

func (u *AlertUsecase) Update(ctx context.Context, dto domain.Alert) error {
	return nil
}

func (u *AlertUsecase) Get(alertId uuid.UUID) (res domain.Alert, err error) {
	res, err = u.repo.Get(alertId)
	if err != nil {
		return domain.Alert{}, err
	}

	return
}

func (u *AlertUsecase) GetByName(organizationId string, name string) (res domain.Alert, err error) {
	res, err = u.repo.GetByName(organizationId, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Alert{}, httpErrors.NewNotFoundError(err)
		}
		return domain.Alert{}, err
	}
	return
}

func (u *AlertUsecase) Fetch(organizationId string) (res []domain.Alert, err error) {
	res, err = u.repo.Fetch(organizationId)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (u *AlertUsecase) Delete(ctx context.Context, dto domain.Alert) (err error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return httpErrors.NewBadRequestError(fmt.Errorf("Invalid token"))
	}

	_, err = u.Get(dto.ID)
	if err != nil {
		return httpErrors.NewNotFoundError(err)
	}

	*dto.UpdatorId = user.GetUserId()

	err = u.repo.Delete(dto)
	if err != nil {
		return err
	}

	return nil
}

func (u *AlertUsecase) CreateAlertAction(ctx context.Context, dto domain.AlertAction) (alertActionId uuid.UUID, err error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return uuid.Nil, httpErrors.NewBadRequestError(fmt.Errorf("Invalid token"))
	}

	_, err = u.repo.Get(dto.AlertId)
	if err != nil {
		return uuid.Nil, httpErrors.NewBadRequestError(fmt.Errorf("Not found alert"))
	}

	userId := user.GetUserId()
	dto.TakerId = &userId

	if dto.Status == domain.AlertActionStatus_INPROGRESS {
		dto.StartedAt = time.Now()
	} else if dto.Status == domain.AlertActionStatus_CLOSED {
		dto.CompletedAt = time.Now()
	} else {
		return uuid.Nil, httpErrors.NewBadRequestError(fmt.Errorf("Invalid Status"))
	}

	alertActionId, err = u.repo.CreateAlertAction(dto)
	if err != nil {
		return uuid.Nil, err
	}
	log.Info("newly created alertActionId:", alertActionId)

	return
}