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
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func TestServerlessHandler_CaddyModule(t *testing.T) {
	handler := ServerlessHandler{}
	info := handler.CaddyModule()
	
	if info.ID != "http.handlers.serverless" {
		t.Errorf("expected module ID 'http.handlers.serverless', got '%s'", info.ID)
	}
	
	if info.New == nil {
		t.Error("expected New function to be set")
	}
	
	instance := info.New()
	if instance == nil {
		t.Error("expected New function to return non-nil instance")
	}
	
	if _, ok := instance.(*ServerlessHandler); !ok {
		t.Error("expected New function to return *ServerlessHandler")
	}
}

func TestServerlessHandler_Provision(t *testing.T) {
	tests := []struct {
		name        string
		handler     ServerlessHandler
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			handler: ServerlessHandler{
				Functions: []FunctionConfig{
					{
						Methods: []string{"GET"},
						Path:    "/api/test",
						Image:   "nginx:latest",
						Port:    8080,
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing image",
			handler: ServerlessHandler{
				Functions: []FunctionConfig{
					{
						Methods: []string{"GET"},
						Path:    "/api/test",
					},
				},
			},
			expectError: true,
			errorMsg:    "image is required",
		},
		{
			name: "missing methods",
			handler: ServerlessHandler{
				Functions: []FunctionConfig{
					{
						Path:  "/api/test",
						Image: "nginx:latest",
					},
				},
			},
			expectError: true,
			errorMsg:    "at least one method is required",
		},
		{
			name: "missing path",
			handler: ServerlessHandler{
				Functions: []FunctionConfig{
					{
						Methods: []string{"GET"},
						Image:   "nginx:latest",
					},
				},
			},
			expectError: true,
			errorMsg:    "path is required",
		},
		{
			name: "invalid regex path",
			handler: ServerlessHandler{
				Functions: []FunctionConfig{
					{
						Methods: []string{"GET"},
						Path:    "[invalid regex",
						Image:   "nginx:latest",
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid path regex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := caddy.NewContext(caddy.Context{Context: context.Background()})
			defer cancel()
			err := tt.handler.Provision(ctx)
			
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				
				// Check defaults were set
				if len(tt.handler.Functions) > 0 {
					fn := tt.handler.Functions[0]
					if fn.Port == 0 {
						t.Error("expected default port to be set")
					}
					if fn.Timeout == 0 {
						t.Error("expected default timeout to be set")
					}
					if fn.pathRegex == nil {
						t.Error("expected path regex to be compiled")
					}
				}
			}
		})
	}
}

func TestServerlessHandler_Validate(t *testing.T) {
	tests := []struct {
		name        string
		handler     ServerlessHandler
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			handler: ServerlessHandler{
				Functions: []FunctionConfig{
					{
						Methods: []string{"GET", "POST"},
						Volumes: []VolumeMount{
							{Source: "/host/path", Target: "/container/path"},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid HTTP method",
			handler: ServerlessHandler{
				Functions: []FunctionConfig{
					{
						Methods: []string{"INVALID"},
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid HTTP method",
		},
		{
			name: "volume missing source",
			handler: ServerlessHandler{
				Functions: []FunctionConfig{
					{
						Methods: []string{"GET"},
						Volumes: []VolumeMount{
							{Target: "/container/path"},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "source path is required",
		},
		{
			name: "volume missing target",
			handler: ServerlessHandler{
				Functions: []FunctionConfig{
					{
						Methods: []string{"GET"},
						Volumes: []VolumeMount{
							{Source: "/host/path"},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "target path is required",
		},
		{
			name: "volume relative source path",
			handler: ServerlessHandler{
				Functions: []FunctionConfig{
					{
						Methods: []string{"GET"},
						Volumes: []VolumeMount{
							{Source: "relative/path", Target: "/container/path"},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "source path must be absolute",
		},
		{
			name: "volume relative target path",
			handler: ServerlessHandler{
				Functions: []FunctionConfig{
					{
						Methods: []string{"GET"},
						Volumes: []VolumeMount{
							{Source: "/host/path", Target: "relative/path"},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "target path must be absolute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.handler.Validate()
			
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestServerlessHandler_findMatchingFunction(t *testing.T) {
	handler := ServerlessHandler{
		Functions: []FunctionConfig{
			{
				Methods:   []string{"GET", "POST"},
				Path:      "/api/users/.*",
				pathRegex: regexp.MustCompile("/api/users/.*"),
			},
			{
				Methods:   []string{"DELETE"},
				Path:      "/api/admin/.*",
				pathRegex: regexp.MustCompile("/api/admin/.*"),
			},
		},
	}

	tests := []struct {
		name           string
		method         string
		path           string
		expectedMatch  bool
		expectedIndex  int
	}{
		{
			name:          "GET request matching first function",
			method:        "GET",
			path:          "/api/users/123",
			expectedMatch: true,
			expectedIndex: 0,
		},
		{
			name:          "POST request matching first function",
			method:        "POST",
			path:          "/api/users/create",
			expectedMatch: true,
			expectedIndex: 0,
		},
		{
			name:          "DELETE request matching second function",
			method:        "DELETE",
			path:          "/api/admin/delete",
			expectedMatch: true,
			expectedIndex: 1,
		},
		{
			name:          "no method match",
			method:        "PUT",
			path:          "/api/users/123",
			expectedMatch: false,
		},
		{
			name:          "no path match",
			method:        "GET",
			path:          "/other/path",
			expectedMatch: false,
		},
		{
			name:          "case insensitive method match",
			method:        "get",
			path:          "/api/users/123",
			expectedMatch: true,
			expectedIndex: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			
			result := handler.findMatchingFunction(req)
			
			if tt.expectedMatch {
				if result == nil {
					t.Error("expected to find matching function but got nil")
				} else {
					expectedFunc := &handler.Functions[tt.expectedIndex]
					if result != expectedFunc {
						t.Errorf("expected function at index %d, got different function", tt.expectedIndex)
					}
				}
			} else {
				if result != nil {
					t.Error("expected no matching function but got one")
				}
			}
		})
	}
}

func TestServerlessHandler_ServeHTTP_NoMatch(t *testing.T) {
	handler := ServerlessHandler{
		Functions: []FunctionConfig{
			{
				Methods:   []string{"GET"},
				Path:      "/api/test",
				pathRegex: regexp.MustCompile("/api/test"),
			},
		},
	}

	// Mock next handler
	nextCalled := false
	next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
		return nil
	})

	req := fakeRequest("POST", "/other/path")
	w := httptest.NewRecorder()

	err := handler.ServeHTTP(w, req, next)
	
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	if !nextCalled {
		t.Error("expected next handler to be called")
	}
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestParseVolumeSpec(t *testing.T) {
	tests := []struct {
		name        string
		spec        string
		expected    VolumeMount
		expectError bool
	}{
		{
			name: "basic volume mount",
			spec: "/host/path:/container/path",
			expected: VolumeMount{
				Source: "/host/path",
				Target: "/container/path",
			},
			expectError: false,
		},
		{
			name: "read-only volume mount",
			spec: "/host/path:/container/path:ro",
			expected: VolumeMount{
				Source:   "/host/path",
				Target:   "/container/path",
				ReadOnly: true,
			},
			expectError: false,
		},
		{
			name:        "invalid format - missing target",
			spec:        "/host/path",
			expectError: true,
		},
		{
			name:        "invalid format - too many parts",
			spec:        "/host/path:/container/path:ro:extra",
			expectError: true,
		},
		{
			name:        "invalid option",
			spec:        "/host/path:/container/path:invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseVolumeSpec(tt.spec)
			
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				
				if result.Source != tt.expected.Source {
					t.Errorf("expected source '%s', got '%s'", tt.expected.Source, result.Source)
				}
				
				if result.Target != tt.expected.Target {
					t.Errorf("expected target '%s', got '%s'", tt.expected.Target, result.Target)
				}
				
				if result.ReadOnly != tt.expected.ReadOnly {
					t.Errorf("expected readonly %v, got %v", tt.expected.ReadOnly, result.ReadOnly)
				}
			}
		})
	}
}

func TestUnmarshalCaddyfile(t *testing.T) {
	input := `serverless {
		function {
			methods GET POST
			path /api/.*
			image nginx:latest
			command /bin/sh -c "echo hello"
			env KEY=value
			env ANOTHER=test
			volume /host:/container
			volume /host2:/container2:ro
			timeout 30s
			port 8080
		}
	}`

	d := caddyfile.NewTestDispenser(input)
	handler := &ServerlessHandler{}
	
	err := handler.UnmarshalCaddyfile(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if len(handler.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(handler.Functions))
	}
	
	fn := handler.Functions[0]
	
	// Check methods
	expectedMethods := []string{"GET", "POST"}
	if len(fn.Methods) != len(expectedMethods) {
		t.Errorf("expected %d methods, got %d", len(expectedMethods), len(fn.Methods))
	}
	for i, method := range expectedMethods {
		if fn.Methods[i] != method {
			t.Errorf("expected method '%s', got '%s'", method, fn.Methods[i])
		}
	}
	
	// Check other fields
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
	
	if fn.Environment["ANOTHER"] != "test" {
		t.Errorf("expected env ANOTHER=test, got '%s'", fn.Environment["ANOTHER"])
	}
	
	if len(fn.Volumes) != 2 {
		t.Errorf("expected 2 volumes, got %d", len(fn.Volumes))
	}
	
	if fn.Timeout != caddy.Duration(30*time.Second) {
		t.Errorf("expected timeout 30s, got %v", fn.Timeout)
	}
	
	if fn.Port != 8080 {
		t.Errorf("expected port 8080, got %d", fn.Port)
	}
}

// Helper function to create a fake request with context
func fakeRequest(method, path string) *http.Request {
	r, _ := http.NewRequest(method, path, nil)
	repl := caddy.NewReplacer()
	ctx := context.WithValue(r.Context(), caddy.ReplacerCtxKey, repl)
	r = r.WithContext(ctx)
	return r
}
