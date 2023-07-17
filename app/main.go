package main

import (
	"net/http"
	"os/exec"
)

func piHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("vcgencmd", "measure_temp")
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, "Failed to execute command: " + err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Current temperature: " + string(output) + "°C"))
}

func main() {
	http.HandleFunc("/pi", piHandler)
	http.ListenAndServe(":8080", nil)
}
