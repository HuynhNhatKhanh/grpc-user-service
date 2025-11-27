package postgres

import (
	"context"
	"grpc-user-service/internal/domain/user"
)

type UserRepoPG struct {
	// db connection pool, e.g. *sql.DB
}

func NewUserRepoPG() *UserRepoPG {
	return &UserRepoPG{}
}

func (r *UserRepoPG) Create(ctx context.Context, u *user.User) (int64, error) {
	// TODO: implement insert
	return 1, nil
}

func (r *UserRepoPG) GetByID(ctx context.Context, id int64) (*user.User, error) {
	// TODO: implement query
	return &user.User{ID: id, Name: "John", Email: "john@example.com"}, nil
}
