package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var apiKey string
var httpGet = http.Get

func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		log.Println("WARNING: Error loading .env file (using system env vars):", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(parts[0], parts[1])
		}
	}
}

func checkCurrencyFormat(s string) bool {
	re := regexp.MustCompile(`^[A-Z]+$`)
	return !re.MatchString(s)
}

type CacheItem struct {
	Data      interface{}
	Timestamp int64
}

var cache = make(map[string]CacheItem)
var cacheTime int64 = 300

func getFromCache(key string) (interface{}, bool) {
	item, ok := cache[key]
	if !ok {
		return nil, false
	}

	if time.Now().Unix()-item.Timestamp > cacheTime {
		delete(cache, key)
		return nil, false
	}
	return item.Data, true
}

func setToCache(key string, data interface{}) {
	cache[key] = CacheItem{
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}

func convertHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	from := strings.ToUpper(query.Get("from"))
	to := strings.ToUpper(query.Get("to"))
	amountStr := query.Get("amount")

	log.Printf("RECEIVED [Convert]: Method=%s | Params: from=%s, to=%s, amount=%s.", r.Method, from, to, amountStr)

	if from == "" || to == "" || amountStr == "" {
		log.Printf("ERROR [Convert]: Method not allowed (%s)", r.Method)
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	if from == to {
		log.Printf("ERROR [Convert]: Same currencies (%s)", from)
		http.Error(w, "Source and target currencies must be different", http.StatusBadRequest)
		return
	}

	if checkCurrencyFormat(from) || checkCurrencyFormat(to) {
		log.Printf("ERROR [Convert]: Invalid currency format (%s, %s)", from, to)
		http.Error(w, "Currency codes must contain only alphabetic letters (no number or symbols)", http.StatusBadRequest)
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		log.Printf("ERROR [Convert]: Invalid amount (%s)", amountStr)
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	if amount <= 0 {
		log.Printf("ERROR [Convert]: Negative or zero amount(%f)", amount)
		http.Error(w, "Amount must be greater than zero", http.StatusBadRequest)
		return
	}

	cacheKey := fmt.Sprintf("convert:%s : %s : %f", from, to, amount)
	if cachedData, found := getFromCache(cacheKey); found {
		log.Printf("SUCCESS [Cache]: %f %s -> %s (Returned from Cache)", amount, from, to)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"result": cachedData})
		return
	}

	url := fmt.Sprintf("https://v6.exchangerate-api.com/v6/%s/pair/%s/%s/%f", apiKey, from, to, amount)
	res, err := httpGet(url)
	if err != nil {
		log.Printf("ERROR [API]: External request failed: %v", err)
		http.Error(w, "Error making API request:", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("ERROR [API]: External API error: %s", res.Status)
		http.Error(w, fmt.Sprintf("External API error: %s", res.Status), http.StatusInternalServerError)
		return
	}

	var apiResponse struct {
		Result float64 `json:"conversion_result"`
	}

	if err = json.NewDecoder(res.Body).Decode(&apiResponse); err != nil {
		log.Printf("ERROR [JSON]: Failed to parse API response: %v", err)
		http.Error(w, "Error parsing API response", http.StatusInternalServerError)
		return
	}

	setToCache(cacheKey, apiResponse.Result)

	log.Printf("SUCCESS [Convert]: %f %s -> %f %s", amount, from, apiResponse.Result, to)

	response := map[string]interface{}{"result": apiResponse.Result}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func ratesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	base := strings.ToUpper(query.Get("base"))

	log.Printf("RECEIVED [Rates]: Method=%s | Params: base=%s.", r.Method, base)

	if base == "" {
		log.Printf("ERROR [Rates]: Missing base parameter")
		http.Error(w, "Missing required parameter: base", http.StatusBadRequest)
		return
	}

	if checkCurrencyFormat(base) {
		log.Printf("ERROR [Rates]: Invalid format for base (%s)", base)
		http.Error(w, "Currency codes must contain only alphabetic letters (no number or symbols)", http.StatusBadRequest)
		return
	}

	cacheKey := fmt.Sprintf("rates:%s", base)
	if cachedData, found := getFromCache(cacheKey); found {
		log.Printf("SUCCESS [Cache]: Rates for base %s (Returned from Cache)", base)
		response := map[string]interface{}{"base": base, "rates": cachedData}
		w.Header().Set("Content-Type", "application/json")
		jsonData, _ := json.MarshalIndent(response, "", "  ")
		w.Write(jsonData)
		return
	}

	url := fmt.Sprintf("https://v6.exchangerate-api.com/v6/%s/latest/%s", apiKey, base)
	res, err := httpGet(url)
	if err != nil {
		log.Printf("ERROR [API]: External request failed: %v", err)
		http.Error(w, "Error making API request", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("ERROR [API]: External API error: %s", res.Status)
		http.Error(w, fmt.Sprintf("External API error: %s", res.Status), http.StatusInternalServerError)
		return
	}
	var apiResponse struct {
		Rates map[string]float64 `json:"conversion_rates"`
	}

	if err = json.NewDecoder(res.Body).Decode(&apiResponse); err != nil {
		log.Printf("ERROR [JSON]: Failed to parse API response: %v", err)
		http.Error(w, "Error parsing API response", http.StatusInternalServerError)
		return
	}

	setToCache(cacheKey, apiResponse.Rates)

	log.Printf("SUCCESS [Rates]: Rates for base %s", base)

	response := map[string]interface{}{"base": base, "rates": apiResponse.Rates}
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("ERROR [JSON]: Failed to format JSON response: %v", err)
		http.Error(w, "Error formatting JSON response", http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func main() {
	loadEnv()

	apiKey = os.Getenv("API_KEY_EXCHANGE")
	if apiKey == "" {
		log.Println("API_KEY_EXCHANGE environment variable not set")
	}

	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", fs)

	http.HandleFunc("/convert", enableCORS(convertHandler))
	http.HandleFunc("/rates", enableCORS(ratesHandler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server is running on port %s...\n", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
