package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	s "github.com/ryota0624/multi-server"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World")
}

func longliveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Hello, World\n")
	time.Sleep(time.Second * 3)
	fmt.Fprintf(w, "Hello, World2\n")
}
func main() {
	var h http.HandlerFunc = handler
	srv1 := &http.Server{
		Handler: h,
	}
	lis1, err := net.Listen("tcp", fmt.Sprintf(":%d", 8000))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var h2 http.HandlerFunc = longliveHandler
	srv2 := &http.Server{
		Handler: h2,
	}

	lis2, err := net.Listen("tcp", fmt.Sprintf(":%d", 8001))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	servers := s.NewServers().
		Resister(
			s.NewHttpServer(srv1, lis1),
		).
		Resister(
			s.NewHttpServer(srv2, lis2),
		).
		ShutdownTimout(time.Second * 6).
		EnableShutdownOnTerminateSignal()

	err = servers.Start(context.Background())
	if err != nil {
		log.Printf("failed to listen: %v\n", err)
	}
	servers.WaitShutdown()
	log.Printf("shutdown\n")

}
