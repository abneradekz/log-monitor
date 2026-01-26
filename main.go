package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync" // <-- ADICIONADO: para sincronizar o acesso ao mapa
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

// Mapa para rastrear arquivos que estão sendo processados e evitar duplicatas.
// A chave é o caminho do arquivo, o valor é um booleano.
var currentlyProcessing = make(map[string]bool)

// Mutex para garantir que o acesso ao mapa seja seguro entre as goroutines.
var processingMutex = &sync.Mutex{}

// ------------------------------------

func main() {
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
					// ---- ALTERAÇÃO AQUI: Lógica de Deduplicação ----
					processingMutex.Lock() // Bloqueia o acesso ao mapa
					if _, isProcessing := currentlyProcessing[event.Name]; isProcessing {
						// Se o arquivo já está na lista, ignora este evento.
						processingMutex.Unlock() // Libera o acesso
						continue
					}
					// Se não está na lista, adiciona e envia para a fila.
					currentlyProcessing[event.Name] = true
					processingMutex.Unlock() // Libera o acesso
					// ---------------------------------------------

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

	log.Println("Iniciando varredura de arquivos existentes em:", watchPath)
	if err := filepath.Walk(watchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 1. Se for diretório, adiciona ao watcher
		if info.IsDir() {
			log.Printf("Adicionando diretório ao watcher: %s", path)
			return watcher.Add(path)
		}

		// 2. Se for arquivo, processa imediatamente (backlog)
		// Usamos a mesma lógica de mutex para evitar conflito caso o watcher
		// dispare um evento ao mesmo tempo (raro para arquivos existentes, mas seguro).
		processingMutex.Lock()
		if _, isProcessing := currentlyProcessing[path]; isProcessing {
			processingMutex.Unlock()
			return nil
		}
		currentlyProcessing[path] = true
		processingMutex.Unlock()

		log.Println("Processando arquivo legado (existente):", path)
		fileQueue <- path

		return nil
	}); err != nil {
		log.Fatalf("Falha ao configurar o watcher recursivo: %v", err)
	}
	log.Println("Varredura inicial concluída. Aguardando novos arquivos...")

	<-make(chan struct{})
}

func processLogFile(filePath string, logger *logging.Logger) {
	// ---- ALTERAÇÃO AQUI: Remove o arquivo do mapa no final ----
	defer func() {
		processingMutex.Lock()
		delete(currentlyProcessing, filePath)
		processingMutex.Unlock()
	}()
	// -----------------------------------------------------------

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
	log.Printf("Log de %s enviado com sucesso.", filePath)

	if err := os.Remove(filePath); err != nil {
		log.Printf("Erro ao remover o arquivo %s: %v", err)
	}
}
