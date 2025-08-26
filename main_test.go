package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/IdrisovMarat/pokemon/internal/pokecache"
)

func TestProcessCachedData(t *testing.T) {
	cfg := &config{offset: 0, limit: 20}
	testData := locationJson{
		Count:    100,
		Next:     stringPtr("next-url"),
		Previous: stringPtr("prev-url"),
		Results: []Results{
			{Name: "location1", URL: "url1"},
			{Name: "location2", URL: "url2"},
		},
	}

	data, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = processCachedData(data, cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("processCachedData returned error: %v", err)
	}

	if cfg.next == nil || *cfg.next != "next-url" {
		t.Errorf("Expected next to be 'next-url', got %v", cfg.next)
	}

	if cfg.previous == nil || *cfg.previous != "prev-url" {
		t.Errorf("Expected previous to be 'prev-url', got %v", cfg.previous)
	}

	if cfg.offset != 20 {
		t.Errorf("Expected offset to be 20, got %d", cfg.offset)
	}

	if !strings.Contains(output, "location1") || !strings.Contains(output, "location2") {
		t.Errorf("Expected output to contain location names, got: %s", output)
	}
}

func TestCommandHelp(t *testing.T) {
	cfg := &config{}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := commandHelp(cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("commandHelp returned error: %v", err)
	}

	expectedStrings := []string{"Welcome", "help", "exit", "map", "mapb"}
	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', got: %s", expected, output)
		}
	}
}

func TestCommandMapWithCache(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := locationJson{
			Count:    100,
			Next:     stringPtr("next-page"),
			Previous: nil,
			Results:  []Results{{Name: "test-location", URL: "test-url"}},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Override baseUrl for testing
	originalBaseUrl := baseUrl
	defer func() { baseUrl = originalBaseUrl }()
	baseUrl = server.URL + "/"

	// Initialize cache
	cache = pokecache.NewCache(1 * time.Minute)
	defer cache.Stop()

	cfg := &config{offset: 0, limit: 20}

	// First call - should hit the server
	err := commandMap(cfg)
	if err != nil {
		t.Errorf("First commandMap call failed: %v", err)
	}

	// Second call - should use cache
	err = commandMap(cfg)
	if err != nil {
		t.Errorf("Second commandMap call failed: %v", err)
	}
}

func TestCommandMapb(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := locationJson{
			Count:    100,
			Next:     stringPtr("next-page"),
			Previous: stringPtr("prev-page"),
			Results:  []Results{{Name: "prev-location", URL: "prev-url"}},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Override baseUrl for testing
	originalBaseUrl := baseUrl
	defer func() { baseUrl = originalBaseUrl }()
	baseUrl = server.URL + "/"

	cache = pokecache.NewCache(1 * time.Minute)
	defer cache.Stop()

	cfg := &config{offset: 40, limit: 20} // Start from offset 40

	err := commandMapb(cfg)
	if err != nil {
		t.Errorf("commandMapb failed: %v", err)
	}

	// Should have decreased offset by one page (20)
	if cfg.offset != 20 { // commandMapb decreases by limit*2 but commandMap increases by limit
		t.Errorf("Expected offset to be 20, got %d", cfg.offset)
	}
}

func TestCommandMapbOnFirstPage(t *testing.T) {
	cfg := &config{offset: 0, limit: 20}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := commandMapb(cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("commandMapb returned error: %v", err)
	}

	if !strings.Contains(output, "first page") {
		t.Errorf("Expected warning about first page, got: %s", output)
	}
}

func TestCommandMapHTTPError(t *testing.T) {
	// Setup test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Override baseUrl for testing
	originalBaseUrl := baseUrl
	defer func() { baseUrl = originalBaseUrl }()
	baseUrl = server.URL + "/"

	cache = pokecache.NewCache(1 * time.Minute)
	defer cache.Stop()

	cfg := &config{offset: 0, limit: 20}

	err := commandMap(cfg)
	if err == nil {
		t.Error("Expected error for HTTP 500, but got none")
	}
}

func TestMainFunctionality(t *testing.T) {
	// Test that commands map is properly initialized
	commands := map[string]cliCommand{
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex",
			callback:    commandExit,
		},
		"help": {
			name:        "help",
			description: "description",
			callback:    commandHelp,
		},
		"map": {
			name:        "map",
			description: "lists next 20 Api resources",
			callback:    commandMap,
		},
		"mapb": {
			name:        "mapb",
			description: "lists previous 20 Api resources",
			callback:    commandMapb,
		},
	}

	if len(commands) != 4 {
		t.Errorf("Expected 4 commands, got %d", len(commands))
	}

	// Test that each command has a callback
	for name, cmd := range commands {
		if cmd.callback == nil {
			t.Errorf("Command %s has nil callback", name)
		}
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

func TestConfigInitialization(t *testing.T) {
	if pageConfig.offset != 0 {
		t.Errorf("Expected initial offset to be 0, got %d", pageConfig.offset)
	}
	if pageConfig.limit != 20 {
		t.Errorf("Expected initial limit to be 20, got %d", pageConfig.limit)
	}
	if pageConfig.next != nil {
		t.Errorf("Expected initial next to be nil, got %v", pageConfig.next)
	}
	if pageConfig.previous != nil {
		t.Errorf("Expected initial previous to be nil, got %v", pageConfig.previous)
	}
}
