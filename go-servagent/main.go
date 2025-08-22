package main

import (
	"log"
	"net"
	"net/http"

	"github.com/gorilla/mux"
)

const clientport = ":8081"
const webport = ":8080"

func main() {
	initGroups()

	listener, err := net.Listen("tcp", clientport)
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

	http.Handle("/", r)

	log.Println("HTTP сервер запущен на " + webport)
	log.Fatal(http.ListenAndServe(webport, nil))
}
