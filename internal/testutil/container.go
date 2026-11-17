package testutil

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// DefaultImage is the Stirling PDF image used by the end-to-end tests. Override
// it with STIRLING_PDF_IMAGE to test another release.
const DefaultImage = "stirlingtools/stirling-pdf:3.3.0"

// StartStirling starts a Stirling PDF container and returns its HTTP endpoint.
// The caller owns the container and must terminate it.
func StartStirling(ctx context.Context) (testcontainers.Container, string, error) {
	image := os.Getenv("STIRLING_PDF_IMAGE")
	if image == "" {
		image = DefaultImage
	}
	request := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"SECURITY_ENABLELOGIN":        "false",
			"DISABLE_ADDITIONAL_FEATURES": "true",
			"SYSTEM_MAXFILESIZE":          "200",
			"SYSTEM_ENABLEANALYTICS":      "false",
		},
		WaitingFor: wait.ForHTTP("/api/v1/info/status").
			WithPort("8080/tcp").
			WithStartupTimeout(5 * time.Minute).
			WithPollInterval(2 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("start Stirling PDF container: %w", err)
	}
	endpoint, err := container.Endpoint(ctx, "http")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, "", fmt.Errorf("resolve Stirling PDF endpoint: %w", err)
	}
	return container, endpoint, nil
}
