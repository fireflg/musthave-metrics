package observer

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type AuditEntry struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

type Auditor struct {
	filePath string
	url      string
	client   *http.Client
}

func NewAuditor(filePath, url string) *Auditor {
	return &Auditor{
		filePath: filePath,
		url:      url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (a *Auditor) Notify(ctx context.Context, metricsID string) {
	a.NotifyBatch(ctx, []string{metricsID})
}

func (a *Auditor) NotifyBatch(ctx context.Context, metricsIDs []string) {
	var ip string
	if v := ctx.Value("client_ip"); v != nil {
		ip, _ = v.(string)
	}

	entry := AuditEntry{
		TS:        time.Now().Unix(),
		Metrics:   metricsIDs,
		IPAddress: ip,
	}

	if a.filePath != "" {
		a.writeToFile(entry)
	}
	if a.url != "" {
		a.sendToURL(entry)
	}
}

func (a *Auditor) writeToFile(entry AuditEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Error marshaling audit entry: %v", err)
		return
	}

	data = append(data, '\n')

	f, err := os.OpenFile(a.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening audit file: %v", err)
		return
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		log.Printf("Error writing to audit file: %v", err)
		return
	}

	log.Printf("Audit record written to file: %s", a.filePath)
}

func (a *Auditor) sendToURL(entry AuditEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Error marshaling audit entry: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, a.url, bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Error creating audit request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		log.Printf("Error sending audit to URL: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("Audit server returned error status: %d", resp.StatusCode)
		return
	}

	log.Printf("Audit record sent to URL: %s", a.url)
}
