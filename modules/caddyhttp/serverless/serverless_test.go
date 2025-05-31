package serverless

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2/caddytest"
)

const (
	testDockerImageName = "caddy-serverless-echoserver-test"
	testDockerImageTag  = "latest"
	echoServerDir       = "./testdata/echoserver"
)

// Helper to build the test Docker image
func buildTestImage(t *testing.T) string {
	t.Helper()
	imageFullName := fmt.Sprintf("%s:%s", testDockerImageName, testDockerImageTag)

	// Check if image already exists
	cmdCheck := exec.Command("docker", "image", "inspect", imageFullName)
	if err := cmdCheck.Run(); err == nil {
		t.Logf("Docker image %s already exists, skipping build", imageFullName)
		return imageFullName
	}

	t.Logf("Building Docker image %s from %s", imageFullName, echoServerDir)
	cmd := exec.Command("docker", "build", "-t", imageFullName, echoServerDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Attempt to remove partial image if build failed
		cleanupCmd := exec.Command("docker", "rmi", imageFullName)
		cleanupCmd.Run() // Ignore error, just best effort
		t.Fatalf("Failed to build Docker image %s: %v", imageFullName, err)
	}
	return imageFullName
}

// Helper to remove the test Docker image
func removeTestImage(t *testing.T, imageName string) {
	t.Helper()
	if imageName == "" {
		return
	}
	t.Logf("Removing Docker image %s", imageName)
	cmd := exec.Command("docker", "rmi", "-f", imageName) // -f to force remove if containers are using it
	if err := cmd.Run(); err != nil {
		// Don't fail the test for cleanup issues, but log it
		t.Logf("Failed to remove Docker image %s: %v. Manual cleanup might be required.", imageName, err)
	}
}

// Structure for the echoserver's response
type EchoResponse struct {
	Headers http.Header `json:"headers"`
	Body    string      `json:"body"`
}

func TestServerlessPlugin_PostEcho(t *testing.T) {
	// Build the Docker image for the echoserver
	// Skip if Docker is not available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found in PATH, skipping integration test")
	}

	imageFullName := buildTestImage(t)
	// Defer image removal, but only if it was built by this test run (or if we decide to always try removing)
	// For simplicity in this example, we'll always try to remove it.
	// A more robust solution might involve checking if the image existed before the test.
	defer removeTestImage(t, imageFullName)

	// Define Caddy JSON configuration
	caddyJSON := fmt.Sprintf(`
	{
		"apps": {
			"http": {
				"servers": {
					"srv0": {
						"listen": [":{{env.SERVER_PORT}}"],
						"routes": [
							{
								"handle": [{
									"handler": "serverless",
									"functions": [{
										"methods": ["POST"],
										"path": "/echo",
										"image": "%s",
										"port": 8080,
										"timeout": "60s"
									}]
								}]
							}
						]
					}
				}
			}
		}
	}
	`, imageFullName)

	// Initialize Caddy server
	tester := caddytest.NewTester(t)
	tester.InitServer(caddyJSON, "json")
	defer tester.StopServer() // Ensure server is stopped

	// Prepare POST request
	requestPayload := `{"message": "hello from caddy test"}`
	requestBody := bytes.NewBufferString(requestPayload)

	req, err := http.NewRequest("POST", tester.URL+"/echo", requestBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom-Header", "CaddyServerlessTest")
	req.Header.Set("User-Agent", "Caddy-Test-Agent") // To check if User-Agent is passed

	// Send request to Caddy
	client := &http.Client{Timeout: 90 * time.Second} // Increased timeout for Docker startup
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status %d, got %d. Response body: %s", http.StatusOK, resp.StatusCode, string(bodyBytes))
	}

	// Read and unmarshal response body
	responseBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var echoResp EchoResponse
	if err := json.Unmarshal(responseBodyBytes, &echoResp); err != nil {
		t.Fatalf("Failed to unmarshal response JSON: %v. Body: %s", err, string(responseBodyBytes))
	}

	// Verify headers
	if contentType := echoResp.Headers.Get("Content-Type"); !strings.Contains(strings.ToLower(contentType), "application/json") {
		// The echoserver itself sets Content-Type: application/json for its *response*
		// Here we are checking the *request* headers that were echoed back.
		// The original request to Caddy had Content-Type: application/json
		originalRequestContentType := echoResp.Headers.Get("Content-Type") // This is the Content-Type of the request *to the echoserver*
		if !strings.Contains(strings.ToLower(originalRequestContentType), "application/json") {
			t.Errorf("Expected echoed 'Content-Type' header to contain 'application/json', got '%s'", originalRequestContentType)
		}
	}
	if customHeader := echoResp.Headers.Get("X-Custom-Header"); customHeader != "CaddyServerlessTest" {
		t.Errorf("Expected echoed 'X-Custom-Header' to be 'CaddyServerlessTest', got '%s'", customHeader)
	}
    if userAgent := echoResp.Headers.Get("User-Agent"); userAgent != "Caddy-Test-Agent" {
		t.Errorf("Expected echoed 'User-Agent' to be 'Caddy-Test-Agent', got '%s'", userAgent)
	}


	// Verify body
	if echoResp.Body != requestPayload {
		t.Errorf("Expected echoed body to be '%s', got '%s'", requestPayload, echoResp.Body)
	}

	t.Log("Serverless POST echo test completed successfully.")
}

// TestMain can be used for global setup/teardown if needed,
// for example, ensuring Docker is available.
func TestMain(m *testing.M) {
	// Optional: Check for Docker availability globally
	// if _, err := exec.LookPath("docker"); err != nil {
	// 	fmt.Println("SKIPPING serverless tests: Docker not found in PATH.")
	// 	os.Exit(0) // Skip all tests in this package
	// }
	os.Exit(m.Run())
}
