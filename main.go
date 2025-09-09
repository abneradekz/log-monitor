package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/logging"
	"github.com/fsnotify/fsnotify"
)

// Constante que não muda
const (
	watchPath = "/logs"
)

// Estrutura do JSON
type LogEntryPayload struct {
	Severity    logging.Severity       `json:"severity"`
	Message     string                 `json:"message"`
	JsonPayload map[string]interface{} `json:"jsonPayload"`
	Labels      map[string]string      `json:"labels"`
}

func main() {
	// Lendo todas as configurações a partir de variáveis de ambiente
	gcpProjectID := os.Getenv("GCP_PROJECT_ID")
	if gcpProjectID == "" {
		log.Fatal("A variável de ambiente GCP_PROJECT_ID não foi definida.")
	}

	logID := os.Getenv("LOG_ID")
	if logID == "" {
		log.Fatal("A variável de ambiente LOG_ID não foi definida.")
	}

	ctx := context.Background()
	client, err := logging.NewClient(ctx, "projects/"+gcpProjectID)
	if err != nil {
		log.Fatalf("Falha ao criar cliente de logging: %v", err)
	}
	defer client.Close()

	logger := client.Logger(logID).StandardLogger(logging.Info)
	logger.Println("Serviço de Log Shipper iniciado.")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Falha ao criar watcher:", err)
	}
	defer watcher.Close()

	fileQueue := make(chan string, 100)
	go func() {
		for filePath := range fileQueue {
			processLogFile(filePath, client.Logger(logID))
		}
	}()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
					info, err := os.Stat(event.Name)
					if err == nil && !info.IsDir() {
						log.Println("Novo arquivo detectado:", event.Name)
						fileQueue <- event.Name
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Erro do watcher:", err)
			}
		}
	}()

	log.Println("Iniciando varredura de subdiretórios em:", watchPath)
	if err := filepath.Walk(watchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			log.Printf("Adicionando diretório ao watcher: %s", path)
			return watcher.Add(path)
		}
		return nil
	}); err != nil {
		log.Fatalf("Falha ao configurar o watcher recursivo: %v", err)
	}
	log.Println("Varredura e configuração do watcher concluídas.")

	<-make(chan struct{})
}

func processLogFile(filePath string, logger *logging.Logger) {
	time.Sleep(100 * time.Millisecond)

	content, err := os.ReadFile(filePath)
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

	logger.Log(entry)

	// --- ALTERAÇÃO AQUI ---
	// Simplificando a mensagem de log para remover a chamada logger.ID()
	log.Printf("Log de %s enviado com sucesso.", filePath)
	// --- FIM DA ALTERAÇÃO ---

	if err := os.Remove(filePath); err != nil {
		log.Printf("Erro ao remover o arquivo %s: %v", err)
	}
}
