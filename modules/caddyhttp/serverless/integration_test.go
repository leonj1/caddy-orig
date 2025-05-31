// Copyright 2015 Matthew Holt and The Caddy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package serverless

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

// MockContainerManager is a mock implementation for testing
type MockContainerManager struct {
	containers map[string]*Container
	shouldFail bool
}

func NewMockContainerManager() *MockContainerManager {
	return &MockContainerManager{
		containers: make(map[string]*Container),
	}
}

// Ensure MockContainerManager implements ContainerManagerInterface
var _ ContainerManagerInterface = (*MockContainerManager)(nil)

func (m *MockContainerManager) StartContainer(ctx context.Context, config ContainerConfig) (*Container, error) {
	if m.shouldFail {
		return nil, &MockError{message: "mock container start failure"}
	}
	
	container := &Container{
		ID:   "mock-container-id",
		IP:   "127.0.0.1",
		Port: 8080,
	}
	
	m.containers[container.ID] = container
	return container, nil
}

func (m *MockContainerManager) WaitForReady(ctx context.Context, container *Container, timeout time.Duration) error {
	if m.shouldFail {
		return &MockError{message: "mock container not ready"}
	}
	return nil
}

func (m *MockContainerManager) StopContainer(ctx context.Context, containerID string) error {
	delete(m.containers, containerID)
	return nil
}

func (m *MockContainerManager) Cleanup() error {
	m.containers = make(map[string]*Container)
	return nil
}

type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

// TestServerlessHandler_Integration tests the complete flow with mocked Docker
func TestServerlessHandler_Integration(t *testing.T) {
	// Create handler with mock container manager
	handler := &ServerlessHandler{
		Functions: []FunctionConfig{
			{
				Methods:   []string{"GET", "POST"},
				Path:      "/api/test.*",
				Image:     "test:latest",
				Port:      8080,
				Timeout:   caddy.Duration(30 * time.Second),
				pathRegex: regexp.MustCompile("/api/test.*"),
			},
		},
	}

	// Provision the handler to initialize logger
	ctx, cancel := caddy.NewContext(caddy.Context{Context: context.Background()})
	defer cancel()
	err := handler.Provision(ctx)
	if err != nil {
		t.Fatalf("failed to provision handler: %v", err)
	}

	// Replace the container manager with a mock
	mockCM := NewMockContainerManager()
	handler.containerManager = mockCM

	// Create a test request
	req := fakeRequest("GET", "/api/test/123")
	w := httptest.NewRecorder()

	// Mock next handler (should not be called)
	nextCalled := false
	next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		nextCalled = true
		return nil
	})

	// Execute the handler - this will fail at the proxy stage since we can't mock that easily
	// but we can verify that the container management works
	err = handler.ServeHTTP(w, req, next)

	// We expect an error because we can't actually proxy to a real container
	if err == nil {
		t.Error("expected error when trying to proxy to non-existent container")
	}

	if nextCalled {
		t.Error("next handler should not have been called")
	}

	// Check that container was started and stopped
	if len(mockCM.containers) != 0 {
		t.Errorf("expected containers to be cleaned up, but %d remain", len(mockCM.containers))
	}
}

// TestServerlessHandler_NoMatchPassesToNext tests that unmatched requests pass to next handler
func TestServerlessHandler_NoMatchPassesToNext(t *testing.T) {
	handler := &ServerlessHandler{
		Functions: []FunctionConfig{
			{
				Methods:   []string{"GET"},
				Path:      "/api/test",
				pathRegex: regexp.MustCompile("/api/test"),
			},
		},
	}

	req := fakeRequest("POST", "/other/path")
	w := httptest.NewRecorder()

	nextCalled := false
	next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
		return nil
	})

	err := handler.ServeHTTP(w, req, next)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !nextCalled {
		t.Error("next handler should have been called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestServerlessHandler_ContainerStartFailure tests error handling when container fails to start
func TestServerlessHandler_ContainerStartFailure(t *testing.T) {
	handler := &ServerlessHandler{
		Functions: []FunctionConfig{
			{
				Methods:   []string{"GET"},
				Path:      "/api/test",
				Image:     "test:latest",
				pathRegex: regexp.MustCompile("/api/test"),
			},
		},
	}

	// Provision the handler to initialize logger
	ctx, cancel := caddy.NewContext(caddy.Context{Context: context.Background()})
	defer cancel()
	err := handler.Provision(ctx)
	if err != nil {
		t.Fatalf("failed to provision handler: %v", err)
	}

	// Use a mock container manager that fails
	mockCM := NewMockContainerManager()
	mockCM.shouldFail = true
	handler.containerManager = mockCM

	req := fakeRequest("GET", "/api/test")
	w := httptest.NewRecorder()

	next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	err = handler.ServeHTTP(w, req, next)

	// Should return an error
	if err == nil {
		t.Error("expected error when container fails to start")
	}

	// Should be a HandlerError with 500 status
	if handlerErr, ok := err.(caddyhttp.HandlerError); ok {
		if handlerErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", handlerErr.StatusCode)
		}
	} else {
		t.Error("expected HandlerError")
	}
}

// TestServerlessHandler_JSONConfiguration tests JSON configuration parsing
func TestServerlessHandler_JSONConfiguration(t *testing.T) {
	jsonConfig := `{
		"functions": [
			{
				"methods": ["GET", "POST"],
				"path": "/api/.*",
				"image": "nginx:latest",
				"command": ["/bin/sh", "-c", "echo hello"],
				"environment": {
					"KEY": "value"
				},
				"volumes": [
					{
						"source": "/host/path",
						"target": "/container/path",
						"readonly": true
					}
				],
				"timeout": "30s",
				"port": 8080
			}
		]
	}`

	var handler ServerlessHandler
	err := json.Unmarshal([]byte(jsonConfig), &handler)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON config: %v", err)
	}

	if len(handler.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(handler.Functions))
	}

	fn := handler.Functions[0]

	// Verify configuration
	expectedMethods := []string{"GET", "POST"}
	if len(fn.Methods) != len(expectedMethods) {
		t.Errorf("expected %d methods, got %d", len(expectedMethods), len(fn.Methods))
	}

	if fn.Path != "/api/.*" {
		t.Errorf("expected path '/api/.*', got '%s'", fn.Path)
	}

	if fn.Image != "nginx:latest" {
		t.Errorf("expected image 'nginx:latest', got '%s'", fn.Image)
	}

	expectedCommand := []string{"/bin/sh", "-c", "echo hello"}
	if len(fn.Command) != len(expectedCommand) {
		t.Errorf("expected %d command args, got %d", len(expectedCommand), len(fn.Command))
	}

	if fn.Environment["KEY"] != "value" {
		t.Errorf("expected env KEY=value, got '%s'", fn.Environment["KEY"])
	}

	if len(fn.Volumes) != 1 {
		t.Errorf("expected 1 volume, got %d", len(fn.Volumes))
	}

	vol := fn.Volumes[0]
	if vol.Source != "/host/path" {
		t.Errorf("expected volume source '/host/path', got '%s'", vol.Source)
	}

	if vol.Target != "/container/path" {
		t.Errorf("expected volume target '/container/path', got '%s'", vol.Target)
	}

	if !vol.ReadOnly {
		t.Error("expected volume to be read-only")
	}

	if fn.Timeout != caddy.Duration(30*time.Second) {
		t.Errorf("expected timeout 30s, got %v", fn.Timeout)
	}

	if fn.Port != 8080 {
		t.Errorf("expected port 8080, got %d", fn.Port)
	}
}

// TestServerlessHandler_Cleanup tests the cleanup functionality
func TestServerlessHandler_Cleanup(t *testing.T) {
	handler := &ServerlessHandler{}
	mockCM := NewMockContainerManager()
	handler.containerManager = mockCM

	// Add some mock containers
	mockCM.containers["container1"] = &Container{ID: "container1"}
	mockCM.containers["container2"] = &Container{ID: "container2"}

	err := handler.Cleanup()
	if err != nil {
		t.Errorf("unexpected error during cleanup: %v", err)
	}

	if len(mockCM.containers) != 0 {
		t.Errorf("expected all containers to be cleaned up, but %d remain", len(mockCM.containers))
	}
}

// TestServerlessHandler_MethodCaseInsensitive tests case-insensitive method matching
func TestServerlessHandler_MethodCaseInsensitive(t *testing.T) {
	handler := &ServerlessHandler{
		Functions: []FunctionConfig{
			{
				Methods:   []string{"GET", "post"},
				Path:      "/test",
				pathRegex: regexp.MustCompile("/test"),
			},
		},
	}

	tests := []struct {
		method      string
		shouldMatch bool
	}{
		{"GET", true},
		{"get", true},
		{"POST", true},
		{"post", true},
		{"PUT", false},
		{"put", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := fakeRequest(tt.method, "/test")
			result := handler.findMatchingFunction(req)

			if tt.shouldMatch && result == nil {
				t.Error("expected to find matching function")
			} else if !tt.shouldMatch && result != nil {
				t.Error("expected no matching function")
			}
		})
	}
}
