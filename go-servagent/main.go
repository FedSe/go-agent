package main

import (
	"log"
	"net"
	"net/http"

	"github.com/gorilla/mux"
)

const webport = ":8080"
const clientport = ":8081"

func main() {
	initGroups()

	listener, err := net.Listen("tcp", webport)
	if err != nil {
		log.Fatal("Не удалось запустить TCP сервер:", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Println("Ошибка при подключении клиента:", err)
				continue
			}
			go handleConnection(conn)
		}
	}()

	r := mux.NewRouter()
	registerRoutes(r)

	//	http.Handle("/", r)

	log.Println("HTTP сервер запущен на " + clientport)
	log.Fatal(http.ListenAndServe(clientport, nil))
}
