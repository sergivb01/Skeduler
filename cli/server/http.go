package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// func handleRequest(tasks chan<- JobRequest) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		var jobRequest JobRequest
// 		if err := json.NewDecoder(r.Body).Decode(&jobRequest); err != nil {
// 			http.Error(w, "error decoding json", http.StatusBadRequest)
// 			return
// 		}
//
// 		schedule(tasks, jobRequest)
// 		log.Printf("scheduled %+v task at %s\n", jobRequest, time.Now())
//
// 		_, _ = w.Write([]byte("OK"))
// 	}
// }

func startHttp(quit <-chan struct{}, conf HttpConfig) error {
	a := func() http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("OK"))
		}
	}
	r := mux.NewRouter()

	r.HandleFunc("/", a()).Methods("POST")

	srv := &http.Server{
		Addr:         conf.Listen,
		WriteTimeout: conf.WriteTimeout,
		ReadTimeout:  conf.ReadTimeout,
		IdleTimeout:  conf.IdleTimeout,
		Handler:      r,
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
		log.Printf("error running http server: %v\n", err)
	}

	<-idleConnsClosed

	return nil
}
