# Etapa 1: Construção (Build)
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copia os arquivos de dependência
COPY go.mod ./
# Se tiver go.sum, descomente a linha abaixo
# COPY go.sum ./

# Copia todo o código fonte
COPY . .

# Compila o executável chamado "server"
RUN go build -o server main.go

# Etapa 2: Imagem Final (Pequena e leve)
FROM alpine:latest

WORKDIR /root/

# Instala certificados de segurança (necessário para chamar a API externa)
RUN apk --no-cache add ca-certificates

# Copia o executável da etapa anterior
COPY --from=builder /app/server .

# --- O PULO DO GATO ---
# Copia a pasta frontend para dentro do container final
COPY --from=builder /app/frontend ./frontend
# ----------------------

# Define a porta padrão (o Fly define isso, mas é bom ter fallback)
ENV PORT=8080

# Comando para rodar
CMD ["./server"]