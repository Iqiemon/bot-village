package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"botvillage/demo"
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
	demoMode := flag.Bool("demo", false, "run against generated fake sessions")
	flag.Parse()

	cleanup := func() {}
	if *demoMode {
		dir, err := os.MkdirTemp("", "bot-village-demo-")
		if err != nil {
			log.Fatal(err)
		}
		stop, err := demo.Seed(dir, 12)
		if err != nil {
			os.RemoveAll(dir)
			log.Fatal(err)
		}
		cleanup = func() {
			stop()
			os.RemoveAll(dir)
		}
		data = []string{dir}
		fmt.Println("demo mode: fake sessions only, prompt delivery is faked")
	} else if len(data) == 0 {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		data = append(data, filepath.Join(home, ".claude"))
	}

	cfg := server.Config{DataDirs: data, Listen: *listen, Demo: *demoMode}
	watcher := ingest.NewWatcher(ingest.WatcherConfig{DataDirs: data})
	watcher.Start(context.Background())
	cfg.Events = watcher.Events()
	addr := cfg.Listen
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	fmt.Printf("Bot Village listening on http://%s\n", addr)

	srv := &http.Server{Addr: cfg.Listen, Handler: server.NewMux(cfg)}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-ctx.Done()
		srv.Close()
	}()
	err := srv.ListenAndServe()
	watcher.Stop()
	cleanup()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
