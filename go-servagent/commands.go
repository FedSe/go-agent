package main

import (
	"encoding/json"
	"fmt"
	"log"
)

// Отправка команды клиенту
func sendCommandToClient(clientID string, command string) error {
	connsMu.RLock()
	conn, ok := activeConns[clientID]
	connsMu.RUnlock()

	if !ok {
		return fmt.Errorf("клиент %s не найден", clientID)
	}

	if conn == nil {
		return fmt.Errorf("клиент %s не подключён", clientID)
	}

	msg := CommandMessage{
		Type:    "command",
		Command: command,
		Target:  clientID,
	}

	err := json.NewEncoder(conn).Encode(msg)
	if err != nil {
		log.Printf("Ошибка отправки команды клиенту %s: %v\n", clientID, err)

		// Сброс соединения при ошибке
		connsMu.Lock()
		delete(activeConns, clientID)
		connsMu.Unlock()

		return err
	}

	log.Printf("Команда отправлена: %s -> %s", clientID, command)
	return nil
}
