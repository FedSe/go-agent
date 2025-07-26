package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	knownClients := loadClientList("clients_known.json")
	pendingClients := loadClientList("clients_pending.json")

	groupsMu.RLock()
	groupList := getAllGroupsForShow()
	groupsMu.RUnlock()

	// Синхронизируем с registry
	registryMu.RLock()
	var approvedList []ClientInfo
	for _, c := range knownClients {
		client := c

		if regClient, ok := clientRegistry[c.ID]; ok {
			client.IP = regClient.IP
			client.Hostname = regClient.Hostname
			client.LastActiveTime = regClient.LastActiveTime
		}

		client.Online = activeConns[client.ID] != nil
		approvedList = append(approvedList, client)
	}
	registryMu.RUnlock()

	data := struct {
		ApprovedClients []ClientInfo
		PendingClients  []ClientInfo
		Groups          []ClientGroupForShow
		Responses       []string
	}{
		ApprovedClients: approvedList,
		PendingClients:  pendingClients,
		Groups:          groupList,
		Responses:       logs,
	}
	fmt.Println(activeConns)
	render(w, "./templates/home.page.tmpl", data)
}

// Отправка команды клиентам
func sendCommandHandler(w http.ResponseWriter, r *http.Request) {
	clientID := r.FormValue("client_id")
	command := r.FormValue("command")

	if clientID == "" || command == "" {
		http.Error(w, "Не указан client_id или команда", http.StatusBadRequest)
		return
	}

	if clientID == "all" {
		connsMu.RLock()
		for id := range activeConns {
			sendCommandToClient(id, command)
		}
		connsMu.RUnlock()
	} else {
		sendCommandToClient(clientID, command)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Подтверждение клиента
func approveClientHandler(w http.ResponseWriter, r *http.Request) {
	clientID := r.FormValue("client_id")
	if clientID == "" {
		http.Error(w, "Не указан ID", http.StatusBadRequest)
		return
	}

	// Загружаем pending
	pendingList := loadClientList("clients_pending.json")

	index := -1
	for i, c := range pendingList {
		if c.ID == clientID {
			index = i
			break
		}
	}

	if index == -1 {
		http.Error(w, "Клиент не найден в списке ожидания", http.StatusNotFound)
		return
	}

	// Удаляем из pending
	pendingList = append(pendingList[:index], pendingList[index+1:]...)
	saveClientList("clients_pending.json", pendingList)

	// Добавляем в known (если ещё не там)
	knownList := loadClientList("clients_known.json")
	exists := false
	for _, c := range knownList {
		if c.ID == clientID {
			exists = true
			break
		}
	}

	if !exists {
		newClient := ClientInfo{
			ID:             clientID,
			IP:             "N/A",
			LastActiveTime: time.Now(),
		}
		knownList = append(knownList, newClient)
		saveClientList("clients_known.json", knownList)
	}

	registryMu.Lock()
	clientRegistry[clientID] = ClientInfo{
		ID:             clientID,
		IP:             "N/A",
		LastActiveTime: time.Now(),
	}
	registryMu.Unlock()

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Удаление клиента
func deleteClientHandler(w http.ResponseWriter, r *http.Request) {
	clientID := r.FormValue("client_id")
	if clientID == "" {
		http.Error(w, "Не указан client_id", http.StatusBadRequest)
		return
	}

	// из known
	knownList := loadClientList("clients_known.json")
	knownList = removeClientFromList(clientID, knownList)
	saveClientList("clients_known.json", knownList)

	// из pending
	pendingList := loadClientList("clients_pending.json")
	pendingList = removeClientFromList(clientID, pendingList)
	saveClientList("clients_pending.json", pendingList)

	// из всех групп
	groupsMu.Lock()
	for groupName, group := range groups {
		var updatedMembers []string
		for _, memberID := range group.Members {
			if memberID != clientID {
				updatedMembers = append(updatedMembers, memberID)
			}
		}
		group.Members = updatedMembers
		groups[groupName] = group
	}
	groupsMu.Unlock()

	err := saveGroups(getAllGroups())
	if err != nil {
		log.Println("Ошибка сохранения групп:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	registryMu.Lock()
	delete(clientRegistry, clientID)
	registryMu.Unlock()

	connsMu.Lock()
	delete(activeConns, clientID)
	connsMu.Unlock()

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Группы
func groupsHandler(w http.ResponseWriter, r *http.Request) {
	groupsMu.RLock()
	defer groupsMu.RUnlock()

	data := struct {
		Groups []ClientGroupForShow
	}{
		Groups: getAllGroupsForShow(),
	}

	render(w, "./templates/groups.page.tmpl", data)
}

// Создание группы
func createGroupHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("group_name")
	if name == "" {
		http.Error(w, "Не указано имя группы", http.StatusBadRequest)
		return
	}

	groupsMu.Lock()

	// Проверка существования группы
	if _, ok := groups[name]; ok {
		groupsMu.Unlock()
		http.Error(w, "Группа с таким именем уже существует", http.StatusConflict)
		return
	}

	// Создаём новую группу
	newGroup := ClientGroup{
		Name:      name,
		Members:   []string{},
		CreatedAt: time.Now(),
	}

	groups[name] = newGroup

	groupsMu.Unlock()
	err := saveGroups(getAllGroups())
	if err != nil {
		log.Println("Ошибка сохранения групп:", err)
		http.Error(w, "Не удалось сохранить группу", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/groups", http.StatusSeeOther)
}

// Добавление клиента в группу
func addClientToGroupHandler(w http.ResponseWriter, r *http.Request) {
	groupName := r.FormValue("group_name")
	clientIDs := r.Form["client_id"]

	if groupName == "" || len(clientIDs) == 0 {
		http.Error(w, "Не указаны параметры", http.StatusBadRequest)
		return
	}

	groupsMu.Lock()

	group, ok := groups[groupName]
	if !ok {
		groupsMu.Unlock()
		http.Error(w, "Группа не найдена", http.StatusNotFound)
		return
	}

	for _, clientID := range clientIDs {
		exists := false
		for _, m := range group.Members {
			if m == clientID {
				exists = true
				break
			}
		}

		if !exists {
			group.Members = append(group.Members, clientID)
		}
	}

	groups[groupName] = group

	groupsMu.Unlock()

	err := saveGroups(getAllGroups())
	if err != nil {
		log.Println("Ошибка сохранения групп:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/groups", http.StatusSeeOther)
}

// Удаление клиента из группы
func removeClientFromGroupHandler(w http.ResponseWriter, r *http.Request) {
	groupName := r.FormValue("group_name")
	clientID := r.FormValue("client_id")

	if groupName == "" || clientID == "" {
		http.Error(w, "Не указаны параметры", http.StatusBadRequest)
		return
	}

	groupsMu.Lock()

	group, ok := groups[groupName]
	if !ok {
		groupsMu.Unlock()
		http.Error(w, "Группа не найдена", http.StatusBadRequest)
		return
	}

	index := -1
	for i, id := range group.Members {
		if id == clientID {
			index = i
			break
		}
	}

	if index != -1 {
		group.Members = append(group.Members[:index], group.Members[index+1:]...)
		groups[groupName] = group
	}

	groupsMu.Unlock()

	if index != -1 {
		err := saveGroups(getAllGroups())
		if err != nil {
			log.Println("Ошибка сохранения групп:", err)
			http.Error(w, "Не удалось сохранить группы", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/groups", http.StatusSeeOther)
}

// Отправляет клиентов, доступных для добавления в заданную группу
func getAvailableClientsHandler(w http.ResponseWriter, r *http.Request) {
	groupName := r.URL.Query().Get("group_name")
	if groupName == "" {
		http.Error(w, "Не указано имя группы", http.StatusBadRequest)
		return
	}

	groupsMu.RLock()
	defer groupsMu.RUnlock()

	group, ok := groups[groupName]
	if !ok {
		http.Error(w, "Группа не найдена", http.StatusNotFound)
		return
	}

	// Одобренные клиенты
	allClients := loadClientList("clients_known.json")

	// Фильтр, кого нет в группе
	var available []ClientInfo
	for _, client := range allClients {
		found := false
		for _, member := range group.Members {
			if client.ID == member {
				found = true
				break
			}
		}
		if !found {
			available = append(available, client)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(available)
}

// Удаление группы
func deleteGroupHandler(w http.ResponseWriter, r *http.Request) {
	groupName := r.FormValue("group_name")
	if groupName == "" {
		http.Error(w, "Не указано имя группы", http.StatusBadRequest)
		return
	}

	groupsMu.Lock()
	delete(groups, groupName)
	groupsMu.Unlock()

	err := saveGroups(getAllGroups())
	if err != nil {
		log.Println("Ошибка сохранения групп:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/groups", http.StatusSeeOther)
}

// Отправляет команды группе
func sendCommandToGroupHandler(w http.ResponseWriter, r *http.Request) {
	groupName := r.FormValue("group_name")
	command := r.FormValue("command")

	groupsMu.RLock()
	group, ok := groups[groupName]
	groupsMu.RUnlock()

	if !ok {
		http.Error(w, "Группа не найдена", http.StatusBadRequest)
		return
	}

	for _, clientID := range group.Members {
		sendCommandToClient(clientID, command)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
