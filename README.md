
# API de Conversão de Moedas

Este é um projeto de API de conversão de moedas que permite aos usuários converter valores entre diferentes moedas e obter as taxas de câmbio mais recentes. A API é construída em Go e possui um frontend simples para interação.

## ✨ Funcionalidades

*   **Conversão de Moedas:** Converta qualquer valor de uma moeda para outra.
*   **Taxas de Câmbio:** Obtenha as taxas de câmbio mais recentes para uma moeda base.
*   **Cache:** Armazena em cache os resultados das solicitações para um desempenho mais rápido e para evitar o uso excessivo da API externa.
*   **🤖 Assistente IA (CurrencyBot):** Chatbot inteligente que responde dúvidas sobre cotações, conversões e moedas usando cotações em tempo real como contexto.
*   **Multi-Provedor de IA:** Suporte a OpenAI, Google Gemini e OpenRouter (modelos gratuitos) — troque com uma linha no `.env`.
*   **Frontend Simples:** Uma interface de usuário simples para interagir com a API.
*   **Deploy Fácil:** O projeto está configurado para ser facilmente implantado usando Docker e Fly.io.

## 🛠️ Tecnologias Utilizadas

*   **Backend:** Go (padrão `net/http`, zero dependências externas)
*   **Frontend:** HTML, CSS, JavaScript
*   **IA:** OpenAI SDK (compatível com OpenAI, Gemini e OpenRouter)
*   **API Externa:** [ExchangeRate-API](https://www.exchangerate-api.com/)
*   **Containerização:** Docker
*   **Hospedagem:** Fly.io

## 🚀 Como Usar

### Pré-requisitos

*   Go (versão 1.23 ou superior)
*   Docker (opcional, para execução em contêiner)
*   Uma chave de API da [ExchangeRate-API](https://www.exchangerate-api.com/)
*   Uma chave de API de IA (OpenAI, Gemini ou OpenRouter)

### Instalação

1.  Clone o repositório:

    ```bash
    git clone https://github.com/alvarossantos/api-moedas.git
    cd api-moedas
    ```

2.  Crie um arquivo `.env` na raiz do projeto com suas chaves de API:

    ```env
    API_KEY_EXCHANGE=sua-chave-exchange-rate

    # ── IA (configure um provedor) ──
    AI_PROVIDER=openrouter          # openai | gemini | openrouter
    OPENROUTER_API_KEY=sua-chave    # https://openrouter.ai/keys
    ```

    > **Modelos disponíveis:** Veja a tabela completa no `.env.example` do repositório.

### Executando Localmente

Para executar o projeto localmente, use o seguinte comando:

```bash
go run main.go
```

O servidor estará disponível em `http://localhost:8080`.

### Executando com Docker

Para executar o projeto com Docker, construa a imagem e execute o contêiner:

```bash
docker build -t api-moedas .
docker run -p 8080:8080 -v ./.env:/app/.env api-moedas
```

## 🔗 Endpoints da API

### `/convert`

Converte um valor de uma moeda para outra.

**Parâmetros:**

*   `from`: A moeda de origem (ex: `USD`).
*   `to`: A moeda de destino (ex: `BRL`).
*   `amount`: O valor a ser convertido.

**Exemplo:**

```
GET /convert?from=USD&to=BRL&amount=10
```

### `/rates`

Obtém as taxas de câmbio para uma moeda base.

**Parâmetros:**

*   `base`: A moeda base (ex: `USD`).

**Exemplo:**

```
GET /rates?base=USD
```

### `/api/chat`

Chatbot IA que responde dúvidas sobre moedas e cotações. Utiliza cotações em tempo real como contexto.

**Método:** `POST`

**Body (JSON):**

```json
{
  "message": "Quanto está o dólar em reais?",
  "history": []
}
```

**Parâmetros:**

*   `message`: A pergunta do usuário (obrigatório).
*   `history`: Array de mensagens anteriores para manter contexto (opcional, máx. 10).

**Exemplo de resposta:**

```json
{
  "reply": "O dólar (USD) está cotado a 5,0938 reais (BRL) neste momento."
}
```

### `/api/ai/status`

Retorna o provedor de IA configurado e se está pronto para uso.

**Exemplo de resposta:**

```json
{
  "provider": "OpenRouter",
  "model": "inclusionai/ling-3.0-flash:free",
  "ready": true
}
```

## 🤝 Contribuindo

Contribuições são bem-vindas! Sinta-se à vontade para abrir uma issue ou enviar um pull request.

## 📄 Licença

Este projeto está licenciado sob a Licença MIT.
