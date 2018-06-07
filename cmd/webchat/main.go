package main

import (
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/nouney/fluxracine/internal/db/redis"
	"github.com/nouney/fluxracine/pkg/chat"

	"github.com/gorilla/websocket"
)

var (
	upgrader websocket.Upgrader
	server   *chat.Server
	port     string
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	opts := []chat.Opt{}

	port = os.Getenv("HTTP_LISTEN_PORT")
	if port == "" {
		port = "8000"
	}

	podIP := os.Getenv("POD_IP")
	if podIP != "" {
		opts = append(opts, chat.WithHTTPAddress(podIP+":8000"))
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		panic("REDIS_ADDR is missing")
	}

	db, err := redis.New(redisAddr, "")
	if err != nil {
		panic(err)
	}

	server, err = chat.NewServer(db, opts...)
	if err != nil {
		panic(err)
	}

	server.Run()
}

func main() {
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/chat", handleChatSession)
	http.ListenAndServe(":"+port, nil)
}
