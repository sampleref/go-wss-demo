package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Payload struct {
	Message  string `json:"message"`
	Type     string `json:"type"`
	ClientId string `json:"clientId"`
}

var sdpAnswers map[string]string
var wsConnections map[string]*websocket.Conn

func main() {
	wsConnections = make(map[string]*websocket.Conn)
	sdpAnswers = make(map[string]string)

	go HandleWhipClients()
	go HandleWebClients()

	// create a channel to subscribe ctrl+c/SIGINT event
	sigInterruptChannel := make(chan os.Signal, 1)
	signal.Notify(sigInterruptChannel, os.Interrupt)
	// block execution from continuing further until SIGINT comes
	<-sigInterruptChannel
	log.Printf("Shutting down... ")
}

func HandleWhipClients() {
	var whipHttp = http.NewServeMux()

	whipHttp.HandleFunc("/whip", func(w http.ResponseWriter, r *http.Request) {
		clientId := r.URL.Query().Get("clientId")
		sdpAnswers[clientId] = ""
		// Read the offer from HTTP Request
		fmt.Printf("HTTP Method Type: \n---\n%s from client: %s \n---\n", r.Method, clientId)
		fmt.Printf("HTTP Content Type: \n---\n%s\n---\n", r.Header.Get("Content-Type"))
		if r.Method == "DELETE" {
			fmt.Printf("WHIP DELETE: \n---\n%s\n---\n", r.Method)
		}
		if r.Method != "POST" {
			fmt.Printf("HTTP Method Type Unhandled: \n---\n%s\n---\n", r.Method)
			return
		}
		offer, err := io.ReadAll(r.Body)
		fmt.Printf("Offer Received: \n---\n%s\n---\n", offer)
		if err != nil {
			panic(err)
			return
		}

		wsConn := wsConnections[clientId]
		if wsConn == nil {
			fmt.Printf("Invalid Client/No Conn found: \n---\n%s\n---\n", clientId)
			return
		}

		WriteMessage(wsConn, string(offer), "sdpOffer")
		// Wait and Write Answer with Candidates as HTTP Response
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				fmt.Println(ctx.Err())
				return
			default:
				if len(sdpAnswers[clientId]) != 0 {
					// Send answer via HTTP Response
					// WHIP+WHEP expects a Location header and a HTTP Status Code of 201
					w.Header().Add("Location", "/whip")
					w.WriteHeader(http.StatusCreated)
					fmt.Printf("Answer Sent: \n---\n%s\n---\n", sdpAnswers[clientId])
					fmt.Fprint(w, sdpAnswers[clientId])
					sdpAnswers[clientId] = ""
					cancel()
				}
				time.Sleep(time.Second)
			}
		}
	})

	log.Printf("Starting server: %s", "http://127.0.0.1:8081")
	err := http.ListenAndServe(":8081", whipHttp)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func HandleWebClients() {
	var webHttp = http.NewServeMux()

	webHttp.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("Received connect: %s", r.RemoteAddr)
		go HandleWsMessages(conn)
	})

	webHttp.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "ws_client.html")
	})

	log.Printf("Starting server: %s", "https://127.0.0.1:8080")
	err := http.ListenAndServeTLS(":8080", "server.crt", "server.key", webHttp)
	if err != nil {
		log.Fatal("ListenAndServeTLS: ", err)
	}
}

func HandleWsMessages(conn *websocket.Conn) {
	for {
		// Read message from browser
		var payload Payload
		err := conn.ReadJSON(&payload)
		if err != nil {
			log.Printf("Error reading ws payload: %s\n", err)
			break
		}

		if len(payload.ClientId) == 0 {
			log.Printf("Invalid ClientId: %s\n", err)
			break
		}

		switch payload.Type {
		case "text":
			fmt.Printf("Sent to: %s\n", conn.RemoteAddr())
			WriteMessage(conn, payload.Message, payload.Type)
			break
		case "clientId":
			wsConnections[payload.ClientId] = conn
			break
		case "sdpAnswer":
			sdpAnswers[payload.ClientId] = payload.Message
			break
		default:
			// Print the message to the console
			fmt.Printf("Sent to: %s\n", conn.RemoteAddr())
			log.Printf("Invalid ws payload type: %s\n", payload.Type)
			WriteMessage(conn, "Invalid Type", payload.Type)
		}
	}
	fmt.Printf("Closing conn: %s\n", conn.RemoteAddr())
	conn.Close()
}

func WriteMessage(conn *websocket.Conn, message string, payloadType string) {
	payload := Payload{
		Message: message,
		Type:    payloadType,
	}

	err := conn.WriteJSON(payload)
	if err != nil {
		log.Printf("Error sending json: %s\n", err)
	}
	return
}
