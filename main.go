package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/your/module/internal/api"
	"github.com/your/module/internal/raftnode"
)

func main() {
	raftAddress := flag.String("raft-address", "127.0.0.1:5000", "Raft address")
	apiAddress := flag.String("api-address", "127.0.0.1:8000", "API server address")
	dataDir := flag.String("data-dir", "./raft-data", "Data directory for raft logs")
	bootstrap := flag.Bool("bootstrap", false, "Bootstrap cluster")
	inMemory := flag.Bool("inmem", false, "Use in-memory raft store")
	joinTarget := flag.String("join", "", "Cluster address to join (ex: 127.0.0.1:8000)")
	flag.Parse()

	err := os.MkdirAll(*dataDir, 0700)
	if err != nil {
		log.Fatalf("failed to create raft dir: %s", err)
	}

	node, _, err := raftnode.NewStore(*raftAddress, *dataDir, *inMemory)
	if err != nil {
		log.Fatalf("failed to create store: %s", err)
	}

	if *bootstrap {
		if err := node.BootstrapSelf(*raftAddress); err != nil {
			log.Fatalf("failed to bootstrap: %s", err)
		}
	}

	if *joinTarget != "" {
		url := fmt.Sprintf("http://%s/join?peerAddress=%s", *joinTarget, *raftAddress)
		resp, err := http.Post(url, "application/json", nil)
		if err != nil || resp.StatusCode != 200 {
			log.Fatalf("failed to join at %s: %v", *joinTarget, err)
		}
		log.Printf("joined cluster at %s", *joinTarget)
	}

	mux := http.NewServeMux()
	api.New(node, *apiAddress).RegisterRoutes(mux)

	server := &http.Server{Addr: *apiAddress, Handler: mux}
	go func() {
		log.Printf("server up at %s", *apiAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch

	log.Println("shutting down")
	server.Shutdown(context.Background())
}
