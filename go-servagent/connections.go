package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Управляет жизненным циклом клиентского подключения
func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Аутентификация
	clientID, ip, hostname, isKnown, err := authenticateClient(conn)
	if err != nil {
		log.Printf("подключение от %v — отклонено. \n%v", conn.RemoteAddr(), err)
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\nContent-Length: 18\r\n\r\nUse proper client!"))
		return
	}

	fmt.Println("Обнаружен клиент:", clientID, ip, hostname, isKnown)

	if !isKnown {
		fmt.Println("Клиент отправлен на регистрацию как неизвестный", clientID)
		registerPendingClient(conn, clientID, ip, hostname)
		return
	}

	// Клиент в known = одобрен
	json.NewEncoder(conn).Encode(struct{ Type string }{Type: "approved"})

	// Регистрация подключения
	err = registerApprovedClient(conn, clientID, ip, hostname)
	if err != nil {
		log.Println("Ошибка регистрации клиента:", err)
		return
	}

	fmt.Println("Готовность, клиент: ", clientID)

	// Ждём команды
	processClientResponses(conn, clientID)
}

// Аутентифицирует клиента
func authenticateClient(conn net.Conn) (string, string, string, bool, error) {
	var authMsg struct {
		Type     string `json:"type"`
		ID       string `json:"client_id"`
		Hostname string `json:"hostname,omitempty"`
	}

	err := json.NewDecoder(conn).Decode(&authMsg)
	if err != nil {
		return "", "", "", false, fmt.Errorf("ошибка авторизации: %w", err)
	}

	if authMsg.Type != "auth" {
		return "", "", "", false, fmt.Errorf("неподдерживаемый тип сообщения: %s", authMsg.Type)
	}

	remoteAddr := conn.RemoteAddr().String()
	ip := strings.Split(remoteAddr, ":")[0]

	knownList := loadClientList("clients_known.json")
	isKnown := false
	for _, c := range knownList {
		if c.ID == authMsg.ID {
			isKnown = true
			break
		}
	}

	return authMsg.ID, ip, authMsg.Hostname, isKnown, nil
}

// Регистрирует нового (неизвестного) клиента как ожидающего одобрения
func registerPendingClient(conn net.Conn, clientID, ip, hostname string) error {
	pendingList := loadClientList("clients_pending.json")

	exists := false
	for _, c := range pendingList {
		if c.ID == clientID {
			exists = true
			break
		}
	}

	if !exists {
		pendingList = append(pendingList, ClientInfo{
			ID:             clientID,
			IP:             ip,
			Hostname:       hostname,
			LastActiveTime: time.Now(),
		})
		saveClientList("clients_pending.json", pendingList)
	}

	updateRegistry(clientID, ip, hostname)

	json.NewEncoder(conn).Encode(struct{ Type string }{Type: "pending"})
	return nil
}

// Регистрирует известного клиента, который уже был одобрен
func registerApprovedClient(conn net.Conn, clientID, ip, hostname string) error {
	updateRegistry(clientID, ip, hostname)

	knownList := loadClientList("clients_known.json")
	found := false
	for i, c := range knownList {
		if c.ID == clientID {
			knownList[i].IP = ip
			knownList[i].Hostname = hostname
			knownList[i].LastActiveTime = time.Now()
			found = true
			break
		}
	}

	if !found {
		knownList = append(knownList, ClientInfo{
			ID:             clientID,
			IP:             ip,
			Hostname:       hostname,
			LastActiveTime: time.Now(),
		})
	}

	saveClientList("clients_known.json", knownList)

	connsMu.Lock()
	activeConns[clientID] = conn
	connsMu.Unlock()

	log.Printf("Клиент зарегистрирован: %s | IP: %s | HN: %s", clientID, ip, hostname)
	return nil
}

// Бесконечный цикл, который ожидает и обрабатывает ответы от клиента (результаты выполнения команд)
func processClientResponses(conn net.Conn, clientID string) {
	var hostname string
	registryMu.RLock()
	if c, ok := clientRegistry[clientID]; ok {
		hostname = c.Hostname
	}
	registryMu.RUnlock()

	for {
		var response ResponseMessage
		err := json.NewDecoder(conn).Decode(&response)
		if err != nil {
			if strings.Contains(err.Error(), "forcibly closed") || strings.Contains(err.Error(), "connection reset") {
				log.Printf("Клиент Отключился: %s ", clientID)
			} else {
				log.Println("Ошибка декодирования ответа:", err)
			}
			// Отключился = убрать Conn
			connsMu.Lock()
			delete(activeConns, clientID)
			connsMu.Unlock()
			break
		}

		registryMu.Lock()
		client := clientRegistry[clientID]
		client.LastActiveTime = time.Now()
		clientRegistry[clientID] = client
		registryMu.Unlock()

		knownList := loadClientList("clients_known.json")
		for i, c := range knownList {
			if c.ID == clientID {
				knownList[i].IP = client.IP
				knownList[i].LastActiveTime = client.LastActiveTime
				break
			}
		}
		saveClientList("clients_known.json", knownList)

		// Лог результата
		var parts []string
		for k, v := range response.Data {
			parts = append(parts, fmt.Sprintf("%s: %s", k, v))
		}
		fmt.Println(parts)

		logEntry := ClientLogEntry{
			Time:    time.Now(),
			Command: response.Command,
			Output:  response.Data,
		}
		addClientLog(clientID, logEntry)

		logEntryStr := fmt.Sprintf("[%s | %s | %s] (%s) -> %s",
			time.Now().Format("2006-01-02 15:04:05"),
			clientID,
			response.Command,
			hostname,
			response.Data,
		)

		logsMu.Lock()
		logs = append([]string{logEntryStr}, logs...)
		logsMu.Unlock()
	}
}

// Для чтения логов
func loadClientLog(clientID string) []ClientLogEntry {
	logFilePath := filepath.Join("logs", clientID+".json")
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		return []ClientLogEntry{}
	}

	data, err := os.ReadFile(logFilePath)
	if err != nil {
		log.Println("Ошибка чтения лога клиента:", err)
		return []ClientLogEntry{}
	}

	var entries []ClientLogEntry
	err = json.Unmarshal(data, &entries)
	if err != nil {
		log.Println("Ошибка парсинга лога клиента:", err)
		return []ClientLogEntry{}
	}

	return entries
}

// Логирование
func addClientLog(clientID string, entry ClientLogEntry) {
	logFilePath := filepath.Join("logs", clientID+".json")

	logs := loadClientLog(clientID)
	logs = append(logs, entry)

	// Размер лога - последние 100 записей
	if len(logs) > 100 {
		logs = logs[len(logs)-100:]
	}

	data, _ := json.MarshalIndent(logs, "", "  ")
	os.WriteFile(logFilePath, data, 0644)
}
