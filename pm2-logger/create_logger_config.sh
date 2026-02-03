#!/bin/bash

# Script para gerar arquivos de configuração do pm2-logger
# Uso: ./create_logger_config.sh <nome_da_pasta> [processo1 processo2 ...]

# Variáveis
FOLDER_NAME="$1"
PROCESS_LIST=("${@:2}")

# 1. Validação/Input da Pasta
if [ -z "$FOLDER_NAME" ]; then
    read -p "Digite o nome da pasta: " FOLDER_NAME
fi

if [ -z "$FOLDER_NAME" ]; then
    echo "Erro: Nome da pasta é obrigatório."
    exit 1
fi

# 2. Validação/Input dos Processos
if [ ${#PROCESS_LIST[@]} -eq 0 ]; then
    echo "Nenhum nome de processo fornecido. Iniciando modo interativo (Enter vazio p/ finalizar)."
    while true; do
        read -p "Digite o nome do processo: " INPUT_NAME
        
        # Remove espaços em branco
        INPUT_NAME=$(echo "$INPUT_NAME" | xargs)
        
        if [ -z "$INPUT_NAME" ]; then
            if [ ${#PROCESS_LIST[@]} -eq 0 ]; then
                echo "Erro: Pelo menos um nome de processo deve ser informado."
                exit 1
            else
                break
            fi
        else
            PROCESS_LIST+=("$INPUT_NAME")
        fi
    done
fi

# Caminhos Base
BASE_DIR=$(dirname "$0")
TEMPLATE_FILE="$BASE_DIR/pm2-logger.model"
DEST_DIR="$BASE_DIR/$FOLDER_NAME"

# Verifica se o template existe
if [ ! -f "$TEMPLATE_FILE" ]; then
    echo "Erro: Arquivo de template '$TEMPLATE_FILE' não encontrado."
    exit 1
fi

# Cria a pasta de destino se não existir
if [ ! -d "$DEST_DIR" ]; then
    echo "Criando diretório: $DEST_DIR"
    mkdir -p "$DEST_DIR"
fi

# 3. Geração dos Arquivos
echo "--- Iniciando geração para ${#PROCESS_LIST[@]} processos ---"

for PROCESS_NAME in "${PROCESS_LIST[@]}"; do
    DEST_FILE="$DEST_DIR/90-pm2-logger-${PROCESS_NAME}.conf"

    # Informa se está criando ou atualizando
    if [ -f "$DEST_FILE" ]; then
        echo "[$PROCESS_NAME] Atualizando arquivo existente: $DEST_FILE"
    else
        echo "[$PROCESS_NAME] Criando novo arquivo: $DEST_FILE"
    fi

    # Gera o arquivo substituindo os valores
    # modelo -> PROCESS_NAME
    sed -e "s/modelo/${PROCESS_NAME}/g" \
        "$TEMPLATE_FILE" > "$DEST_FILE"
done

echo "--- Concluído! ---"
