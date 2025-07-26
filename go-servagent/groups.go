package main

import (
	"encoding/json"
	"log"
	"os"
	"sort"
	"sync"
)

var (
	groups   = make(map[string]ClientGroup)
	groupsMu sync.RWMutex
)

// Загрузка групп из файла
func loadGroups() []ClientGroup {
	data, err := os.ReadFile("client_groups.json")
	if err != nil {
		return []ClientGroup{}
	}

	var groupsList []ClientGroup
	err = json.Unmarshal(data, &groupsList)
	if err != nil {
		log.Println("Ошибка парсинга групп:", err)
		return []ClientGroup{}
	}
	return groupsList
}

func saveGroups(groups []ClientGroup) error {
	data, _ := json.MarshalIndent(groups, "", "  ")
	return os.WriteFile("client_groups.json", data, 0644)
}

// Инициализация групп
func initGroups() {
	groupsList := loadGroups()
	groupsMu.Lock()
	for _, g := range groupsList {
		groups[g.Name] = g
	}
	groupsMu.Unlock()
}

// Получить все группы
func getAllGroups() []ClientGroup {
	groupsMu.RLock()
	defer groupsMu.RUnlock()

	//log.Printf("Всего групп: %d", len(groups))

	var groupNames []string
	for name := range groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	var res []ClientGroup
	for _, name := range groupNames {
		res = append(res, groups[name])
	}

	//log.Printf("Отсортированный список групп: %+v", res)

	return res
}

// Передает данные о группах и участниках
func getAllGroupsForShow() []ClientGroupForShow {
	groupsList := loadGroups()
	var res []ClientGroupForShow

	knownClients := loadClientList("clients_known.json")
	clientMap := make(map[string]ClientInfo)
	for _, c := range knownClients {
		clientMap[c.ID] = c
	}

	// Проверка онлайнов
	connsMu.RLock()
	onlineMap := make(map[string]bool)
	for id := range activeConns {
		onlineMap[id] = true
	}
	connsMu.RUnlock()

	// Обработка групп
	for _, group := range groupsList {
		var members []ClientInfo
		for _, memberID := range group.Members {
			if client, ok := clientMap[memberID]; ok {
				client.Online = onlineMap[client.ID]
				members = append(members, client)
			}
		}

		res = append(res, ClientGroupForShow{
			Name:      group.Name,
			Members:   members,
			CreatedAt: group.CreatedAt,
		})
	}

	return res
}
