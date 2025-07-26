package main

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

//var storageMu sync.RWMutex

// Загружает список клиентов из файла
func loadClientList(filename string) []ClientInfo {
	data, err := os.ReadFile(filename)
	if err != nil {
		return []ClientInfo{}
	}

	var list []ClientInfo
	err = json.Unmarshal(data, &list)
	if err != nil {
		log.Printf("Ошибка парсинга %s: %v\n", filename, err)
		return []ClientInfo{}
	}
	return list
}

// Сохраняет список клиентов в файл
func saveClientList(filename string, list []ClientInfo) error {
	data, _ := json.MarshalIndent(list, "", "  ")
	return os.WriteFile(filename, data, 0644)
}

// Обновляет или добавляет информацию о клиенте в оперативное хранилище — clientRegistry
func updateRegistry(clientID, ip, hostname string) {
	registryMu.Lock()
	defer registryMu.Unlock()

	clientRegistry[clientID] = ClientInfo{
		ID:             clientID,
		IP:             ip,
		Hostname:       hostname,
		LastActiveTime: time.Now(),
	}
}

// Удаляет клиента с указанным ID из списка ClientInfo
func removeClientFromList(clientID string, list []ClientInfo) []ClientInfo {
	var newList []ClientInfo
	for _, c := range list {
		if c.ID != clientID {
			newList = append(newList, c)
		}
	}
	return newList
}
