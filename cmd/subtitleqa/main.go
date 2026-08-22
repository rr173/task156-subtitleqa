// Command subtitleqa is the entry point of the accessible-subtitle timeline
// proofreading workbench. It supports --smoke-test for offline self-checking
// (used by the Docker build gates) and a long-running HTTP mode.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"task156-subtitleqa/internal/demo"
	"task156-subtitleqa/internal/httpapi"
	"task156-subtitleqa/internal/metrics"
	"task156-subtitleqa/internal/service"
	"task156-subtitleqa/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "task156-subtitleqa.db", "SQLite database path")
	smoke := flag.Bool("smoke-test", false, "run embedded end-to-end self-check and exit")
	flag.Parse()
	if err := run(*addr, *dbPath, *smoke); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(addr, dbPath string, smoke bool) error {
	if smoke {
		res, err := demo.Seed(context.Background())
		if err != nil {
			return err
		}
		fmt.Printf("smoke test passed: media=%s segments=%d quality=%d versions=%d conflicts=%d withdrawals=%d persisted=%v\n",
			res.MediaID, res.Segments, res.Quality, res.Versions, res.Conflicts, res.Withdrawals, res.Persisted)
		return nil
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()
	svc := service.New(s)
	if err := svc.Recover(context.Background()); err != nil {
		return err
	}
	api := httpapi.New(svc, &metrics.Metrics{})
	log.Printf("subtitle proofreading service listening on %s (db=%s)", addr, dbPath)
	return http.ListenAndServe(addr, api)
}
