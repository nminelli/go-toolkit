package webapp_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MFN-AISystems/go-toolkit/webapp"
)

func TestLoadEnvFile(t *testing.T) {
	testCases := []struct {
		name               string
		envVars            map[string]string
		createFiles        map[string]string
		expectError        bool
		validateError      func(t *testing.T, err error)
		validateEnvLoaded  func(t *testing.T)
	}{
		{
			name: "loads .env.local when ENVIRONMENT is empty (default)",
			envVars: map[string]string{
				"ENVIRONMENT": "",
			},
			createFiles: map[string]string{
				".env.local": "TEST_VAR_LOCAL=loaded_from_local\n",
			},
			expectError: false,
			validateEnvLoaded: func(t *testing.T) {
				assert.Equal(t, "loaded_from_local", os.Getenv("TEST_VAR_LOCAL"))
			},
		},
		{
			name: "loads .env.{environment} when ENVIRONMENT is set",
			envVars: map[string]string{
				"ENVIRONMENT": "production",
			},
			createFiles: map[string]string{
				".env.production": "TEST_VAR_PROD=loaded_from_production\n",
			},
			expectError: false,
			validateEnvLoaded: func(t *testing.T) {
				assert.Equal(t, "loaded_from_production", os.Getenv("TEST_VAR_PROD"))
			},
		},
		{
			name: "fallback to .env when environment-specific file doesn't exist",
			envVars: map[string]string{
				"ENVIRONMENT": "staging",
			},
			createFiles: map[string]string{
				".env": "TEST_VAR_FALLBACK=loaded_from_fallback\n",
			},
			expectError: false,
			validateEnvLoaded: func(t *testing.T) {
				assert.Equal(t, "loaded_from_fallback", os.Getenv("TEST_VAR_FALLBACK"))
			},
		},
		{
			name: "skip loading when GO_ENV=test",
			envVars: map[string]string{
				"GO_ENV":      "test",
				"ENVIRONMENT": "",
			},
			createFiles: map[string]string{},
			expectError: false,
			validateEnvLoaded: func(t *testing.T) {
				// No environment variables should be loaded
			},
		},
		{
			name: "return error when both environment-specific and fallback files don't exist",
			envVars: map[string]string{
				"ENVIRONMENT": "nonexistent",
			},
			createFiles: map[string]string{},
			expectError: true,
			validateError: func(t *testing.T, err error) {
				require.Error(t, err)
				// The error should contain information about both failures
				// errors.Join creates an error that unwraps to multiple errors
				var joinedErrs interface{ Unwrap() []error }
				if errors.As(err, &joinedErrs) {
					unwrapped := joinedErrs.Unwrap()
					assert.Len(t, unwrapped, 2, "expected error to contain both original and fallback errors")
				}
			},
		},
		{
			name: "loads .env.development when ENVIRONMENT is development",
			envVars: map[string]string{
				"ENVIRONMENT": "development",
			},
			createFiles: map[string]string{
				".env.development": "TEST_VAR_DEV=loaded_from_development\n",
			},
			expectError: false,
			validateEnvLoaded: func(t *testing.T) {
				assert.Equal(t, "loaded_from_development", os.Getenv("TEST_VAR_DEV"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory for test files
			tmpDir := t.TempDir()
			oldDir, err := os.Getwd()
			require.NoError(t, err)
			
			// Change to temp directory
			err = os.Chdir(tmpDir)
			require.NoError(t, err)
			defer func() {
				err := os.Chdir(oldDir)
				require.NoError(t, err)
			}()

			// Unset GO_ENV first to allow actual env loading (unless test explicitly sets it)
			if _, exists := tc.envVars["GO_ENV"]; !exists {
				t.Setenv("GO_ENV", "")
			}

			// Set environment variables
			for key, value := range tc.envVars {
				t.Setenv(key, value)
			}

			// Create test files
			for filename, content := range tc.createFiles {
				err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
				require.NoError(t, err)
			}

			// Run the test
			err = webapp.LoadEnvFile()

			// Validate results
			if tc.expectError {
				require.Error(t, err)
				if tc.validateError != nil {
					tc.validateError(t, err)
				}
			} else {
				require.NoError(t, err)
				if tc.validateEnvLoaded != nil {
					tc.validateEnvLoaded(t)
				}
			}
		})
	}
}

func TestLoadEnvFile_Integration(t *testing.T) {
	t.Run("New() fails gracefully when env files cannot be loaded", func(t *testing.T) {
		// Create temporary directory with no env files
		tmpDir := t.TempDir()
		oldDir, err := os.Getwd()
		require.NoError(t, err)
		
		err = os.Chdir(tmpDir)
		require.NoError(t, err)
		defer func() {
			err := os.Chdir(oldDir)
			require.NoError(t, err)
		}()

		t.Setenv("ENVIRONMENT", "nonexistent")

		app, err := webapp.New()
		assert.Error(t, err)
		assert.Nil(t, app)
		assert.Contains(t, err.Error(), "failed to load environment file")
	})

	t.Run("New() succeeds when env files load successfully", func(t *testing.T) {
		// Create temporary directory with valid env file
		tmpDir := t.TempDir()
		oldDir, err := os.Getwd()
		require.NoError(t, err)
		
		err = os.Chdir(tmpDir)
		require.NoError(t, err)
		defer func() {
			err := os.Chdir(oldDir)
			require.NoError(t, err)
		}()

		// Create a valid .env.local file
		envContent := "OTEL_SERVICE_NAME=test-service\n"
		err = os.WriteFile(filepath.Join(tmpDir, ".env.local"), []byte(envContent), 0644)
		require.NoError(t, err)

		t.Setenv("ENVIRONMENT", "")

		app, err := webapp.New()
		assert.NoError(t, err)
		assert.NotNil(t, app)
		assert.NotNil(t, app.Router)
	})

	t.Run("New() succeeds when GO_ENV=test (skips env loading)", func(t *testing.T) {
		// Set GO_ENV to test to skip env file loading
		t.Setenv("GO_ENV", "test")
		t.Setenv("ENVIRONMENT", "")

		app, err := webapp.New()
		assert.NoError(t, err)
		assert.NotNil(t, app)
		assert.NotNil(t, app.Router)
	})
}

