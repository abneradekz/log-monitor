# --- Build Stage ---
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copia e prepara os arquivos de dependência
COPY go.mod go.sum ./
RUN go mod download

# Copia o código fonte e compila
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /log-shipper

# --- Final Stage ---
#FROM alpine:latest
FROM scratch

# ---- A MÁGICA ACONTECE AQUI ----
# Copia o "bundle" de certificados de CA do estágio de build (que funciona)
# para a imagem final. O programa Go automaticamente encontrará e usará este arquivo.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
# --------------------------------

# Copia apenas o binário compilado
COPY --from=builder /log-shipper /log-shipper

# Define o diretório que será monitorado
VOLUME /logs

# Define o ponto de entrada do container
ENTRYPOINT ["/log-shipper"]
