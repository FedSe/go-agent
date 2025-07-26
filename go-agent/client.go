package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type CommandMessage struct {
	Type    string `json:"type"`    // "auth", "command"
	Command string `json:"command"` // "get HN", "list VC"
	Target  string `json:"target"`  // clientID или "all"
}

func getClientID() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("не удалось получить текущую директорию: %v", err)
	}

	idPath := filepath.Join(dir, idFile)

	if _, err := os.Stat(idPath); err == nil {
		file, err := os.Open(idPath)
		if err != nil {
			return "", fmt.Errorf("не удалось открыть файл: %v", err)
		}
		defer file.Close()

		var data struct {
			ID string `json:"client_id"`
		}
		err = json.NewDecoder(file).Decode(&data)
		if err != nil {
			return "", fmt.Errorf("ошибка парсинга client_id: %v", err)
		}

		log.Printf("Клиент загружен с ID: %s", data.ID)
		return data.ID, nil

	} else if os.IsNotExist(err) {
		newID := fmt.Sprintf("agent-%s-%d", runtime.GOARCH, time.Now().UnixNano())
		log.Printf("Создан новый ID: %s", newID)

		file, err := os.Create(idPath)
		if err != nil {
			return "", fmt.Errorf("не удалось создать файл: %v", err)
		}
		defer file.Close()

		data := struct {
			ID string `json:"client_id"`
		}{ID: newID}

		err = json.NewEncoder(file).Encode(data)
		if err != nil {
			return "", fmt.Errorf("не удалось записать ID в файл: %v", err)
		}

		return newID, nil
	}

	return "", fmt.Errorf("не удалось проверить файл с ID: %v", err)
}

func startClient(clientID string) {
	for {
		conn, err := net.Dial("tcp", serverAddr)
		if err != nil {
			log.Println("Ошибка подключения:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		defer conn.Close()

		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}

		// Регистрация
		authMsg := struct {
			Type     string `json:"type"`
			ID       string `json:"client_id"`
			Hostname string `json:"hostname"`
		}{
			Type:     "auth",
			ID:       clientID,
			Hostname: hostname,
		}
		json.NewEncoder(conn).Encode(authMsg)

		// Ожидание ответа
		var authResponse struct {
			Type string `json:"type"`
		}
		err = json.NewDecoder(conn).Decode(&authResponse)
		if err != nil {
			log.Println("Ошибка приёма ответа от сервера:", err)
			conn.Close()
			continue
		}

		if authResponse.Type == "pending" {
			fmt.Println("Ожидаем одобрения на сервере...")
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		} else if authResponse.Type != "approved" {
			log.Println("Неизвестный тип ответа от сервера:", authResponse.Type)
			conn.Close()
			continue
		}

		fmt.Println("Клиент одобрен. Слушаем команды...")

		for {
			var msg CommandMessage
			err := json.NewDecoder(conn).Decode(&msg)
			if err != nil {
				log.Println("Ошибка декодирования команды:", err)
				break
			}

			log.Printf("Получена команда: %+v\n", msg)

			if msg.Target == "all" || msg.Target == clientID {
				response := handleCommand(msg)
				log.Printf("Отправляем ответ:\n%+v\n", response)
				err := json.NewEncoder(conn).Encode(response)
				if err != nil {
					log.Println("Ошибка отправки ответа:", err)
				}
			}
		}
	}
}
