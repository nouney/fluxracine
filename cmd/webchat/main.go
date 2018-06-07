package main

import (
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nouney/fluxracine/internal/db/redis"
	"github.com/nouney/fluxracine/pkg/chat"
	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

var (
	upgrader websocket.Upgrader
	server   *chat.Server
	port     string
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	port = os.Getenv("HTTP_LISTEN_PORT")
	if port == "" {
		port = "8000"
	}

	opts := []chat.Opt{}
	clusterHTTPListenPort := os.Getenv("CLUSTER_HTTP_LISTEN_PORT")
	if clusterHTTPListenPort == "" {
		clusterHTTPListenPort = "3000"
	}
	podIP := os.Getenv("POD_IP")
	opts = append(opts, chat.WithHTTPAddress(podIP+":"+clusterHTTPListenPort))

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

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGTERM,
	)
	go func() {
		<-sigc
		server.GracefulShutdown()
	}()
	server.Run()
}

func main() {
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/chat", handleChatSession)

	log.Infof("listening on :%s", port)
	http.ListenAndServe(":"+port, nil)
}
