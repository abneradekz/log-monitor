# **PM2 Log Listener & JSON Dumper**

Este serviço atua como um *middleware* de observabilidade local. Ele se conecta ao **PM2 Interactor Bus** em memória, intercepta logs (stdout e stderr) de aplicações específicas e os persiste como arquivos JSON estruturados.  
O objetivo principal é servir como **produtor** para agentes de log (como Log Shippers em Go) que monitoram diretórios via fsnotify, desacoplando a coleta de logs do envio para a nuvem.

## **🚀 Funcionalidades**

* **Interceptação em Tempo Real:** Escuta o barramento do PM2 sem necessidade de ler arquivos de log brutos do disco.
* **Whitelisting de Processos:** Monitora apenas os aplicativos definidos explicitamente no .env.
* **Estrutura GCP-Ready:** Gera JSONs já formatados com severity, jsonPayload e labels compatíveis com o Google Cloud Logging.
* **Atomicidade:** Cada linha de log gera um arquivo único (com UUID) para garantir que o *watcher* do consumidor detecte o evento Create sem conflitos de *lock*.
* **Fail Fast:** O serviço se recusa a iniciar se não houver aplicações configuradas para monitoramento.

## **📋 Pré-requisitos**

* Node.js (v14 ou superior)
* PM2 instalado globalmente (npm install \-g pm2)
* Ambiente Linux/WSL (recomendado)

## **⚙️ Instalação e Configuração**

### **1\. Clone o repositório e instale as dependências**

npm install

### **2\. Configure as variáveis de ambiente**

Crie um arquivo .env na raiz do projeto com o seguinte conteúdo:  
\# Diretório onde os JSONs serão salvos (o Agente Go deve escutar esta pasta)  
LOG\_OUTPUT\_PATH=./pm2\_logs\_queue

\# Lista de processos do PM2 para monitorar (nomes exatos, separados por vírgula)  
\# Exemplo: backend-api, nextjs-front, worker-jobs  
PM2\_APPS\_TO\_MONITOR=nome-do-app-1,nome-do-app-2

\# Metadado de ambiente (opcional)  
NODE\_ENV=production

**Nota Crítica:** Se PM2\_APPS\_TO\_MONITOR estiver vazio ou não definido, o serviço encerrará imediatamente com erro (Exit Code 1\) para evitar execução ociosa.

## **▶️ Como Executar**

Utilize o script start.sh incluído para gerenciar o ciclo de vida do processo no PM2. Ele cuida automaticamente da instalação de dependências, validação do arquivo .env e aplica um *reload* inteligente se o processo já estiver rodando.

### **Opção 1: Iniciar com nome padrão**

O nome padrão do processo no PM2 será pm2-log-listener.  
./start.sh

### **Opção 2: Iniciar com nome personalizado**

Isso é útil se você precisa rodar múltiplas instâncias desse script na mesma máquina, monitorando grupos de aplicações diferentes.  
./start.sh meu-monitor-customizado

## **📦 Formato do Output**

Os arquivos gerados na pasta LOG\_OUTPUT\_PATH seguem o padrão de nomenclatura TIMESTAMP-UUID.json.  
**Exemplo de conteúdo gerado:**  
{  
"severity": "INFO",  
"message": "Usuário 123 logou com sucesso",  
"jsonPayload": {  
"process\_name": "backend-api",  
"pm\_id": 4,  
"timestamp": "2025-10-25T14:30:00.000Z"  
},  
"labels": {  
"source": "pm2-listener",  
"environment": "production"  
}  
}

## **🛠 Integração com Agente Go (Log Shipper)**

Se você estiver utilizando um agente externo (ex: em Go) para enviar esses logs para o Google Cloud Platform (GCP):

1. **Caminho:** Aponte o agente Go para monitorar a mesma pasta definida em LOG\_OUTPUT\_PATH.
2. **Estrutura:** O JSON gerado casa perfeitamente com a struct LogEntryPayload (Severity, Message, JsonPayload).
3. **Eventos:** O uso de arquivos únicos com UUID garante que o evento fsnotify.Create seja disparado corretamente no Go para cada linha de log, evitando condições de corrida (race conditions).

## **🐛 Troubleshooting**

**O serviço não inicia:**

* Verifique se o arquivo .env foi criado.
* Verifique se PM2\_APPS\_TO\_MONITOR possui pelo menos um nome de app válido.

**Logs não aparecem na pasta de saída:**

* Verifique se os nomes no .env batem *exatamente* (case-sensitive) com os nomes listados no comando pm2 list.
* Verifique se o usuário que roda o script tem permissões de escrita na pasta de destino.
* Consulte os logs do próprio listener para ver erros de execução:  
  pm2 logs pm2-log-listener  
