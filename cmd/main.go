package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	middleware "github.com/oapi-codegen/echo-middleware"
	"github.com/shotokan/firebase-training/internal/common"
	"github.com/shotokan/firebase-training/internal/users/adapters"
	"github.com/shotokan/firebase-training/internal/users/ports"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

func main() {
	port := flag.String("port", "8080", "Port for test HTTP server")
	flag.Parse()

	// Create a fake authenticator. This allows us to issue tokens, and also
	// implements a validator to check their validity.
	fa, err := common.NewFakeAuthenticator()
	if err != nil {
		log.Fatalln("error creating authenticator:", err)
	}

	// This is how you set up a basic Echo router
	e := echo.New()
	// Log all requests
	e.Use(echomiddleware.Logger())

	// path := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	client, err := firestore.NewClient(context.Background(), os.Getenv("GCP_PROJECT"))
	if err != nil {
		e.Logger.Fatal(err)
	}

	var opts []option.ClientOption
	if file := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); file != "" {
		opts = append(opts, option.WithCredentialsFile(file))
	}

	config := &firebase.Config{ProjectID: os.Getenv("GCP_PROJECT")}
	firebaseApp, err := firebase.NewApp(context.Background(), config, opts...)
	if err != nil {
		logrus.Fatalf("error initializing app: %v\n", err)
	}

	authClient, err := firebaseApp.Auth(context.Background())
	if err != nil {
		logrus.WithError(err).Fatal("Unable to create firebase Auth client")
	}

	// Create middleware for validating tokens.
	mw, err := CreateMiddleware(fa, authClient)
	if err != nil {
		log.Fatalln("error creating middleware:", err)
	}

	// Use our validation middleware to check all requests against the
	// OpenAPI schema.
	e.Use(mw...)

	userRepo := adapters.NewUserFirestoreRepository(client)
	users := ports.NewHttpServer(userRepo)
	ports.RegisterHandlers(e, users)
	// We're going to print some useful things for interacting with this server.
	// This token allows access to any API's with no specific claims.
	readerJWS, err := fa.CreateJWSWithClaims([]string{})
	if err != nil {
		log.Fatalln("error creating reader JWS:", err)
	}
	// This token allows access to API's with no scopes, and with the "things:w" claim.
	writerJWS, err := fa.CreateJWSWithClaims([]string{"things:w"})
	if err != nil {
		log.Fatalln("error creating writer JWS:", err)
	}

	log.Println("Reader token", string(readerJWS))
	log.Println("Writer token", string(writerJWS))

	data, err := json.MarshalIndent(e.Routes(), "", "  ")
	if err != nil {
		e.Logger.Fatal(err)
	}
	os.WriteFile("routes.json", data, 0644)

	// And we serve HTTP until the world ends.
	e.Logger.Fatal(e.Start(net.JoinHostPort("0.0.0.0", *port)))
}

func CreateMiddleware(v common.JWSValidator, authClient *auth.Client) ([]echo.MiddlewareFunc, error) {
	spec, err := ports.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("loading spec: %w", err)
	}
	// Clear out the servers array in the swagger spec, that skips validating
	// that server names match. We don't know how this thing will be run.
	spec.Servers = nil
	validator := middleware.OapiRequestValidatorWithOptions(spec,
		&middleware.Options{
			Options: openapi3filter.Options{
				AuthenticationFunc: common.NewAuthenticator(v, authClient),
			},
		})

	return []echo.MiddlewareFunc{validator}, nil
}
