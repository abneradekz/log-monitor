# --- Build Stage ---
# Usando a versão estável mais recente do Go para segurança e performance.
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 1. Copia primeiro o arquivo de módulo.
COPY go.mod ./
COPY go.sum ./

# 2. Copia o código fonte.
COPY *.go ./

# 3. Roda o 'tidy' para sincronizar o go.mod com o código fonte.
# RUN go mod tidy

# 4. Baixa as dependências.
RUN go mod download

# 5. Compila a aplicação, agora com todas as dependências disponíveis.
RUN CGO_ENABLED=0 GOOS=linux go build -o /log-shipper

# --- Final Stage ---
FROM scratch

# Copia apenas o binário compilado
COPY --from=builder /log-shipper /log-shipper

# Define o diretório que será monitorado
VOLUME /logs

# Define o ponto de entrada do container
ENTRYPOINT ["/log-shipper"]
