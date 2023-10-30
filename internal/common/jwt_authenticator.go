package common

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"firebase.google.com/go/auth"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/lestrrat-go/jwx/jwt"
	middleware "github.com/oapi-codegen/echo-middleware"
	commonerrors "github.com/shotokan/firebase-training/internal/common/errors"
)

// JWSValidator is used to validate JWS payloads and return a JWT if they're
// valid
type JWSValidator interface {
	ValidateJWS(jws string) (jwt.Token, error)
}

const JWTClaimsContextKey = "jwt_claims"
const UserContextKey = "user"

var (
	ErrNoAuthHeader      = errors.New("Authorization header is missing")
	ErrInvalidAuthHeader = errors.New("Authorization header is malformed")
	ErrClaimsInvalid     = errors.New("Provided claims do not match expected scopes")
)

// GetJWSFromRequest extracts a JWS string from an Authorization: Bearer <jws> header
func GetJWSFromRequest(req *http.Request) (string, error) {
	authHdr := req.Header.Get("Authorization")
	// Check for the Authorization header.
	if authHdr == "" {
		return "", ErrNoAuthHeader
	}
	// We expect a header value of the form "Bearer <token>", with 1 space after
	// Bearer, per spec.
	prefix := "Bearer "
	if !strings.HasPrefix(authHdr, prefix) {
		return "", ErrInvalidAuthHeader
	}
	return strings.TrimPrefix(authHdr, prefix), nil
}

func NewAuthenticator(v JWSValidator, authClient *auth.Client) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		return Authenticate(v, ctx, input, authClient)
	}
}

// Authenticate uses the specified validator to ensure a JWT is valid, then makes
// sure that the claims provided by the JWT match the scopes as required in the API.
func Authenticate(v JWSValidator, ctx context.Context, input *openapi3filter.AuthenticationInput, authClient *auth.Client) error {
	// Our security scheme is named BearerAuth, ensure this is the case
	if !strings.EqualFold(input.SecuritySchemeName, "BearerAuth") {
		return fmt.Errorf("security scheme %s != 'BearerAuth'", input.SecuritySchemeName)
	}

	// Now, we need to get the JWS from the request, to match the request expectations
	// against request contents.
	jws, err := GetJWSFromRequest(input.RequestValidationInput.Request)
	if err != nil {
		return fmt.Errorf("getting jws: %w", err)
	}

	// if the JWS is valid, we have a JWT, which will contain a bunch of claims.
	token, err := authClient.VerifyIDToken(ctx, jws)
	if err != nil {
		return commonerrors.Unauthorised("unable-to-verify-jwt", err)
	}

	// Set the property on the echo context so the handler is able to
	// access the claims data we generate in here.

	eCtx := middleware.GetEchoContext(ctx)
	eCtx.Set(JWTClaimsContextKey, token)
	eCtx.Set(UserContextKey, token)
	ctx = context.WithValue(ctx, UserContextKey, User{
		UUID:        token.UID,
		Email:       token.Claims["email"].(string),
		Role:        token.Claims["role"].(string),
		DisplayName: token.Claims["name"].(string),
	})

	return nil
}

// GetClaimsFromToken returns a list of claims from the token. We store these
// as a list under the "perms" claim, short for permissions, to keep the token
// shorter.
func GetClaimsFromToken(t jwt.Token) ([]string, error) {
	rawPerms, found := t.Get(PermissionsClaim)
	if !found {
		// If the perms aren't found, it means that the token has none, but it has
		// passed signature validation by now, so it's a valid token, so we return
		// the empty list.
		return make([]string, 0), nil
	}

	// rawPerms will be an untyped JSON list, so we need to convert it to
	// a string list.
	rawList, ok := rawPerms.([]interface{})
	if !ok {
		return nil, fmt.Errorf("'%s' claim is unexpected type'", PermissionsClaim)
	}

	claims := make([]string, len(rawList))

	for i, rawClaim := range rawList {
		var ok bool
		claims[i], ok = rawClaim.(string)
		if !ok {
			return nil, fmt.Errorf("%s[%d] is not a string", PermissionsClaim, i)
		}
	}
	return claims, nil
}

func CheckTokenClaims(expectedClaims []string, t jwt.Token) error {
	claims, err := GetClaimsFromToken(t)
	if err != nil {
		return fmt.Errorf("getting claims from token: %w", err)
	}
	// Put the claims into a map, for quick access.
	claimsMap := make(map[string]bool, len(claims))
	for _, c := range claims {
		claimsMap[c] = true
	}

	for _, e := range expectedClaims {
		if !claimsMap[e] {
			return ErrClaimsInvalid
		}
	}
	return nil
}

func tokenFromHeader(r *http.Request) string {
	headerValue := r.Header.Get("Authorization")

	if len(headerValue) > 7 && strings.ToLower(headerValue[0:6]) == "bearer" {
		return headerValue[7:]
	}

	return ""
}

type User struct {
	UUID  string
	Email string
	Role  string

	DisplayName string
}

var (
	// if we expect that the user of the function may be interested with concrete error,
	// it's a good idea to provide variable with this error
	NoUserInContextError = commonerrors.NewAuthorizationError("no user in context", "no-user-found")
)

func UserFromCtx(ctx context.Context) (User, error) {
	u, ok := ctx.Value(UserContextKey).(User)
	if ok {
		return u, nil
	}

	return User{}, NoUserInContextError
}
