package control

import "time"

type Request struct {
	Op string `json:"op"`
}

type Status struct {
	Running     bool         `json:"running"`
	UptimeSec   float64      `json:"uptime_sec"`
	Transcripts []Transcript `json:"transcripts"`
}

type SimpleResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type Transcript struct {
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}
