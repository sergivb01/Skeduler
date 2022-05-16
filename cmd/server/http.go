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

func startHttp(quit <-chan struct{}, cfg conf, db database.Database, finished chan<- struct{}) error {
	r := mux.NewRouter()

	s := &httpServer{
		db: db,
		t: &telegramClient{
			Token:  cfg.Telegram.Token,
			ChatId: cfg.Telegram.ChatId,
		},
	}

	r.HandleFunc("/experiments", s.handleGetJobs()).Methods("GET")
	r.HandleFunc("/experiments", s.handleNewJob()).Methods("POST")
	r.HandleFunc("/experiments/{id}", s.handleGetById()).Methods("GET")
	r.HandleFunc("/experiments/{id}", s.handleJobUpdate()).Methods("PUT")
	r.HandleFunc("/logs/{id}", s.handleGetLogs()).Methods("GET")
	r.HandleFunc("/logs/{id}/tail", s.handleFollowLogs()).Methods("GET")

	r.HandleFunc("/workers/poll", s.handleWorkerFetch()).Methods("GET")
	r.HandleFunc("/logs/{id}/upload", s.handleWorkerLogs()).Methods("GET")

	h := handlers.RecoveryHandler(handlers.PrintRecoveryStack(true))(
		handlers.CombinedLoggingHandler(os.Stderr,
			handlers.CompressHandler(authMiddleware(r, cfg.Tokens))))

	srv := &http.Server{
		Addr: cfg.Http.Listen,
		// deshabilitat per poder fer streaming de logs
		// WriteTimeout: conf.WriteTimeout,
		ReadTimeout: cfg.Http.ReadTimeout,
		IdleTimeout: cfg.Http.IdleTimeout,
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

	log.Printf("started http server: %s\n", cfg.Http.Listen)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		return fmt.Errorf("running http server: %w", err)
	}

	<-idleConnsClosed
	finished <- struct{}{}

	return nil
}

type httpServer struct {
	db database.Database
	t  *telegramClient
}

func (h *httpServer) handleWorkerFetch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		job, err := h.db.FetchJob(r.Context())
		if err != nil {
			errorHttp(w, "Error fetching jobs: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if job == nil {
			errorHttp(w, "No job found", http.StatusNoContent)
			return
		}

		_ = json.NewEncoder(w).Encode(job)
	}
}

func (h *httpServer) handleWorkerLogs() http.HandlerFunc {
	upgrader := websocket.Upgrader{}
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := uuid.FromString(vars["id"])
		if err != nil {
			errorHttp(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		logFile, err := os.OpenFile(fmt.Sprintf("./logs/%v.log", id), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("error creating log file: %v", err)
			errorHttp(w, "Error creating log file", http.StatusInternalServerError)
			return
		}
		defer logFile.Close()

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			errorHttp(w, "Error upgrading to websocket", http.StatusBadRequest)
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

func (h *httpServer) handleGetJobs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		job, err := h.db.GetAll(r.Context())
		if err != nil {
			errorHttp(w, fmt.Sprintf("error getting all jobs: %v\n", err), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(job)
	}
}

func (h *httpServer) handleNewJob() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var jobRequest database.InsertParams
		if err := json.NewDecoder(r.Body).Decode(&jobRequest); err != nil {
			errorHttp(w, "error decoding json", http.StatusBadRequest)
			log.Printf("%+v\n", err)
			return
		}

		job, err := h.db.Insert(r.Context(), jobRequest)
		if err != nil {
			errorHttp(w, fmt.Sprintf("error inserting job: %v\n", err), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(job)
	}
}

func (h *httpServer) handleJobUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := uuid.FromString(vars["id"])
		if err != nil {
			errorHttp(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		var changes database.UpdateParams
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&changes); err != nil {
			errorHttp(w, "Error decoding json: "+err.Error(), http.StatusBadRequest)
			return
		}
		changes.Id = id

		job, err := h.db.Update(r.Context(), changes)
		if err != nil {
			errorHttp(w, "Error updating job: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(job); err != nil {
			log.Printf("error encoding job: %v", err)
		}

		if err := h.t.sendNotification(*job); err != nil {
			log.Printf("error sending job update notification via telegram: %v", err)
		}
	}
}

func (h *httpServer) handleGetById() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := uuid.FromString(vars["id"])
		if err != nil {
			errorHttp(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		job, err := h.db.GetById(r.Context(), id)
		if err != nil {
			errorHttp(w, "job does not exist", http.StatusNotFound)
			return
		}

		if job == nil {
			errorHttp(w, "job with given ID not found", http.StatusNotFound)
			return
		}

		_ = json.NewEncoder(w).Encode(job)
	}
}

func (h *httpServer) handleGetLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := uuid.FromString(vars["id"])
		if err != nil {
			errorHttp(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		// TODO: check if job exists
		f, err := os.OpenFile(fmt.Sprintf("./logs/%v.log", id), os.O_RDONLY, 0644)
		if err != nil {
			errorHttp(w, fmt.Sprintf("error opening log file: %v", err), http.StatusInternalServerError)
			return
		}

		// TODO: add support for range requests
		if _, err := io.Copy(w, f); err != nil {
			errorHttp(w, fmt.Sprintf("error copying log file to body: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func (h *httpServer) handleFollowLogs() http.HandlerFunc {
	upgrader := websocket.Upgrader{}
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := uuid.FromString(vars["id"])
		if err != nil {
			errorHttp(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		// function to send to either websockets or http flushing
		var sendFunc func([]byte) error
		if r.URL.Query().Has("ws") {
			// upgrade the connection to a WebSocket connection
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				errorHttp(w, "error upgrading websocket: "+err.Error(), http.StatusBadRequest)
				return
			}
			defer conn.Close()
			sendFunc = func(b []byte) error {
				return conn.WriteMessage(websocket.BinaryMessage, b)
			}
		} else {
			flusher, ok := w.(http.Flusher)
			if !ok {
				errorHttp(w, "expected connection to be a flusher", http.StatusBadRequest)
				return
			}
			defer flusher.Flush()

			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Content-Type", "text/plain")

			sendFunc = func(b []byte) error {
				b = append(b, '\n')
				_, err := w.Write(b)
				if err == nil {
					flusher.Flush()
				}
				return err
			}
		}

		// open log file and follow it
		t, err := tail.TailFile(fmt.Sprintf("./logs/%v.log", id), tail.Config{
			ReOpen:      false,
			MustExist:   false, // podem fer el tail abans que existeixi l'execució
			Poll:        true,
			Follow:      true,
			MaxLineSize: 0,
			Logger:      nil,
		})
		if err != nil {
			errorHttp(w, "Error tailing: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer t.Cleanup()
		defer t.Stop()

		for {
			select {
			// if the connection is closed, stop the loop
			case <-r.Context().Done():
				return

			case line := <-t.Lines:
				if err := line.Err; err != nil {
					log.Printf("err = %+v\n", err)
					errorHttp(w, "error tailing: "+err.Error(), http.StatusInternalServerError)
					return
				}

				text := line.Text
				// if the line is magic, stop the loop
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

// authMiddleware is a middleware that checks the token in the request header
func authMiddleware(next http.Handler, tokens []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get the token from the header
		ok := isValid(r.Header.Get("Authorization"), tokens)
		if ok {
			next.ServeHTTP(w, r)
			return
		}

		// if we didn't find a token, or it's invalid, return an error
		w.WriteHeader(http.StatusUnauthorized)
	}
}

type errResponse struct {
	Error string `json:"error"`
}

// errorHttp returns an error as a JSON object
func errorHttp(w http.ResponseWriter, error string, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)

	_ = json.NewEncoder(w).Encode(errResponse{Error: error})
}
