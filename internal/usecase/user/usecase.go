package user

import (
	"context"
	"grpc-user-service/internal/domain/user"
)

type UserRepository interface {
	Create(ctx context.Context, u *user.User) (int64, error)
	GetByID(ctx context.Context, id int64) (*user.User, error)
}

type UserUsecase struct {
	repo UserRepository
}

func NewUserUsecase(r UserRepository) *UserUsecase {
	return &UserUsecase{repo: r}
}

func (u *UserUsecase) CreateUser(ctx context.Context, uEntity *user.User) (int64, error) {
	return u.repo.Create(ctx, uEntity)
}

func (u *UserUsecase) GetUser(ctx context.Context, id int64) (*user.User, error) {
	return u.repo.GetByID(ctx, id)
}
