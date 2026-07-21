package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nminelli/go-toolkit/telemetry/log"
)

type cobreClaims struct {
	ClientType string `json:"custom:type"`
	ClientID   string `json:"custom:linked_clients"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	jwt.RegisteredClaims
}

func (t cobreClaims) IsInternal() bool {
	return strings.Contains(t.Issuer, "cognito")
}

func retrieveClientIDFromJWT(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is missing")
	}

	claims, err := extractClaimsFromJWTHeader(authHeader)
	if err != nil {
		log.Error(r.Context(), fmt.Sprintf("Error extracting JWT: %v", err))
		return "", fmt.Errorf("invalid token")
	}

	if claims.ClientType != "api_user" {
		log.Error(r.Context(), fmt.Sprintf("Invalid client type: %s. Only 'api_user' is supported.", claims.ClientType))
		return "", fmt.Errorf("invalid token type")
	}

	if claims.ClientID == "" {
		return "", fmt.Errorf("client ID is missing")
	}
	clientID := strings.Split(claims.ClientID, "|")[0]

	return clientID, nil
}

func extractClaimsFromJWTHeader(authHeader string) (cobreClaims, error) {
	// Remove "Bearer " prefix if present
	jwtString := authHeader
	if strings.HasPrefix(authHeader, "Bearer ") {
		jwtString = authHeader[len("Bearer "):]
	}

	var jwtToken cobreClaims
	parser := jwt.NewParser()
	// For now, we don't use the token, but at least we validate its structure and integrate the lib
	// with the JWT library to be used in the future if needed
	_, _, err := parser.ParseUnverified(jwtString, &jwtToken)
	if err != nil {
		return cobreClaims{}, fmt.Errorf("invalid JWT format: %v", err)
	}

	return jwtToken, nil
}
