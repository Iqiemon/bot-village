// Package demo seeds a fake Claude Code data directory so the village runs
// with nothing real running.
package demo

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	projects = []string{
		"my-app", "atlas", "weather-bot", "garden-planner", "site-redesign",
		"cli-tools", "parser", "indexer", "pixel-art", "timezone", "search", "notebook",
	}
	tools = []string{"Edit", "Bash", "Read", "Write", "Grep", "Glob"}
)

var tick = func() time.Duration { return time.Duration(2+rand.Intn(5)) * time.Second }

type sessionFile struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	CWD       string `json:"cwd"`
	Status    string `json:"status"`
	UpdatedAt int64  `json:"updatedAt"`
}

// Seed writes n sessions and transcripts under dir and appends a line to each
// transcript every 2 to 6 seconds. The returned func stops the appending.
func Seed(dir string, n int) (func(), error) {
	var paths []string
	for i := 0; i < n; i++ {
		name := projects[i%len(projects)]
		id := fmt.Sprintf("demo-%02d-%s", i, name)
		b, err := json.Marshal(sessionFile{
			PID:       9000 + i,
			SessionID: id,
			CWD:       "/demo/" + name,
			Status:    "busy",
			UpdatedAt: time.Now().UnixMilli(),
		})
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Join(dir, "sessions"), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, "sessions", fmt.Sprint(9000+i)+".json"), b, 0o644); err != nil {
			return nil, err
		}
		path := filepath.Join(dir, "projects", name, id+".jsonl")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(line(tools[i%len(tools)])), 0o644); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}

	stop := make(chan struct{})
	var once sync.Once
	var wg sync.WaitGroup
	for _, path := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				return
			}
			defer f.Close()
			for {
				t := time.NewTimer(tick())
				select {
				case <-stop:
					t.Stop()
					return
				case <-t.C:
					if _, err := f.WriteString(line(tools[rand.Intn(len(tools))])); err != nil {
						return
					}
				}
			}
		}(path)
	}
	return func() {
		once.Do(func() { close(stop) })
		wg.Wait()
	}, nil
}

func line(tool string) string {
	return `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"` + tool + `","input":{}}]}}` + "\n"
}
