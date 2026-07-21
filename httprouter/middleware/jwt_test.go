package middleware

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestCobreClaims_IsInternal(t *testing.T) {
	testCases := []struct {
		name           string
		claims         cobreClaims
		expectedResult bool
	}{
		{
			name: "internal cognito issuer",
			claims: cobreClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer: "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_EXAMPLE",
				},
			},
			expectedResult: true,
		},
		{
			name: "internal cognito issuer - different region",
			claims: cobreClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer: "https://cognito-idp.us-west-2.amazonaws.com/us-west-2_TEST",
				},
			},
			expectedResult: true,
		},
		{
			name: "external issuer",
			claims: cobreClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer: "https://external-provider.com",
				},
			},
			expectedResult: false,
		},
		{
			name: "empty issuer",
			claims: cobreClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer: "",
				},
			},
			expectedResult: false,
		},
		{
			name: "partial cognito string in issuer",
			claims: cobreClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer: "https://cognito.example.com",
				},
			},
			expectedResult: true,
		},
		{
			name: "case sensitive - uppercase COGNITO not matched",
			claims: cobreClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer: "https://COGNITO-idp.us-east-1.amazonaws.com/us-east-1_EXAMPLE",
				},
			},
			expectedResult: false,
		},
		{
			name: "cognito in path only",
			claims: cobreClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer: "https://example.com/cognito/test",
				},
			},
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.claims.IsInternal()
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}
