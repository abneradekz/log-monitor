#!/bin/bash

# Cores para output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Pega o primeiro argumento. Se vazio, usa "pm2-log-listener"
APP_NAME="${1:-pm2-log-listener}"

echo -e "${YELLOW}--- Iniciando Setup do ${APP_NAME} ---${NC}"

# 1. Validação de Pré-requisitos
if [ ! -f .env ]; then
    echo -e "${RED}ERRO: Arquivo .env não encontrado!${NC}"
    echo "Por favor, crie o arquivo .env com as variáveis LOG_OUTPUT_PATH e PM2_APPS_TO_MONITOR."
    exit 1
fi

# 2. Instalação de Dependências (se necessário)
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}Dependências não encontradas. Executando npm install...${NC}"
    npm install
    if [ $? -ne 0 ]; then
        echo -e "${RED}Falha ao instalar dependências.${NC}"
        exit 1
    fi
fi

# 3. Gerenciamento do Processo PM2
# Verifica se o processo já existe no PM2
pm2 describe $APP_NAME > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}O processo '$APP_NAME' já existe. Aplicando Reload para atualizar o .env...${NC}"
    # Usa reload --update-env para garantir que novas variáveis do .env sejam lidas
    pm2 reload $APP_NAME --update-env
else
    echo -e "${GREEN}Iniciando novo processo '$APP_NAME' no PM2...${NC}"
    pm2 start index.js --name "$APP_NAME"
fi

echo -e "${YELLOW}--- Concluído! ---${NC}"
echo -e "Para ver os logs deste listener, execute: ${GREEN}pm2 logs $APP_NAME${NC}"
