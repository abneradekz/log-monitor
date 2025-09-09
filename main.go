package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"time"

	"cloud.google.com/go/logging"
	"github.com/fsnotify/fsnotify"
)

const (
	logID     = "application-logs" // Nome do log que aparecerá no Google Logging
	watchPath = "/logs"            // Pasta que será monitorada dentro do container
)

// Estrutura do JSON que esperamos ler dos arquivos
type LogEntryPayload struct {
	Severity    logging.Severity       `json:"severity"`
	Message     string                 `json:"message"`
	JsonPayload map[string]interface{} `json:"jsonPayload"`
	Labels      map[string]string      `json:"labels"`
}

func main() {
	// Lê o Project ID a partir da variável de ambiente
	gcpProjectID := os.Getenv("GCP_PROJECT_ID")
	if gcpProjectID == "" {
		log.Fatal("A variável de ambiente GCP_PROJECT_ID não foi definida.")
	}

	ctx := context.Background()

	// 1. Inicia o cliente do Google Logging
	// A biblioteca gerencia o batching e o envio assíncrono automaticamente.
	client, err := logging.NewClient(ctx, "projects/"+gcpProjectID)
	if err != nil {
		log.Fatalf("Falha ao criar cliente de logging: %v", err)
	}
	defer client.Close()

	logger := client.Logger(logID).StandardLogger(logging.Info)
	logger.Println("Serviço de Log Shipper iniciado. Observando a pasta:", watchPath)

	// 2. Cria o watcher para o sistema de arquivos
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Falha ao criar watcher:", err)
	}
	defer watcher.Close()

	// Canal para processar arquivos de forma concorrente
	fileQueue := make(chan string, 100) // Buffer de 100 arquivos

	// 3. Inicia um "worker" para processar a fila de arquivos
	go func() {
		for filePath := range fileQueue {
			processLogFile(filePath, client.Logger(logID))
		}
	}()

	// 4. Goroutine principal que assiste os eventos do watcher
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Nos interessa apenas quando um arquivo é criado ou escrito.
				if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("Novo arquivo detectado:", event.Name)
					fileQueue <- event.Name // Adiciona o arquivo na fila para processamento
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Erro do watcher:", err)
			}
		}
	}()

	// 5. Adiciona a pasta ao watcher
	err = watcher.Add(watchPath)
	if err != nil {
		log.Fatal("Falha ao adicionar pasta ao watcher:", err)
	}

	// Mantém o serviço rodando
	<-make(chan struct{})
}

func processLogFile(filePath string, logger *logging.Logger) {
	// Pequeno delay para garantir que a escrita no arquivo terminou
	time.Sleep(100 * time.Millisecond)

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Printf("Erro ao ler o arquivo %s: %v", filePath, err)
		return
	}

	var payload LogEntryPayload
	if err := json.Unmarshal(content, &payload); err != nil {
		log.Printf("Erro ao fazer parse do JSON de %s: %v", filePath, err)
		return
	}

	entry := logging.Entry{
		Payload:  payload.JsonPayload,
		Severity: payload.Severity,
		Labels:   payload.Labels,
	}

	// Envia o log. A biblioteca cuida do batching.
	logger.Log(entry)
	log.Printf("Log de %s enviado com sucesso.", filePath)

	// Opcional: remover o arquivo após o processamento
	if err := os.Remove(filePath); err != nil {
		log.Printf("Erro ao remover o arquivo %s: %v", filePath, err)
	}
}
