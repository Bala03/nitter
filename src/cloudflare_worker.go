package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/cloudflare/cloudflare-go"
	"github.com/gorilla/mux"
)

type CloudflareWorker struct {
	api *cloudflare.API
}

func NewCloudflareWorker(apiKey, email string) (*CloudflareWorker, error) {
	api, err := cloudflare.New(apiKey, email)
	if err != nil {
		return nil, err
	}
	return &CloudflareWorker{api: api}, nil
}

func (cw *CloudflareWorker) DeployWorker(scriptName, scriptContent string) error {
	zoneID, err := cw.api.ZoneIDByName(os.Getenv("CLOUDFLARE_ZONE"))
	if err != nil {
		return err
	}

	_, err = cw.api.UploadWorker(zoneID, scriptName, scriptContent, true)
	return err
}

func createCloudflareWorkerRouter(r *mux.Router, cw *CloudflareWorker) {
	r.HandleFunc("/deploy", func(w http.ResponseWriter, r *http.Request) {
		scriptName := r.URL.Query().Get("scriptName")
		scriptContent := r.URL.Query().Get("scriptContent")

		if scriptName == "" || scriptContent == "" {
			http.Error(w, "Missing scriptName or scriptContent", http.StatusBadRequest)
			return
		}

		err := cw.DeployWorker(scriptName, scriptContent)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to deploy worker: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Worker deployed successfully"))
	}).Methods("POST")
}

func main() {
	apiKey := os.Getenv("CLOUDFLARE_API_KEY")
	email := os.Getenv("CLOUDFLARE_EMAIL")

	cw, err := NewCloudflareWorker(apiKey, email)
	if err != nil {
		fmt.Printf("Failed to create Cloudflare worker: %v\n", err)
		return
	}

	r := mux.NewRouter()
	createCloudflareWorkerRouter(r, cw)

	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}
