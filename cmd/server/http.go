package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gofrs/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/hpcloud/tail"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

func startHttp(quit <-chan struct{}, conf httpConfig, db database.Database, finished chan<- struct{}) error {
	r := mux.NewRouter()

	r.HandleFunc("/experiments", handleGetJobs(db)).Methods("GET")
	r.HandleFunc("/experiments", handleNewJob(db)).Methods("POST")
	r.HandleFunc("/experiments/{id}", handleGetById(db)).Methods("GET")
	r.HandleFunc("/experiments/{id}", handleJobUpdate(db)).Methods("PUT")
	r.HandleFunc("/logs/{id}", handleGetLogs()).Methods("GET")
	r.HandleFunc("/logs/{id}/tail", handleFollowLogs()).Methods("GET")

	r.HandleFunc("/workers/poll", handleWorkerFetch(db)).Methods("GET")
	r.HandleFunc("/logs/{id}/upload", handleWorkerLogs()).Methods("GET")

	h := handlers.RecoveryHandler(handlers.PrintRecoveryStack(true))(
		handlers.CombinedLoggingHandler(os.Stderr,
			handlers.CompressHandler(authMiddleware(r, conf.Tokens))))

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
			messageType, read, err := conn.NextReader()
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					return
				}
				log.Printf("error reading ws message: %v", err)
				break
			}

			// logs are binary messages, discard others
			if messageType != websocket.BinaryMessage {
				continue
			}

			if _, err := io.Copy(logFile, read); err != nil {
				log.Printf("error copying log contents to file: %v", err)
				return
			}
		}
	}
}

func handleGetJobs(_ database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: implement
		_, _ = w.Write([]byte("ok"))
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
	upgrader := websocket.Upgrader{}
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := uuid.FromString(vars["id"])
		if err != nil {
			http.Error(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		// function to send to either websockets or http flushing
		var sendFunc func([]byte) error
		if r.URL.Query().Has("ws") {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				http.Error(w, "error upgrading websocket: "+err.Error(), http.StatusBadRequest)
				return
			}
			defer conn.Close()
			sendFunc = func(b []byte) error {
				return conn.WriteMessage(websocket.TextMessage, b)
			}
		} else {
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "expected connection to be a flusher", http.StatusBadRequest)
				return
			}
			defer flusher.Flush()

			w.Header().Set("X-Content-Type-Options", "nosniff")

			sendFunc = func(b []byte) error {
				b = append(b, '\n')
				_, err := w.Write(b)
				if err == nil {
					flusher.Flush()
				}
				return err
			}
		}

		t, err := tail.TailFile(fmt.Sprintf("./logs/%v.log", id), tail.Config{
			ReOpen:      false,
			MustExist:   false, // podem fer el tail abans que existeixi l'execució
			Poll:        true,
			Follow:      true,
			MaxLineSize: 0,
			Logger:      nil,
		})
		if err != nil {
			http.Error(w, "Error tailing: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer t.Cleanup()
		defer t.Stop()

		for {
			select {
			case <-r.Context().Done():
				return

			case line := <-t.Lines:
				if err := line.Err; err != nil {
					log.Printf("err = %+v\n", err)
					http.Error(w, "error tailing: "+err.Error(), http.StatusInternalServerError)
					return
				}

				text := line.Text
				if text == jobs.MagicEnd {
					return
				}

				if err := sendFunc([]byte(text)); err != nil {
					log.Printf("error sending message: %v", err)
					return
				}
				break
			}
		}
	}
}

func isValid(token string, allowedTokens []string) bool {
	for _, x := range allowedTokens {
		if token == x {
			return true
		}
	}
	return false
}

func authMiddleware(next http.Handler, tokens []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok := isValid(r.Header.Get("Authorization"), tokens)
		if ok {
			next.ServeHTTP(w, r)
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
	}
}
