package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

var (
	clients   = make(map[*websocket.Conn]bool)
	broadcast = make(chan Message)
	upgrader  = websocket.Upgrader{}
	mutex     sync.Mutex
)

type Message struct {
	IR     string `json:"ir"`
	BPM    string `json:"bpm"`
	AvgBpm string `json:"average_bpm"`
}

func main() {
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}).Handler(http.DefaultServeMux)

	http.HandleFunc("/", handleConnections)

	server := &http.Server{
		Addr:    ":8080",
		Handler: corsHandler,
	}

	fmt.Println("Servidor escuchando en el puerto 8080")
	log.Fatal(server.ListenAndServe())
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	mutex.Lock()
	clients[conn] = true
	mutex.Unlock()

	log.Printf("Cliente conectado desde la dirección IP: %s", conn.RemoteAddr())

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error al leer mensaje: %v", err)
			deleteClient(conn)
			break
		}
		broadcast <- msg
	}

	conn.Close()
}

func deleteClient(conn *websocket.Conn) {
	mutex.Lock()
	delete(clients, conn)
	mutex.Unlock()
}

func init() {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	go func() {
		for {
			msg := <-broadcast

			for client := range clients {
				err := client.WriteJSON(msg)
				if err != nil {
					log.Printf("Error al escribir mensaje: %v", err)
					client.Close()
					deleteClient(client)
				}
			}
		}
	}()
}
