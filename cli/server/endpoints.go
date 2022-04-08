package main

import (
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
	"github.com/nxadm/tail"
	"github.com/nxadm/tail/ratelimiter"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

func handleWorkerFetch(db database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		job, err := db.FetchJob(r.Context())
		if err != nil {
			http.Error(w, "Error fetching jobs: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if job == nil {
			http.Error(w, "No job found", http.StatusNotFound)
			return
		}

		_ = json.NewEncoder(w).Encode(job)
	}
}

func handleWorkerLogs(db database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func handleNewJob(db database.Database) http.HandlerFunc {
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

func handleJobUpdate(db database.Database) http.HandlerFunc {
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
		if changes.Docker.Environment != nil {
			job.Docker.Environment = changes.Docker.Environment
		}
		if changes.Status != "" {
			job.Status = changes.Status
		}
		if changes.Metadata != nil {
			job.Metadata = changes.Metadata
		}

		if err := db.Update(r.Context(), job); err != nil {
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

		job, err := db.GetJobById(r.Context(), id)
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
					// TODO: extreure error
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
