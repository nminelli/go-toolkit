package webapp

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/MFN-AISystems/go-toolkit/telemetry/log"
)

func LoadEnvFile() error {
	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		if os.Getenv("GO_ENV") == "test" {
			return nil
		}

		environment = "local"
	}

	envFile := fmt.Sprintf(".env.%s", environment)
	if originalErr := godotenv.Load(envFile); originalErr != nil {
		log.Error(context.Background(), fmt.Sprintf("Warning: Could not load %s file: %v\n", envFile, originalErr))

		// Fallback to .env file
		if err := godotenv.Load(".env"); err != nil {
			return errors.Join(originalErr, err)
		}
	}

	return nil
}
