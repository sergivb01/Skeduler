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
	"github.com/gorilla/mux"
	"github.com/hpcloud/tail"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

func handlePost(db database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var jobRequest jobs.Job
		if err := json.NewDecoder(r.Body).Decode(&jobRequest); err != nil {
			http.Error(w, "error decoding json", http.StatusBadRequest)
			return
		}

		if err := db.InsertJob(r.Context(), &jobRequest); err != nil {
			http.Error(w, fmt.Sprintf("error inserting job: %v\n", err), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(jobRequest)
	}
}

func handlePut(db database.Database) http.HandlerFunc {
	type updateBody struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Docker      struct {
			Environment map[string]interface{} `json:"environment"`
		} `json:"docker"`
		Status   jobs.JobStatus `json:"status"`
		Metadata interface{}    `json:"metadata"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := uuid.FromString(vars["id"])
		if err != nil {
			http.Error(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		job, err := db.GetJobById(r.Context(), id)
		if err != nil {
			http.Error(w, "job does not exist", http.StatusNotFound)
			return
		}

		if job == nil {
			http.Error(w, "Job with given ID not found", http.StatusNotFound)
			return
		}
		fmt.Printf("%+v\n", job)

		var changes updateBody
		if err := json.NewDecoder(r.Body).Decode(&changes); err != nil {
			http.Error(w, "Error decoding json: "+err.Error(), http.StatusBadRequest)
			return
		}

		if changes.Name != "" {
			job.Name = changes.Name
		}
		if changes.Description != "" {
			job.Description = changes.Description
		}
		if job.Docker.Environment != nil {
			job.Docker.Environment = changes.Docker.Environment
		}
		if job.Status != "" {
			job.Status = changes.Status
		}
		if job.Metadata != nil {
			job.Metadata = changes.Metadata
		}

		if err := db.Update(r.Context(), job); err != nil {
			http.Error(w, "Error updating job: "+err.Error(), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(job)
	}
}

func handleGet(db database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := uuid.FromString(vars["id"])
		if err != nil {
			http.Error(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		job, err := db.GetJobById(r.Context(), id)
		if err != nil {
			http.Error(w, "job does not exist", http.StatusNotFound)
			return
		}

		if job == nil {
			http.Error(w, "Job with given ID not found", http.StatusNotFound)
			return
		}
		fmt.Printf("%+v\n", job)

		_ = json.NewEncoder(w).Encode(job)
	}
}

func handleLogs() http.HandlerFunc {
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

func handleLogsTail() http.HandlerFunc {
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
			ReOpen:    false,
			MustExist: true,
			Follow:    true,
		})
		if err != nil {
			http.Error(w, "Error tailing: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")

		tick := time.NewTicker(time.Millisecond * 500)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				flusher.Flush()
				break
			case line := <-t.Lines:
				if err := line.Err; err != nil {
					http.Error(w, "error tailing: "+err.Error(), http.StatusInternalServerError)
					return
				}

				text := line.Text
				if strings.Contains(text, jobs.MagicEnd) {
					text = strings.TrimSuffix(text, jobs.MagicEnd)
					_, _ = w.Write([]byte(fmt.Sprintf("%s\n", text)))
					return
				}
				_, _ = w.Write([]byte(fmt.Sprintf("%s\n", text)))
				break
			}
		}
	}
}

func startHttp(quit <-chan struct{}, conf HttpConfig, db database.Database, finished chan<- struct{}) error {
	r := mux.NewRouter()

	r.HandleFunc("/", handlePost(db)).Methods("POST")
	r.HandleFunc("/{id}", handleGet(db)).Methods("GET")
	r.HandleFunc("/{id}", handlePut(db)).Methods("PUT")
	r.HandleFunc("/logs/{id}", handleLogs()).Methods("GET")
	r.HandleFunc("/logs/{id}/tail", handleLogsTail()).Methods("GET")

	srv := &http.Server{
		Addr: conf.Listen,
		// deshabilitat per poder fer streaming de logs
		// WriteTimeout: conf.WriteTimeout,
		ReadTimeout: conf.ReadTimeout,
		IdleTimeout: conf.IdleTimeout,
		Handler:     r,
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
