package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/nxadm/tail"
	"github.com/nxadm/tail/ratelimiter"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

func startHttp(quit <-chan struct{}, conf httpConfig, db database.Database, finished chan<- struct{}) error {
	r := mux.NewRouter()

	// TODO: afegir autenticació TOKEN_CLIENT
	r.HandleFunc("/experiments", handleGetJobs(db)).Methods("GET")
	r.HandleFunc("/experiments", handleNewJob(db)).Methods("POST")
	r.HandleFunc("/experiments/{id}", handleGetById(db)).Methods("GET")
	r.HandleFunc("/experiments/{id}", handleJobUpdate(db)).Methods("PUT")
	r.HandleFunc("/logs/{id}", handleGetLogs()).Methods("GET")
	r.HandleFunc("/logs/{id}/tail", handleFollowLogs()).Methods("GET")

	// TODO: afegir autenticació TOKEN_WORKER
	r.HandleFunc("/workers/poll", handleWorkerFetch(db)).Methods("GET")
	r.HandleFunc("/logs/{id}/upload", handleWorkerLogs()).Methods("GET")

	h := handlers.RecoveryHandler(handlers.PrintRecoveryStack(true))(
		handlers.CombinedLoggingHandler(os.Stderr,
			handlers.CompressHandler(r)))

	srv := &http.Server{
		Addr: conf.Listen,
		// deshabilitat per poder fer streaming de logs
		// WriteTimeout: conf.WriteTimeout,
		ReadTimeout: conf.ReadTimeout,
		IdleTimeout: conf.IdleTimeout,
		Handler:     h,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		<-quit

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("error http server shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("started http server: %s\n", conf.Listen)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		return fmt.Errorf("running http server: %w", err)
	}

	<-idleConnsClosed
	finished <- struct{}{}

	return nil
}

func handleWorkerFetch(db database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		job, err := db.FetchJob(r.Context())
		if err != nil {
			http.Error(w, "Error fetching jobs: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if job == nil {
			http.Error(w, "No job found", http.StatusNoContent)
			return
		}

		_ = json.NewEncoder(w).Encode(job)
	}
}

func handleWorkerLogs() http.HandlerFunc {
	upgrader := websocket.Upgrader{}
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := uuid.FromString(vars["id"])
		if err != nil {
			http.Error(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		logFile, err := os.OpenFile(fmt.Sprintf("./logs/%v.log", id), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("error creating log file: %v", err)
			http.Error(w, "Error creating log file", http.StatusInternalServerError)
			return
		}
		defer logFile.Close()

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "Error upgrading to websocket", http.StatusBadRequest)
			return
		}
		defer conn.Close()

		for {
			messageType, r, err := conn.NextReader()
			if err != nil {
				log.Printf("error reading ws message: %v", err)
				break
			}

			// logs are binary messages, discard others
			if messageType != websocket.BinaryMessage {
				continue
			}

			n, err := io.Copy(logFile, r)
			if err != nil {
				log.Printf("error copying log contents to file: %v", err)
				return
			}
			log.Printf("wrote %d bytes into %v log file", n, id)
		}
	}
}

func handleGetJobs(_ database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}
}

func handleNewJob(db database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var jobRequest database.InsertParams
		if err := json.NewDecoder(r.Body).Decode(&jobRequest); err != nil {
			http.Error(w, "error decoding json", http.StatusBadRequest)
			log.Printf("%+v\n", err)
			return
		}

		job, err := db.Insert(r.Context(), jobRequest)
		if err != nil {
			http.Error(w, fmt.Sprintf("error inserting job: %v\n", err), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(job)
	}
}

func handleJobUpdate(db database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := uuid.FromString(vars["id"])
		if err != nil {
			http.Error(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		var changes database.UpdateParams
		if err := json.NewDecoder(r.Body).Decode(&changes); err != nil {
			http.Error(w, "Error decoding json: "+err.Error(), http.StatusBadRequest)
			return
		}
		changes.Id = id

		job, err := db.Update(r.Context(), changes)
		if err != nil {
			http.Error(w, "Error updating job: "+err.Error(), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(job)
	}
}

func handleGetById(db database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := uuid.FromString(vars["id"])
		if err != nil {
			http.Error(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		job, err := db.GetById(r.Context(), id)
		if err != nil {
			http.Error(w, "job does not exist", http.StatusNotFound)
			return
		}

		if job == nil {
			http.Error(w, "Job with given ID not found", http.StatusNotFound)
			return
		}

		_ = json.NewEncoder(w).Encode(job)
	}
}

func handleGetLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := uuid.FromString(vars["id"])
		if err != nil {
			http.Error(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		f, err := os.OpenFile(fmt.Sprintf("./logs/%v.log", id), os.O_RDONLY, 0644)
		if err != nil {
			http.Error(w, fmt.Sprintf("error opening log file: %v", err), http.StatusInternalServerError)
			return
		}

		if _, err := io.Copy(w, f); err != nil {
			http.Error(w, fmt.Sprintf("error copying log file to body: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func handleFollowLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			panic("expected http.ResponseWriter to be an http.Flusher")
		}
		defer flusher.Flush()

		vars := mux.Vars(r)
		id, err := uuid.FromString(vars["id"])
		if err != nil {
			http.Error(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		t, err := tail.TailFile(fmt.Sprintf("./logs/%v.log", id), tail.Config{
			ReOpen:      false,
			MustExist:   false, // podem fer el tail abans que existeixi l'execució
			Follow:      true,
			MaxLineSize: 0,
			// TODO: configurar ratelimit
			RateLimiter: ratelimiter.NewLeakyBucket(64, time.Second),
			Logger:      nil,
		})
		if err != nil {
			http.Error(w, "Error tailing: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer t.Cleanup()

		w.Header().Set("X-Content-Type-Options", "nosniff")

		for {
			select {
			case line := <-t.Lines:
				if err := line.Err; err != nil {
					if err.Error() == "Too much log activity; waiting a second before resuming tailing" {
						continue
					}
					log.Printf("err = %+v\n", err)
					http.Error(w, "error tailing: "+err.Error(), http.StatusInternalServerError)
					return
				}

				text := line.Text
				if strings.Contains(text, jobs.MagicEnd) {
					text = strings.TrimSuffix(text, jobs.MagicEnd)
					_, _ = w.Write([]byte(fmt.Sprintf("%s\n", text)))
					flusher.Flush()
					return
				}
				_, _ = w.Write([]byte(fmt.Sprintf("%s\n", text)))
				flusher.Flush()
				break
			}
		}
	}
}
