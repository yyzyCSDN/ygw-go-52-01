package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"graphstore/internal/rebuild"
	"graphstore/internal/store"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8612", "http listen address")
	dataDir := flag.String("data", "graphstore-data", "directory for the write-ahead log")
	flag.Parse()

	st, err := store.New(store.Options{
		EdgeBuckets:   4,
		ShardCapacity: 256,
		LabelCap:      32,
		EnableWAL:     true,
		WalDir:        filepath.Join(*dataDir, "wal"),
	})
	if err != nil {
		log.Fatalf("graphstore: create store: %v", err)
	}
	defer st.Close()

	if err := seedDemo(st); err != nil {
		log.Fatalf("graphstore: seed demo: %v", err)
	}
	rebuilder := rebuild.New(st, st.LabelIndex())
	if err := rebuilder.Rebuild(context.Background()); err != nil {
		log.Fatalf("graphstore: rebuild label index: %v", err)
	}

	server := newServer(st, *dataDir)
	log.Printf("graphstore listening on %s", *addr)
	if err := server.ListenAndServe(*addr); err != nil {
		log.Fatalf("graphstore: serve: %v", err)
	}
	os.Exit(0)
}
