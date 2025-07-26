package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

func registerRoutes(r *mux.Router) {
	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/send", sendCommandHandler).Methods("POST")
	r.HandleFunc("/approve", approveClientHandler).Methods("POST")

	r.HandleFunc("/client/delete", deleteClientHandler).Methods("POST")

	r.HandleFunc("/groups", groupsHandler).Methods("GET")
	r.HandleFunc("/group/create", createGroupHandler).Methods("POST")
	r.HandleFunc("/group/add", addClientToGroupHandler).Methods("POST")
	r.HandleFunc("/group/remove", removeClientFromGroupHandler).Methods("POST")
	r.HandleFunc("/group/delete", deleteGroupHandler).Methods("POST")
	r.HandleFunc("/group/send", sendCommandToGroupHandler).Methods("POST")

	r.HandleFunc("/api/group/available", getAvailableClientsHandler).Methods("GET")

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

}
