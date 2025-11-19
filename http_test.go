package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupMockAPI(response string, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(response))
	}))
}

func clearCache() {
	cache = make(map[string]CacheItem)
}

func TestValidConversion(t *testing.T) {
	clearCache()

	mockServer := setupMockAPI(`{"conversion_result": 42}`, http.StatusOK)
	defer mockServer.Close()

	originalHttpGet := httpGet
	defer func() { httpGet = originalHttpGet }()

	httpGet = func(_ string) (*http.Response, error) {
		return http.Get(mockServer.URL)
	}

	req := httptest.NewRequest(http.MethodGet, "/convert?from=USD&to=EUR&amount=10", nil)
	w := httptest.NewRecorder()

	convertHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var data map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	if val, ok := data["result"]; !ok || val != 42.0 {
		t.Fatalf("Expected result in response")
	}
}

func TestRatesHandler(t *testing.T) {
	clearCache()

	mockServer := setupMockAPI(`{"conversion_rates":{"EUR":0.9,"JPY":111}}`, http.StatusOK)
	defer mockServer.Close()

	originalHttpGet := httpGet
	defer func() { httpGet = originalHttpGet }()

	httpGet = func(_ string) (*http.Response, error) {
		return http.Get(mockServer.URL)
	}

	req := httptest.NewRequest(http.MethodGet, "/rates?base=USD", nil)
	w := httptest.NewRecorder()

	ratesHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if data["base"] != "USD" {
		t.Fatalf("Expected base USD, got %v", data["base"])
	}

	rates := data["rates"].(map[string]interface{})
	if rates["EUR"] != 0.9 || rates["JPY"] != 111.0 {
		t.Fatalf("Expected rates EUR:0.9, JPY:111, got %v", rates)
	}
}
func TestSameCurrency(t *testing.T) {
	clearCache()
	req := httptest.NewRequest(http.MethodGet, "/convert?from=USD&to=USD&amount=10", nil)
	w := httptest.NewRecorder()

	convertHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for same currency, got %d", resp.StatusCode)
	}
}

func TestInvalidAmount(t *testing.T) {
	clearCache()
	req := httptest.NewRequest(http.MethodGet, "/convert?from=USD&to=EUR&amount=abc", nil)
	w := httptest.NewRecorder()

	convertHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for invalid amount, got %d", resp.StatusCode)
	}
}

func TestRatesMissingBase(t *testing.T) {
	clearCache()
	req := httptest.NewRequest(http.MethodGet, "/rates", nil)
	w := httptest.NewRecorder()

	ratesHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request when 'base' is missing, got %d", resp.StatusCode)
	}
}

func TestInvalidCurrencyFormat(t *testing.T) {
	clearCache()
	testCases := []struct {
		from   string
		to     string
		amount string
	}{
		{"US1", "EUR", "10"},
		{"U$D", "EUR", "10"},
		{"USD", "E#R", "10"},
		{"USD", "E1R", "10"},
	}

	for _, tc := range testCases {
		url := fmt.Sprintf("/convert?from=%s&to=%s&amount=%s", tc.from, tc.to, tc.amount)
		req := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()

		convertHandler(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Expected 400 Bad Request for from=%s, to=%s, amount=%s; got %d", tc.from, tc.to, tc.amount, resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "Currency codes must contain only alphabetic letters") {
			t.Errorf("Expected error message about invalid currency format, got %s", string(body))
		}
	}
}

func TestCacheHit(t *testing.T) {
	clearCache()
	cacheKey := fmt.Sprintf("convert:%s : %s : %f", "USD", "EUR", 10.0)
	setToCache(cacheKey, 99.0)

	req := httptest.NewRequest(http.MethodGet, "/convert?from=USD&to=EUR&amount=10", nil)
	w := httptest.NewRecorder()

	convertHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 for cache hit, got %d", resp.StatusCode)
	}

	var data map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	if data["result"] != 99.0 {
		t.Fatalf("Expected cache result 99, got %v", data["result"])
	}
}
func TestExternalAPIAvailable(t *testing.T) {
	clearCache()

	originalAPIKey := apiKey
	apiKey = "test-key"
	defer func() { apiKey = originalAPIKey }()

	mockerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pair/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"conversion_result": 150.75}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockerServer.Close()

	originalHttpGet := httpGet
	httpGet = func(_ string) (*http.Response, error) {
		return http.Get(mockerServer.URL + "/v6/test-key/pair/USD/BRL/50")
	}
	defer func() { httpGet = originalHttpGet }()

	req := httptest.NewRequest(http.MethodGet, "/convert?from=USD&to=BRL&amount=50", nil)
	w := httptest.NewRecorder()

	convertHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200 when API is available, got %d. Body: %s", resp.StatusCode, string(body))
	}

	var data map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	if data["result"] != 150.75 {
		t.Fatalf("Expected result 150.75, got %v", data["result"])
	}
}

func TestRatesExternalAPIAvailable(t *testing.T) {
	clearCache()

	mockerServer := setupMockAPI(`{"conversion_rates":{"BRL":5.45,"EUR":0.92,"GBP":0.79}}`, http.StatusOK)
	defer mockerServer.Close()

	originalHttpGet := httpGet
	defer func() { httpGet = originalHttpGet }()

	httpGet = func(_ string) (*http.Response, error) {
		return http.Get(mockerServer.URL)
	}

	req := httptest.NewRequest(http.MethodGet, "/rates?base=USD", nil)
	w := httptest.NewRecorder()

	ratesHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 when rates API is available, got %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	rates := data["rates"].(map[string]interface{})
	if rates["BRL"] != 5.45 || rates["EUR"] != 0.92 || rates["GBP"] != 0.79 {
		t.Fatalf("Expected rates from external API, got %v", rates)
	}
}
