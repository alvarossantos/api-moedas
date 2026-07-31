package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// ──────────────────────────────────────────────
//  Configuração do Provedor de IA
// ──────────────────────────────────────────────

type AIProvider struct {
	Name       string
	BaseURL    string
	EnvKey     string
	DefaultModel string
}

var providers = map[string]AIProvider{
	"openai": {
		Name:         "OpenAI",
		BaseURL:      "https://api.openai.com/v1",
		EnvKey:       "OPENAI_API_KEY",
		DefaultModel: "gpt-4o-mini",
	},
	"gemini": {
		Name:         "Google Gemini",
		BaseURL:      "https://generativelanguage.googleapis.com/v1beta/openai/",
		EnvKey:       "GEMINI_API_KEY",
		DefaultModel: "gemini-2.0-flash",
	},
	"openrouter": {
		Name:         "OpenRouter",
		BaseURL:      "https://openrouter.ai/api/v1",
		EnvKey:       "OPENROUTER_API_KEY",
		DefaultModel: "inclusionai/ling-3.0-flash:free",
	},
}

func getAIProvider() AIProvider {
	name := strings.ToLower(os.Getenv("AI_PROVIDER"))
	if name == "" {
		name = "openrouter"
	}
	p, ok := providers[name]
	if !ok {
		p = providers["openrouter"]
	}
	return p
}

func getAIModel() string {
	if m := os.Getenv("AI_MODEL"); m != "" {
		return m
	}
	return getAIProvider().DefaultModel
}

func getAIApiKey() string {
	p := getAIProvider()
	return os.Getenv(p.EnvKey)
}

// ──────────────────────────────────────────────
//  Chamada à API de IA (OpenAI-compatible)
// ──────────────────────────────────────────────

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	MaxTokens int          `json:"max_tokens"`
	Temperature float64    `json:"temperature"`
}

type ChatChoice struct {
	Message ChatMessage `json:"message"`
}

type ChatResponse struct {
	Choices []ChatChoice `json:"choices"`
}

func callAI(messages []ChatMessage) (string, error) {
	provider := getAIProvider()
	apiKey := getAIApiKey()
	model := getAIModel()

	if apiKey == "" {
		return "", fmt.Errorf("chave da API de IA não configurada. Defina %s no .env", provider.EnvKey)
	}

	body := ChatRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   400,
		Temperature: 0.7,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("erro ao serializar request: %w", err)
	}

	url := provider.BaseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("erro ao criar request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// OpenRouter recomenda esses headers
	if provider.Name == "OpenRouter" {
		req.Header.Set("HTTP-Referer", "https://github.com/alvarossantos/api-moedas")
		req.Header.Set("X-Title", "GoCurrency")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro na chamada HTTP: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("erro ao ler resposta: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("AI API error (%d): %s", resp.StatusCode, string(respBody))
		return "", fmt.Errorf("erro da API de IA (HTTP %d)", resp.StatusCode)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("erro ao parsear resposta: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("resposta vazia da API de IA")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// ──────────────────────────────────────────────
//  Handler: POST /api/chat
// ──────────────────────────────────────────────

type ChatRequestBody struct {
	Message  string        `json:"message"`
	History  []ChatMessage `json:"history,omitempty"`
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody ChatRequestBody
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(reqBody.Message) == "" {
		http.Error(w, "Mensagem não pode estar vazia", http.StatusBadRequest)
		return
	}

	log.Printf("RECEIVED [Chat]: %s", reqBody.Message)

	// 1. Busca cotações atuais para dar contexto à IA
	ratesContext := getRatesContext()

	// 2. Monta os messages para a IA
	systemPrompt := fmt.Sprintf(`Você é o assistente virtual do GoCurrency, uma API de cotações de moedas.

SOBRE O SERVIÇO:
- API de conversão de moedas com cotações em tempo real
- Moedas suportadas: USD, EUR, BRL, GBP, JPY, CAD, AUD, ARS e 160+ outras
- Fonte dos dados: ExchangeRate-API

COTAÇÕES ATUAIS:
%s

SUAS FUNCIONALIDADES:
- Converter valores entre moedas
- Informar cotações atuais
- Explicar variações de câmbio
- Ajudar a entender qual moeda usar

REGRAS:
- Responda em português brasileiro
- Seja objetivo e direto
- Use os dados de cotação acima quando relevante
- Máximo de 3 frases por resposta
- NÃO use markdown, apenas texto simples
- Se o usuário pedir uma conversão, faça o cálculo com as cotações acima`, ratesContext)

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
	}

	// Adiciona histórico (últimas 10 mensagens)
	if len(reqBody.History) > 0 {
		start := 0
		if len(reqBody.History) > 10 {
			start = len(reqBody.History) - 10
		}
		for _, msg := range reqBody.History[start:] {
			if msg.Role == "user" || msg.Role == "assistant" {
				messages = append(messages, msg)
			}
		}
	}

	messages = append(messages, ChatMessage{Role: "user", Content: reqBody.Message})

	// 3. Chama a IA
	resposta, err := callAI(messages)
	if err != nil {
		log.Printf("ERROR [Chat]: %v", err)
		http.Error(w, fmt.Sprintf("Erro ao processar: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("SUCCESS [Chat]: resposta gerada (%d chars)", len(resposta))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"reply": resposta})
}

// ──────────────────────────────────────────────
//  Handler: GET /api/ai/status
// ──────────────────────────────────────────────

func aiStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	provider := getAIProvider()
	apiKey := getAIApiKey()
	model := getAIModel()

	status := map[string]interface{}{
		"provider":       provider.Name,
		"providerId":     strings.ToLower(provider.Name),
		"model":          model,
		"keyConfigured":  apiKey != "",
		"ready":          apiKey != "",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// ──────────────────────────────────────────────
//  Helper: Busca cotações atuais para contexto
// ──────────────────────────────────────────────

func getRatesContext() string {
	if apiKey == "" {
		return "(cotações indisponíveis — API key não configurada)"
	}

	base := "USD"
	url := fmt.Sprintf("https://v6.exchangerate-api.com/v6/%s/latest/%s", apiKey, base)

	resp, err := httpGet(url)
	if err != nil {
		return "(erro ao buscar cotações)"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "(cotações indisponíveis)"
	}

	var apiResponse struct {
		Rates map[string]float64 `json:"conversion_rates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return "(erro ao parsear cotações)"
	}

	if len(apiResponse.Rates) == 0 {
		return "(nenhuma cotação disponível)"
	}

	// Monta um resumo das principais moedas
	keyCurrencies := []string{"BRL", "EUR", "GBP", "JPY", "CAD", "AUD", "ARS"}
	lines := []string{fmt.Sprintf("Base: %s", base)}

	for _, code := range keyCurrencies {
		if rate, ok := apiResponse.Rates[code]; ok {
			lines = append(lines, fmt.Sprintf("- %s: %.4f", code, rate))
		}
	}

	return strings.Join(lines, "\n")
}
