package service

import (
	"context"
	"errors"
	"github.com/lvow2022/udisk/internel/domain"
	"github.com/lvow2022/udisk/internel/repository"
)

var (
	ErrDuplicateEmail        = repository.ErrDuplicateUser
	ErrInvalidUserOrPassword = errors.New("用户不存在或者密码不对")
)

type UserService interface {
	Signup(ctx context.Context, u domain.User) error
	Signin(ctx context.Context, email string, password string) (domain.User, error)

	FindById(ctx context.Context,
		uid int64) (domain.User, error)
	FindOrCreate(ctx context.Context, phone string) (domain.User, error)
}

type userService struct {
	repo repository.UserRepository
}

func (svc *userService) Signup(ctx context.Context, u domain.User) error {
	//TODO implement me
	panic("implement me")
}

func (svc *userService) Signin(ctx context.Context, email string, password string) (domain.User, error) {
	//TODO implement me
	panic("implement me")
}

func (svc *userService) FindById(ctx context.Context, uid int64) (domain.User, error) {
	//TODO implement me
	panic("implement me")
}

func (svc *userService) FindOrCreate(ctx context.Context, phone string) (domain.User, error) {
	//TODO implement me
	panic("implement me")
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{
		repo: repo,
	}
}
