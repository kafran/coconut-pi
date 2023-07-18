package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/kafran/coconut-pi/app/views"
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
		var cmd *exec.Cmd
		for _, event := range []string{"temp", "clock", "volt"} {
			switch event {
			case "temp":
				cmd = exec.Command("vcgencmd", "measure_temp")
			case "clock":
				cmd = exec.Command("vcgencmd", "measure_clock", "arm")
			case "volt":
				cmd = exec.Command("vcgencmd", "measure_volts", "core")
			}

			output, err := cmd.Output()
			if err != nil {
				fmt.Fprintf(w, "event: %s\ndata: Error: %v\n\n", event, err)
				flusher.Flush()
				time.Sleep(time.Second)
				continue
			}

			var formattedOutput string
			switch event {
			case "temp":
				formattedOutput = strings.TrimSuffix(string(output), "'C\n")
				formattedOutput = strings.TrimPrefix(formattedOutput, "temp=")
				formattedOutput = fmt.Sprintf("%s °C", formattedOutput)
			case "clock":
				formattedOutput = strings.Split(string(output), "=")[1]
				hz, err := strconv.ParseFloat(formattedOutput, 64)
				if err != nil {
					fmt.Fprintf(w, "event: %s\ndata: Error: %v\n\n", event, err)
					flusher.Flush()
					time.Sleep(time.Second)
					continue
				}
				ghz := hz / 1e9
				formattedOutput = fmt.Sprintf("%.2f GHz", ghz)
			case "volt":
				formattedOutput = strings.TrimSuffix(string(output), "V\n")
				formattedOutput = strings.TrimPrefix(formattedOutput, "volt=")
				formattedOutput = fmt.Sprintf("%s V", formattedOutput)
			}

			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, formattedOutput)
			flusher.Flush()
			time.Sleep(time.Second)
		}
	}
}

func main() {
	http.Handle("/", http.FileServer(http.FS(views.Files)))
	http.HandleFunc("/status", piHandler)
	http.ListenAndServe(":8080", nil)
}
