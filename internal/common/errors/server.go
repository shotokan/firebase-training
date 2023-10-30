package errors

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func Unauthorised(slug string, err error) *echo.HTTPError {
	return httpRespondWithError(err, slug, http.StatusUnauthorized)
}

func httpRespondWithError(err error, slug string, status int) *echo.HTTPError {
	return echo.NewHTTPError(http.StatusUnauthorized, "Please provide valid credentials")

}

type ErrorResponse struct {
	Slug       string `json:"slug"`
	httpStatus int
}
