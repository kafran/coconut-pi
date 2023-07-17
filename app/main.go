package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

func piHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		cmd := exec.Command("vcgencmd", "measure_temp")
		output, err := cmd.Output()
		if err != nil {
			fmt.Fprintf(w, "data: Error: %v\n\n", err)
			flusher.Flush()
			time.Sleep(time.Second)
			continue
		}

		temp := strings.TrimSuffix(string(output), "'C\n")
		temp = strings.TrimPrefix(temp, "temp=")

		fmt.Fprintf(w, "data: Current temperature: %s°C\n\n", temp)
		flusher.Flush()
		time.Sleep(time.Second)
	}
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("./app/views/")))
	http.HandleFunc("/temp", piHandler)
	http.ListenAndServe(":8080", nil)
}
