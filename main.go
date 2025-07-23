package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn     *websocket.Conn
	Nickname string
}

type Broadcast struct {
	Message      string
	IgnoreClient *Client
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[*websocket.Conn]*Client)
var broadcast = make(chan Broadcast)

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade the connection to use websocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	client := &Client{Conn: conn, Nickname: ""}
	clients[conn] = client

	// Print info for new connection
	fmt.Printf("New connection from: %s\n", conn.RemoteAddr().String())

	for {
		_, inputBytes, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("Connection closed: %s (%s)\n", conn.RemoteAddr().String(), client.Nickname)
			delete(clients, conn)
			break
		}

		if inputBytes[0] == '/' {
			inputStr := string(inputBytes[1:])
			if trimmedInput, ok := strings.CutPrefix(inputStr, "nickname"); ok {
				// Trim the command
				trimmedInput = strings.TrimSpace(trimmedInput) // Remove leading spaces

				// Check for quotes
				if len(trimmedInput) >= 2 && trimmedInput[0] == '"' && trimmedInput[len(trimmedInput)-1] == '"' {
					// Remove the surrounding quotes
					newNickname := trimmedInput[1 : len(trimmedInput)-1]
					changeClientNickname(client, newNickname)
				} else {
					changeClientNickname(client, trimmedInput)
				}
			}
		} else {
			if client.Nickname == "" {
				fmt.Println("empty nickname detected")
				continue
			}
			formattedMessage := fmt.Sprintf("%s: %s", client.Nickname, inputBytes)
			fmt.Println(formattedMessage)
			broadcast <- Broadcast{Message: formattedMessage, IgnoreClient: nil}
		}
	}
}

func handleMessages() {
	for {
		bc := <-broadcast
		for conn, client := range clients {
			if client == bc.IgnoreClient {
				continue
			}
			err := client.Conn.WriteMessage(websocket.TextMessage, []byte(bc.Message))
			if err != nil {
				client.Conn.Close()
				delete(clients, conn)
			}
		}
	}
}

func changeClientNickname(client *Client, newNickname string) {
	if client.Nickname == newNickname {
		return
	}
	isRename := client.Nickname != ""
	oldNickname := client.Nickname
	client.Nickname = newNickname
	if !isRename {
		formattedMessage := fmt.Sprintf("%s has joined the chat room", newNickname)
		broadcast <- Broadcast{Message: formattedMessage, IgnoreClient: client}
		fmt.Println(formattedMessage)
	} else {
		formattedMessage := fmt.Sprintf("%s has changed their nickname to %s", oldNickname, newNickname)
		broadcast <- Broadcast{Message: formattedMessage, IgnoreClient: nil}
		fmt.Println(formattedMessage)
	}
}

func main() {
	go handleMessages()

	http.HandleFunc("/ws", handleConnections)

	fmt.Println("Chat server starts listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
