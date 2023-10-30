package ports

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shotokan/firebase-training/internal/users/models"
)

type UserRepository interface {
	AddUser(ctx context.Context, user models.User) error
}

//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen --config=server.cfg.yaml ../../../api/users.yml
type HttpServer struct {
	repo UserRepository
}

func NewHttpServer(repo UserRepository) *HttpServer {
	return &HttpServer{
		repo: repo,
	}
}

func (h HttpServer) GetUsers(ctx echo.Context) error {
	result := User{Name: "Ivan"}
	return ctx.JSON(http.StatusOK, result)
}

func (h HttpServer) GetUserById(ctx echo.Context, userId int) error {
	result := []User{{Name: "Ivan"}}
	return ctx.JSON(http.StatusOK, result)
}

func (h HttpServer) CreateUser(ctx echo.Context) error {
	user := User{}
	err := ctx.Bind(&user)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, Error{
			Message: "something bad",
		})
	}
	userModel := models.User{
		Name:     user.Name,
		Email:    user.Email,
		Password: user.Password,
	}
	err = h.repo.AddUser(ctx.Request().Context(), userModel)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, Error{
			Message: "something bad",
		})
	}
	return ctx.JSON(http.StatusCreated, nil)
}
