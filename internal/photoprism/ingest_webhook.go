package photoprism

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"
)

// ingestWebhookPayload is sent to ingest-api on successful import.
type ingestWebhookPayload struct {
	PhotoUID     string `json:"photo_uid"`
	OriginalName string `json:"original_name"`
	FileHash     string `json:"file_hash"`
}

var (
	ingestClient     = &http.Client{Timeout: 10 * time.Second}
	ingestWebhookURL = os.Getenv("INGEST_WEBHOOK_URL")
	ingestSecret     = os.Getenv("INGEST_WEBHOOK_SECRET")
	ingestWG         sync.WaitGroup
)

// FireIngestWebhook posts an async HMAC-signed webhook to ingest-api.
// Called after a successful import. Failures are logged, not returned.
func FireIngestWebhook(photoUID, originalName, fileHash string) {
	if ingestWebhookURL == "" || ingestSecret == "" {
		return
	}
	if photoUID == "" {
		return
	}

	ingestWG.Add(1)
	go func() {
		defer ingestWG.Done()

		body, err := json.Marshal(ingestWebhookPayload{
			PhotoUID:     photoUID,
			OriginalName: originalName,
			FileHash:     fileHash,
		})
		if err != nil {
			log.Warnf("ingest-webhook: marshal: %s", err)
			return
		}

		mac := hmac.New(sha256.New, []byte(ingestSecret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))

		req, err := http.NewRequest("POST", ingestWebhookURL, bytes.NewReader(body))
		if err != nil {
			log.Warnf("ingest-webhook: new request: %s", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Hub-Signature-256", sig)

		resp, err := ingestClient.Do(req)
		if err != nil {
			log.Warnf("ingest-webhook: post %s: %s", ingestWebhookURL, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			log.Warnf("ingest-webhook: %s returned %d", ingestWebhookURL, resp.StatusCode)
			return
		}
		log.Debugf("ingest-webhook: posted photo_uid=%s", photoUID)
	}()
}
