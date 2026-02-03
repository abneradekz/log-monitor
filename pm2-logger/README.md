# **PM2 Logger Config Generator**

Este repositório contém um utilitário para gerar arquivos de configuração do rsyslog para monitoramento de processos PM2, facilitando a integração com o **Google Cloud Ops Agent** e outros sistemas de monitoramento via Syslog.

## **🛠 Script de Geração de Configuração**

O script `create_logger_config.sh` facilita a criação de arquivos de configuração baseados em um modelo padrão (`pm2-logger.model`).

### **✅ Funcionalidades**

* Cria automaticamente a estrutura de diretórios.  
* Gera arquivos de configuração `.conf` personalizados.  
* Suporta execução em lote (múltiplos processos).  
* Modo interativo inteligente para inserção de múltiplos processos.  
* Utiliza um template (`pm2-logger.model`) para padronização.

### **🚀 Como Usar o Gerador**

#### **1. Modo em Lote (Batch)**

Execute o script passando o nome da pasta seguido pelos nomes dos processos:
```bash
./create_logger_config.sh <nome_da_pasta> [processo1] [processo2] ...
```

**Exemplo:**  
```bash
./create_logger_config.sh api-logs backend-api worker-jobs notification-service
```

Isso criará a pasta `api-logs` e três arquivos de configuração dentro dela, um para cada processo.

#### **2. Modo Interativo**

Se você executar o script apenas com o nome da pasta (ou sem argumentos), ele entrará em um loop solicitando os nomes:
```bash
./create_logger_config.sh minhapasta
```

O script pedirá:

1. `Digite o nome da pasta:` (se não fornecido)  
2. `Digite o nome do processo:` (pressione Enter após cada nome)  
3. Para finalizar e gerar os arquivos, pressione **Enter** sem digitar nada.

## **📦 Instalação e Ativação**

Após gerar os arquivos `.conf`, siga os passos abaixo para ativar o envio dos logs para o Syslog/Google Cloud.

### **1. Mover os arquivos de configuração**

Mova os arquivos gerados (`.conf`) para o diretório de configuração do Rsyslog:  
```bash
# Exemplo: copiando da pasta gerada 'api-logs'  
sudo cp api-logs/*.conf /etc/rsyslog.d/
```

### **2. Ajustar Permissões (Crítico)**

O Rsyslog roda com um usuário restrito (`syslog`) e, por padrão, não consegue ler logs dentro da pasta `/root/.pm2/`. É necessário liberar a leitura:  
```bash
# Libera acesso de execução aos diretórios pais  
sudo chmod 711 /root  
sudo chmod 711 /root/.pm2

# Libera leitura na pasta de logs e nos arquivos  
sudo chmod 755 /root/.pm2/logs  
sudo chmod 644 /root/.pm2/logs/*.log
```

### **3. Reiniciar o Rsyslog**

Aplique as alterações reiniciando o serviço:  
```bash
sudo systemctl restart rsyslog
```

## **🔍 Verificação**

### **Verificar Localmente**

Para confirmar se o Rsyslog está capturando os dados, monitore o syslog filtrando pela tag do seu processo (ex: `pm2-backend-api`):
```bash
tail -f /var/log/syslog | grep "pm2-backend-api"
```

*Gere algum log na sua aplicação. Se aparecer aqui, a configuração está correta.*

### **Verificar no Google Cloud**

Se o **Google Cloud Ops Agent** estiver instalado, os logs aparecerão automaticamente no **Logs Explorer**.

1. Vá ao Google Cloud Console > Logging.  
2. Filtre pela tag configurada:
    ```bash
    jsonPayload.syslog_tag="pm2-backend-api"
    ```

### **📁 Estrutura de Arquivos do Projeto**

* **create_logger_config.sh**: O script gerador.  
* **pm2-logger.model**: O template base (placeholders são substituídos pelo nome do processo).  
* **.gitignore**: Ignora arquivos `.conf` gerados para evitar commits acidentais.

# 
# 
# Quando não tiver chegando os logs
## **Verificação e Instalação do Google Cloud Ops Agent**

Este procedimento descreve como verificar se o Agente de Operações do Google Cloud (Ops Agent) está instalado e como instalá-lo caso não esteja.

### **1. Verificar Status do Agente**

Antes de instalar, verifique se o agente já está em execução no servidor.  
**Para Linux:**  
Execute o seguinte comando para verificar o status do serviço:  
```bash
sudo systemctl status google-cloud-ops-agent
```

* **Se estiver instalado:** Você verá um status `active (running)`.  
* **Se não estiver instalado:** O comando retornará um erro `Unit google-cloud-ops-agent.service could not be found` ou similar.

### **2. Instalar o Agente (Caso não tenha)**

Se o agente não estiver instalado, utilize o script oficial de instalação do Google. Este script adiciona o repositório do agente e realiza a instalação.  
**Passo a passo para Linux (Debian, Ubuntu, CentOS, RHEL):**

1. Baixe o script de instalação do repositório:
    ```bash
    curl -sSO [https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh](https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh)
    ```

2. Execute o script para adicionar o repositório e instalar o agente:  
    ```bash
    sudo bash add-google-cloud-ops-agent-repo.sh --also-install
    ```

3. (Opcional) Verifique se a instalação foi bem-sucedida verificando o status novamente:  
    ```bash
    sudo systemctl status google-cloud-ops-agent"*"
    ```

### **3. Configuração Padrão**

Após a instalação, o agente começa a coletar automaticamente:

* Métricas de host (CPU, Memória, Disco, Rede).  
* Logs do sistema (`syslog` no Linux ou Event Log no Windows).

O arquivo de configuração principal fica localizado em:

* **Linux:** `/etc/google-cloud-ops-agent/config.yaml`

Se precisar reiniciar o agente após uma mudança de configuração:  
sudo systemctl restart google-cloud-ops-agent  
```bash
sudo systemctl restart google-cloud-ops-agent
```