package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/h2non/filetype"
)

var EgamiServerHost = orDefault(os.Getenv("EGAMI_SERVER_HOST"), "127.0.0.1")

var EgamiServerPort = orDefault(os.Getenv("EGAMI_SERVER_PORT"), "3333")

var EgamiDataDir = orDefault(os.Getenv("EGAMI_DATA_DIRECTORY"), "/var/lib/egami/data")

var EgamiUserToken = os.Getenv("EGAMI_USER_TOKEN")

type uploadResponse struct {
	Uploads []upload `json:"uploads"`
}

type upload struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
}

func main() {
	if EgamiUserToken == "" {
		panic("EGAMI_USER_TOKEN environment variable is not set")
	}

	router := mux.NewRouter()

	router.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Server", "Egami")
			fmt.Printf("%s %s\n", r.Method, r.URL.Path)

			h.ServeHTTP(w, r)
		})
	})

	router.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := mux.CurrentRoute(r).GetName()
			if route == "fs" || route == "hi" || route == "health" {
				h.ServeHTTP(w, r)
				return
			}

			if r.Header.Get("Authorization") != "Bearer "+EgamiUserToken {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			h.ServeHTTP(w, r)
		})
	})

	// upload route
	router.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("content-type"))
		if err != nil || mediaType != "multipart/form-data" {
			http.Error(w, "Invalid content-type", http.StatusBadRequest)
			return
		}

		mp := multipart.NewReader(r.Body, params["boundary"])

		images := []upload{}
		for {
			part, err := mp.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				http.Error(w, "Error reading multipart data", http.StatusInternalServerError)
				return
			} else if part.FileName() != "file" {
				continue
			}

			var body io.Reader = part

			fileExt := filepath.Ext(part.FileName())
			if fileExt == "" {
				matchBuf := make([]byte, 8192) // 8KB buffer
				if _, err := part.Read(matchBuf); err != nil && err != io.EOF {
					http.Error(w, "Error reading file data", http.StatusInternalServerError)
					return
				}

				if kind, _ := filetype.Match(matchBuf); kind != filetype.Unknown {
					fileExt = "." + kind.Extension
				}

				body = io.MultiReader(bytes.NewReader(matchBuf), part)
			}

			id := strings.ReplaceAll(uuid.NewString(), "-", "")[6 : 6+8]

			fileName := id + fileExt

			f, err := os.OpenFile(
				filepath.Join(EgamiDataDir, fileName),
				os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
				os.ModePerm,
			)
			if err != nil {
				http.Error(w, "Error saving file", http.StatusInternalServerError)
				return
			}
			defer f.Close()

			if _, err = io.Copy(f, body); err != nil {
				http.Error(w, "Error saving file", http.StatusInternalServerError)
				return
			}

			images = append(images, upload{
				ID:       id,
				Filename: fileName,
			})
		}

		resp, err := json.Marshal(&uploadResponse{
			Uploads: images,
		})

		if len(images) == 0 {
			http.Error(w, "No files uploaded", http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(w, "Error preparing response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	}).Methods(http.MethodPost)

	router.NewRoute().Name("health").Path("/health").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK\n"))
	})

	router.NewRoute().Name("hi").Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Egami is running\n"))
	})

	// File server route
	router.NewRoute().
		Name("fs").
		Methods(http.MethodGet).
		PathPrefix("/").
		Handler(http.FileServer(justFilesFilesystem{fs: http.Dir(EgamiDataDir)}))

	addr := EgamiServerHost + ":" + EgamiServerPort
	fmt.Printf("listening on %s\n", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		panic(err)
	}
}

func orDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
