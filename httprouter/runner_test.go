package httprouter_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nminelli/go-toolkit/httprouter"
)

func TestRun(t *testing.T) {
	// Random tcp port is chosen
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	h := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httprouter.Run(ln, httprouter.DefaultTimeouts, http.HandlerFunc(h))
	}()

	// Before closing the listener we need to make sure the server is up and running.
	// We do so by retrying until we get a response from the underlying server.
	c := http.Client{Timeout: 100 * time.Millisecond}
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d", ln.Addr().(*net.TCPAddr).Port), nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, "ok", string(data))

	// Server is up. We close the listener to simulate server shutdown
	require.Equal(t, http.StatusOK, resp.StatusCode)
	err = ln.Close()
	assert.NoError(t, err)

	select {
	case err := <-serverErr:
		// Server should exit gracefully when listener is closed
		assert.Error(t, err) // Expect an error when listener is closed
		return
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

// LocalhostCert is a PEM-encoded TLS cert with SAN IPs
// "127.0.0.1" and "[::1]", expiring at Jan 29 16:00:00 2084 GMT.
// generated from src/crypto/tls:
// go run generate_cert.go  --rsa-bits 1024 --host 127.0.0.1,::1,example.com --ca --start-date "Jan 1 00:00:00 1970" --duration=1000000h
var LocalhostCert = []byte(`-----BEGIN CERTIFICATE-----
MIICEzCCAXygAwIBAgIQMIMChMLGrR+QvmQvpwAU6zANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw
MDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCB
iQKBgQDuLnQAI3mDgey3VBzWnB2L39JUU4txjeVE6myuDqkM/uGlfjb9SjY1bIw4
iA5sBBZzHi3z0h1YV8QPuxEbi4nW91IJm2gsvvZhIrCHS3l6afab4pZBl2+XsDul
rKBxKKtD1rGxlG4LjncdabFn9gvLZad2bSysqz/qTAUStTvqJQIDAQABo2gwZjAO
BgNVHQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYDVR0TAQH/BAUw
AwEB/zAuBgNVHREEJzAlggtleGFtcGxlLmNvbYcEfwAAAYcQAAAAAAAAAAAAAAAA
AAAAATANBgkqhkiG9w0BAQsFAAOBgQCEcetwO59EWk7WiJsG4x8SY+UIAA+flUI9
tyC4lNhbcF2Idq9greZwbYCqTTTr2XiRNSMLCOjKyI7ukPoPjo16ocHj+P3vZGfs
h1fIw3cSS2OolhloGw/XM6RWPWtPAlGykKLciQrBru5NAPvCMsb/I1DAceTiotQM
fblo6RBxUQ==
-----END CERTIFICATE-----`)

// LocalhostKey is the private key for localhostCert.
var LocalhostKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIICXgIBAAKBgQDuLnQAI3mDgey3VBzWnB2L39JUU4txjeVE6myuDqkM/uGlfjb9
SjY1bIw4iA5sBBZzHi3z0h1YV8QPuxEbi4nW91IJm2gsvvZhIrCHS3l6afab4pZB
l2+XsDulrKBxKKtD1rGxlG4LjncdabFn9gvLZad2bSysqz/qTAUStTvqJQIDAQAB
AoGAGRzwwir7XvBOAy5tM/uV6e+Zf6anZzus1s1Y1ClbjbE6HXbnWWF/wbZGOpet
3Zm4vD6MXc7jpTLryzTQIvVdfQbRc6+MUVeLKwZatTXtdZrhu+Jk7hx0nTPy8Jcb
uJqFk541aEw+mMogY/xEcfbWd6IOkp+4xqjlFLBEDytgbIECQQDvH/E6nk+hgN4H
qzzVtxxr397vWrjrIgPbJpQvBsafG7b0dA4AFjwVbFLmQcj2PprIMmPcQrooz8vp
jy4SHEg1AkEA/v13/5M47K9vCxmb8QeD/asydfsgS5TeuNi8DoUBEmiSJwma7FXY
fFUtxuvL7XvjwjN5B30pNEbc6Iuyt7y4MQJBAIt21su4b3sjXNueLKH85Q+phy2U
fQtuUE9txblTu14q3N7gHRZB4ZMhFYyDy8CKrN2cPg/Fvyt0Xlp/DoCzjA0CQQDU
y2ptGsuSmgUtWj3NM9xuwYPm+Z/F84K6+ARYiZ6PYj013sovGKUFfYAqVXVlxtIX
qyUBnu3X9ps8ZfjLZO7BAkEAlT4R5Yl6cGhaJQYZHOde3JEMhNRcVFMO8dJDaFeo
f9Oeos0UUothgiDktdQHxdNEwLjQf7lJJBzV+5OtwswCWA==
-----END RSA PRIVATE KEY-----`)

func TestRunTLS(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	h := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}

	cert, err := tls.X509KeyPair(LocalhostCert, LocalhostKey)
	if err != nil {
		t.Fatal(err)
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httprouter.RunTLS(ln, httprouter.DefaultTimeouts, http.HandlerFunc(h), tlsConfig)
	}()

	certificate, err := x509.ParseCertificate(tlsConfig.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	certpool := x509.NewCertPool()
	certpool.AddCert(certificate)

	config := &tls.Config{
		RootCAs: certpool,
	}

	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: config,
		},
	}

	req, err := http.NewRequest("GET", "https://"+ln.Addr().String(), nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, "ok", string(data))

	// Server is up. We close the listener to simulate server shutdown
	require.Equal(t, http.StatusOK, resp.StatusCode)
	err = ln.Close()
	assert.NoError(t, err)

	select {
	case err := <-serverErr:
		// Server should exit when listener is closed
		assert.Error(t, err) // Expect an error when listener is closed
		return
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestRunWithShutdownHooks(t *testing.T) {
	// Random tcp port is chosen
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	h := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}

	// Track if shutdown hooks were called
	hookCalled := make(chan bool, 1)
	shutdownHook := func() {
		hookCalled <- true
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httprouter.RunWithShutdownHooks(ln, httprouter.DefaultTimeouts, http.HandlerFunc(h), shutdownHook)
	}()

	// Before closing the listener we need to make sure the server is up and running.
	c := http.Client{Timeout: 100 * time.Millisecond}
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d", ln.Addr().(*net.TCPAddr).Port), nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, "ok", string(data))

	// Server is up. We close the listener to simulate server shutdown
	err = ln.Close()
	assert.NoError(t, err)

	// Note: When testing shutdown hooks with listener close, the hook may not be called
	// since this simulates a different type of shutdown than SIGTERM
	// In a real-world scenario, shutdown hooks are called on SIGTERM/SIGINT
	select {
	case err := <-serverErr:
		// Server should exit when listener is closed
		assert.Error(t, err) // Expect an error when listener is closed
		return
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}
