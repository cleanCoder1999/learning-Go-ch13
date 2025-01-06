package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {

	var wg sync.WaitGroup
	wg.Add(4)
	// ### - exercise 1.1: write a small web server that returns the current time in RFC 3339 when a GET request is sent
	{
		// simple http server
		s := http.Server{
			Addr:         ":8080",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 90 * time.Second,
			IdleTimeout:  120 * time.Second,
			Handler:      GetHandler{},
		}

		go func() {
			err := s.ListenAndServe()
			wg.Done()
			if err != nil {
				// ErrServerClosed is a Sentinel error indicating that the server shut down which is not an error per-se
				if !errors.Is(err, http.ErrServerClosed) {
					panic(err)
				}
			}
		}()
	}

	// ### - exercise 1.2: (ServeMux) write a small web server that returns the current time in RFC 3339 when a GET request is sent
	{
		mux := http.NewServeMux()

		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

			accept := r.Header.Get("Accept")
			fmt.Println(accept)

			now := time.Now().UTC()
			response := now.Format(time.RFC3339)

			if strings.ToUpper(accept) == "JSON" {
				response = buildJson(now)
			}

			w.WriteHeader(http.StatusOK)
			n, err := w.Write([]byte(response))
			if err != nil {
				slog.Error(err.Error())
				return
			}

			fmt.Printf("%d bytes written\n", n)
		})

		// mux allows for path variables
		mux.HandleFunc("/hello/{name}", func(w http.ResponseWriter, r *http.Request) {
			name := r.PathValue("name")

			n, err := w.Write([]byte("Hello " + name))
			if err != nil {
				slog.Error(err.Error())
				return
			}

			fmt.Printf("%d bytes written\n", n)
		})

		s := http.Server{
			Addr:         ":8081",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 90 * time.Second,
			IdleTimeout:  120 * time.Second,
			Handler:      mux, // uses a mux as request handler
		}

		go func() {
			err := s.ListenAndServe()
			wg.Done()
			if err != nil {
				// ErrServerClosed is a Sentinel error indicating that the server shut down which is not an error per-se
				if !errors.Is(err, http.ErrServerClosed) {
					panic(err)
				}
			}
		}()
	}

	// ### - exercise 2: write a small middleware component that uses JSON structured logging to log the IP address of each ingress request
	{
		// this can be done by using ...
		// (1) Go's standard lib support
		// (2) alice (3rd party lib)
		// (3) gorilla mux (3rd party lib)
		// (4) chi (3rd party lib)
		// (5) Gin (web framework)
		// (4) Echo (web framework)
		//
		// for the exercise I use (1)
		mux := http.NewServeMux()

		mux.Handle("/log",

			// chains middleware function IpAddressLogger
			IpAddressLogger(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						n, err := w.Write([]byte("logged IP"))
						if err != nil {
							slog.Error(err.Error())
						}

						fmt.Printf("%d bytes written\n", n)
					},
				),
			),
		)

		s := http.Server{
			Addr:         ":8083",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 90 * time.Second,
			IdleTimeout:  120 * time.Second,
			Handler:      mux,
		}

		go func() {
			err := s.ListenAndServe()
			wg.Done()
			if err != nil {
				// ErrServerClosed is a Sentinel error indicating that the server shut down which is not an error per-se
				if !errors.Is(err, http.ErrServerClosed) {
					panic(err)
				}
			}
		}()
	}

	wg.Wait()
}

func buildJson(now time.Time) string {
	customTime := struct {
		DayOfWeek  string `json:"day_of_week"`
		DayOfMonth int    `json:"day_of_month"`
		Month      string `json:"month"`
		Year       int    `json:"year"`
		Hour       int    `json:"hour"`
		Minute     int    `json:"minute"`
		Second     int    `json:"second"`
	}{
		DayOfWeek:  now.Weekday().String(),
		DayOfMonth: now.Day(),
		Month:      now.Month().String(),
		Year:       now.Year(),
		Hour:       now.Hour(),
		Minute:     now.Minute(),
		Second:     now.Second(),
	}

	out, _ := json.Marshal(customTime)
	return string(out)
}

func IpAddressLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// create a structured logging instance
		options := &slog.HandlerOptions{Level: slog.LevelInfo}
		handler := slog.NewJSONHandler(os.Stdout, options)
		mySlog := slog.New(handler)

		// LogAttrs() provides higher performance than using "alternating keys"
		mySlog.Info("slower logging with alternating keys", "ip:", r.RemoteAddr)
		mySlog.LogAttrs(r.Context(), slog.LevelInfo, "faster logging with LogAttrs()", slog.String("ip:", r.RemoteAddr))

		h.ServeHTTP(w, r)
	})
}

type GetHandler struct{}

func (gh GetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.WriteHeader(http.StatusOK)
	n, err := w.Write([]byte(time.Now().Format(time.RFC3339)))
	if err != nil {
		slog.Error(err.Error())
		return
	}

	fmt.Printf("%d bytes written\n", n)
}
