package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"botvillage/ingest"
	"botvillage/server"
)

type stringList []string

func (s *stringList) String() string { return fmt.Sprint([]string(*s)) }

func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func main() {
	var data stringList
	flag.Var(&data, "data", "Claude Code data directory (repeatable)")
	listen := flag.String("listen", ":8040", "listen address")
	demo := flag.Bool("demo", false, "run against generated fake sessions")
	flag.Parse()

	if len(data) == 0 {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		data = append(data, filepath.Join(home, ".claude"))
	}

	cfg := server.Config{DataDirs: data, Listen: *listen, Demo: *demo}
	watcher := ingest.NewWatcher(ingest.WatcherConfig{DataDirs: data})
	watcher.Start(context.Background())
	defer watcher.Stop()
	cfg.Events = watcher.Events()
	fmt.Printf("Bot Village listening on %s\n", cfg.Listen)
	log.Fatal(http.ListenAndServe(cfg.Listen, server.NewMux(cfg)))
}
