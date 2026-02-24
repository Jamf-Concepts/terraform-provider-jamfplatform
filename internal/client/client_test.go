// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleAPIResponse_Success(t *testing.T) {
	c := &Client{}
	body := `{"id":"test-1","name":"Test"}`

	req, _ := http.NewRequest(http.MethodGet, "http://example.com/api/test", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
		Header:     http.Header{},
	}

	var result struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	err := c.handleAPIResponse(context.Background(), resp, http.StatusOK, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "test-1" {
		t.Errorf("expected ID 'test-1', got %q", result.ID)
	}
	if result.Name != "Test" {
		t.Errorf("expected Name 'Test', got %q", result.Name)
	}
}

func TestHandleAPIResponse_NilResult(t *testing.T) {
	c := &Client{}

	req, _ := http.NewRequest(http.MethodDelete, "http://example.com/api/delete", nil)
	resp := &http.Response{
		StatusCode: http.StatusNoContent,
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
		Header:     http.Header{},
	}

	err := c.handleAPIResponse(context.Background(), resp, http.StatusNoContent, nil)
	if err != nil {
		t.Fatalf("unexpected error for nil result: %v", err)
	}
}

func TestHandleAPIResponse_StatusMismatch_RawBody(t *testing.T) {
	c := &Client{}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com/api/test", nil)
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("internal error occurred")),
		Request:    req,
		Header:     http.Header{},
	}

	err := c.handleAPIResponse(context.Background(), resp, http.StatusOK, nil)
	if err == nil {
		t.Fatal("expected error for status mismatch")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain '500', got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "internal error occurred") {
		t.Errorf("expected error to contain raw body, got %q", err.Error())
	}
}

func TestHandleAPIResponse_StatusMismatch_APIError(t *testing.T) {
	c := &Client{}

	apiErrBody := `{
		"httpStatus": 400,
		"traceId": "trace-123",
		"errors": [
			{"code": "INVALID_FIELD", "field": "name", "description": "Name is required"}
		]
	}`

	req, _ := http.NewRequest(http.MethodPost, "http://example.com/api/create", nil)
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(apiErrBody)),
		Request:    req,
		Header:     http.Header{},
	}

	err := c.handleAPIResponse(context.Background(), resp, http.StatusCreated, nil)
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !strings.Contains(err.Error(), "trace-123") {
		t.Errorf("expected traceId in error, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "INVALID_FIELD") {
		t.Errorf("expected error code in error, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "Name is required") {
		t.Errorf("expected error description in error, got %q", err.Error())
	}
}

func TestHandleAPIResponse_MalformedJSON(t *testing.T) {
	c := &Client{}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com/api/test", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("not valid json")),
		Request:    req,
		Header:     http.Header{},
	}

	var result struct{ ID string }
	err := c.handleAPIResponse(context.Background(), resp, http.StatusOK, &result)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "failed to decode response") {
		t.Errorf("expected decode error, got %q", err.Error())
	}
}

func TestHandleAPIResponse_EmptyBody_StatusMismatch(t *testing.T) {
	c := &Client{}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com/api/test", nil)
	resp := &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
		Header:     http.Header{},
	}

	err := c.handleAPIResponse(context.Background(), resp, http.StatusOK, nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected '404' in error, got %q", err.Error())
	}
}

func TestHandleAPIResponse_MultipleErrors(t *testing.T) {
	c := &Client{}

	apiErrBody := `{
		"httpStatus": 422,
		"traceId": "trace-456",
		"errors": [
			{"code": "REQUIRED", "field": "name", "description": "Name is required"},
			{"code": "INVALID", "field": "email", "description": "Email format invalid"}
		]
	}`

	req, _ := http.NewRequest(http.MethodPost, "http://example.com/api/create", nil)
	resp := &http.Response{
		StatusCode: http.StatusUnprocessableEntity,
		Body:       io.NopCloser(strings.NewReader(apiErrBody)),
		Request:    req,
		Header:     http.Header{},
	}

	err := c.handleAPIResponse(context.Background(), resp, http.StatusCreated, nil)
	if err == nil {
		t.Fatal("expected error for multiple API errors")
	}
	if !strings.Contains(err.Error(), "Name is required") {
		t.Errorf("expected first error in message, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "Email format invalid") {
		t.Errorf("expected second error in message, got %q", err.Error())
	}
}

func TestBuildURL_WithLeadingSlash(t *testing.T) {
	c := &Client{baseURL: "https://api.example.com"}
	url := c.buildURL("/api/v1/test")
	if url != "https://api.example.com/api/v1/test" {
		t.Errorf("expected 'https://api.example.com/api/v1/test', got %q", url)
	}
}

func TestBuildURL_WithoutLeadingSlash(t *testing.T) {
	c := &Client{baseURL: "https://api.example.com"}
	url := c.buildURL("api/v1/test")
	if url != "https://api.example.com/api/v1/test" {
		t.Errorf("expected 'https://api.example.com/api/v1/test', got %q", url)
	}
}

func TestDoRequest_SetsContentType(t *testing.T) {
	var capturedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	body := map[string]string{"key": "value"}
	_, err := c.doRequest(context.Background(), http.MethodPost, "/test", body, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedContentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", capturedContentType)
	}
}

func TestDoRequest_PatchContentType(t *testing.T) {
	var capturedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	body := map[string]string{"key": "value"}
	_, err := c.doRequest(context.Background(), http.MethodPatch, "/test", body, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedContentType != "application/merge-patch+json" {
		t.Errorf("expected Content-Type 'application/merge-patch+json', got %q", capturedContentType)
	}
}

func TestDoRequest_CustomContentType(t *testing.T) {
	var capturedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	body := map[string]string{"key": "value"}
	_, err := c.doRequest(context.Background(), http.MethodPost, "/test", body, "text/plain")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedContentType != "text/plain" {
		t.Errorf("expected Content-Type 'text/plain', got %q", capturedContentType)
	}
}

func TestDoRequest_NilBody_NoContentType(t *testing.T) {
	var capturedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := &Client{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	_, err := c.doRequest(context.Background(), http.MethodGet, "/test", nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedContentType != "" {
		t.Errorf("expected no Content-Type for nil body, got %q", capturedContentType)
	}
}
