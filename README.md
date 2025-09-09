# Serviço de Coleta de Logs (Log Shipper) para o Google Cloud Logging

Este projeto contém um microserviço de alta performance escrito em Go, projetado para monitorar diretórios no sistema de arquivos, ler arquivos de log em formato JSON e enviá-los de forma eficiente e em lote (batch) para o Google Cloud Logging.

O objetivo principal é desacoplar a escrita de logs das aplicações principais, garantindo que elas não sejam bloqueadas ou sofram com a latência de rede ao enviar logs para um serviço centralizado.

## Arquitetura e Funcionamento

O serviço opera de forma assíncrona e concorrente para garantir máxima eficiência:

1.  **Monitoramento (File Watcher):** Uma `goroutine` utiliza a biblioteca `fsnotify` para monitorar um diretório raiz (`/logs` dentro do contêiner) de forma contínua e não-bloqueante.
2.  **Fila de Processamento (Queue):** Ao detectar a criação de um novo arquivo, o caminho desse arquivo é enviado para um `channel` (canal) em Go, que funciona como uma fila em memória. Isso desacopla a detecção do processamento.
3.  **Processamento Concorrente (Worker):** Uma ou mais `goroutines` (workers) escutam esse canal. Assim que um caminho de arquivo é recebido, o worker:
    a. Lê o conteúdo do arquivo.
    b. Faz o parse do conteúdo JSON para uma estrutura de Log do Google.
    c. Envia a entrada de log para a biblioteca cliente do Google Cloud.
4.  **Envio em Lote (Batching):** A biblioteca cliente do Google Cloud Logging é responsável por agrupar múltiplas entradas de log em um único lote (batch) e enviá-las para a API do Google. Isso reduz drasticamente o número de requisições de rede, otimizando custos e performance.
5.  **Limpeza:** Após o envio bem-sucedido, o arquivo de log original é removido para evitar processamento duplicado.

## Tech Stack

* **Go (Golang):** Escolhido por sua excelência em concorrência (goroutines e channels), baixo consumo de recursos e compilação para um binário estático único, o que resulta em imagens Docker extremamente leves e seguras.
* **Docker & Docker Compose:** Para garantir um ambiente de desenvolvimento e produção padronizado, portátil e fácil de configurar.

## Pré-requisitos

1.  **Docker e Docker Compose:** [Instruções de Instalação](https://docs.docker.com/get-docker/).
2.  **Conta no Google Cloud Platform (GCP):** Com um projeto criado.
3.  **Service Account (Conta de Serviço):** É necessário criar uma Service Account no seu projeto GCP com a role (papel) de **"Gravador de Registros de Acesso" (`roles/logging.logWriter`)**.
    * Após criar a conta, gere uma chave do tipo JSON e faça o download.

## Configuração

1.  **Clone o repositório:**
    ```bash
    git clone <url-do-seu-repositorio>
    cd <nome-do-repositorio>
    ```

2.  **Credenciais do GCP:**
    Renomeie o arquivo de chave JSON da sua Service Account para `gcp-credentials.json` e coloque-o na raiz do projeto.

3.  **Variáveis de Ambiente:**
    Este projeto usa um arquivo `.env` para configuração. Crie o seu copiando o arquivo de exemplo:
    ```bash
    cp .env.example .env
    ```
    Agora, edite o arquivo `.env` e preencha o valor da variável `GCP_PROJECT_ID` com o ID do seu projeto no Google Cloud.

    **IMPORTANTE:** O arquivo `.env` contém configurações do seu ambiente e não deve ser versionado. Certifique-se de que ele está listado no seu arquivo `.gitignore`.

4.  **Pastas de Log:**
    Crie os diretórios locais que você deseja monitorar (ex: `mkdir logs-app-1`). Eles devem ser mapeados no arquivo `docker-compose.yml`.

## Como Executar

Com o Docker em execução, suba o serviço com o Docker Compose:

```bash
# O comando --build garante que a imagem será construída na primeira vez
# O -d (detached) faz com que o container rode em segundo plano
docker-compose up --build -d
```

Para ver os logs do próprio serviço de coleta, você pode usar:
```bash
docker-compose logs -f
```

## Como Usar

Com o serviço rodando, basta que sua aplicação (ou qualquer processo) crie um arquivo com extensão `.json` em uma das pastas que você mapeou no `docker-compose.yml` (ex: `logs-app-1`).

O serviço irá detectar o novo arquivo automaticamente, processá-lo e enviá-lo para o Google Cloud Logging.

### Estrutura do JSON de Log

Para aproveitar ao máximo os recursos de filtragem e visualização do Google Cloud Logging, os arquivos JSON devem seguir a estrutura do objeto `LogEntry`.

**Exemplo de arquivo `log-exemplo.json`:**
```json
{
  "severity": "ERROR",
  "message": "Falha ao processar o pagamento do usuário.",
  "timestamp": "2025-09-08T14:30:00.123456789Z",
  "labels": {
    "application": "backend-pagamentos",
    "module": "checkout",
    "version": "1.2.5"
  },
  "jsonPayload": {
    "userId": "user-12345",
    "orderId": "order-abc-987",
    "paymentMethod": "credit_card",
    "errorDetails": {
      "code": 5003,
      "reason": "Insufficient funds"
    }
  },
  "trace": "projects/seu-projeto/traces/a1b2c3d4e5f6a1b2c3d4e5f6",
  "httpRequest": {
    "requestMethod": "POST",
    "requestUrl": "/api/v1/payments",
    "status": 500,
    "userAgent": "MobileApp/1.0",
    "remoteIp": "192.0.2.1"
  }
}
```

#### Detalhes dos Campos Principais:

* `severity` (Opcional): A severidade do log. Valores comuns: `DEFAULT`, `DEBUG`, `INFO`, `NOTICE`, `WARNING`, `ERROR`, `CRITICAL`, `ALERT`, `EMERGENCY`. Isso transforma seu log em uma entrada colorida e filtrável na UI do Google Logging.
* `message` (Opcional): Uma mensagem de texto simples. Se você usar `jsonPayload`, este campo se torna menos importante, pois o Google Logging dá preferência ao conteúdo estruturado.
* `jsonPayload` (Recomendado): **O campo mais importante para logs estruturados.** O conteúdo deste objeto JSON é totalmente indexado e pesquisável. Você pode expandir os campos na interface do Logging e criar filtros complexos como `jsonPayload.userId="user-12345"`.
* `timestamp` (Opcional): O carimbo de tempo exato de quando o evento ocorreu, no formato RFC3339. Se for omitido, o Google atribui o horário em que recebeu o log, o que pode ser impreciso.
* `labels` (Opcional): Um conjunto de pares chave-valor para indexação de alta performance. Use para metadados que identificam a origem do log, como a aplicação, o ambiente (dev/prod), a versão, etc. São ótimos para filtros rápidos.
* `trace` (Opcional): Se você utiliza o Google Cloud Trace para rastreamento distribuído, este campo associa a entrada de log a um *trace* específico. Isso permite ver todos os logs de uma única requisição que passou por múltiplos microsserviços.
* `httpRequest` (Opcional): Se o log está relacionado a uma requisição HTTP, preencher este campo com detalhes da requisição (método, URL, status, IP de origem) faz com que o Google Logging agrupe os logs por requisição, facilitando a análise.

### Acessando logs no Google Cloud Logging

1. Acesse o [Google Cloud Console](https://console.cloud.google.com/).
2. Navegue até **Logging** > **Logs Explorer**.
3. Use o nome do log que você configurou (padrão: `application-log`).
   1. exemplo de filtro básico:
      ```
      logName="projects/[seu-projeto]/logs/[LOG_ID]"
      ```
   2. trocar `[seu-projeto]` pelo ID do seu projeto e `[LOG_ID]` pelo nome do log configurado no .env (padrão: `application-log`).
4. Você pode usar consultas avançadas para filtrar logs com base em `severity`, `labels`, `jsonPayload`, etc.
