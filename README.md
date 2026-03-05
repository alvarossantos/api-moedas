
# API de Conversão de Moedas

Este é um projeto de API de conversão de moedas que permite aos usuários converter valores entre diferentes moedas e obter as taxas de câmbio mais recentes. A API é construída em Go e possui um frontend simples para interação.

## ✨ Funcionalidades

*   **Conversão de Moedas:** Converta qualquer valor de uma moeda para outra.
*   **Taxas de Câmbio:** Obtenha as taxas de câmbio mais recentes para uma moeda base.
*   **Cache:** Armazena em cache os resultados das solicitações para um desempenho mais rápido e para evitar o uso excessivo da API externa.
*   **Frontend Simples:** Uma interface de usuário simples para interagir com a API.
*   **Deploy Fácil:** O projeto está configurado para ser facilmente implantado usando Docker e Fly.io.

## 🛠️ Tecnologias Utilizadas

*   **Backend:** Go
*   **Frontend:** HTML, CSS, JavaScript
*   **API Externa:** [ExchangeRate-API](https://www.exchangerate-api.com/)
*   **Containerização:** Docker
*   **Hospedagem:** Fly.io

## 🚀 Como Usar

### Pré-requisitos

*   Go (versão 1.23 ou superior)
*   Docker (opcional, para execução em contêiner)
*   Uma chave de API da [ExchangeRate-API](https://www.exchangerate-api.com/)

### Instalação

1.  Clone o repositório:

    ```bash
    git clone https://github.com/alvarossantos/api-moedas.git
    cd api-moedas
    ```

2.  Crie um arquivo `.env` na raiz do projeto e adicione sua chave de API:

    ```
    API_KEY_EXCHANGE=sua-chave-de-api
    ```

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

## 🤝 Contribuindo

Contribuições são bem-vindas! Sinta-se à vontade para abrir uma issue ou enviar um pull request.

## 📄 Licença

Este projeto está licenciado sob a Licença MIT.
