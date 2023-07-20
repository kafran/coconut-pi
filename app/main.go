package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kafran/coconut-pi/app/views"
)

type Event struct {
	Type string
	Data string
}

type Subscriber struct {
	Events chan Event
}

type Publisher struct {
	Subscribers map[*Subscriber]struct{}
	Mu          sync.Mutex
}

func (p *Publisher) Subscribe() *Subscriber {
	p.Mu.Lock()
	defer p.Mu.Unlock()
	sub := &Subscriber{
		Events: make(chan Event),
	}
	p.Subscribers[sub] = struct{}{}
	defer fmt.Println("Subscriber subscribed ", sub)
	return sub
}

func (p *Publisher) Unsubscribe(sub *Subscriber) {
	p.Mu.Lock()
	defer p.Mu.Unlock()
	if _, ok := p.Subscribers[sub]; !ok {
		panic("Subscriber not found.")
	}
	delete(p.Subscribers, sub)
	close(sub.Events)
	defer fmt.Println("Subscriber unsubscribed ", sub)
}

func (p *Publisher) Publish(event Event) {
	p.Mu.Lock()
	defer p.Mu.Unlock()
	for subscriber := range p.Subscribers {
		subscriber.Events <- event
	}
}

func monitor() {
	for {
		for _, event := range []string{"temp", "clock", "volt"} {
			var cmd *exec.Cmd
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
				p.Publish(Event{event, fmt.Sprintf("Error: %v", err)})
				continue
			}

			var formattedOutput string
			switch event {
			case "temp":
				formattedOutput = strings.TrimSuffix(string(output), "'C\n")
				formattedOutput = strings.TrimPrefix(formattedOutput, "temp=")
				formattedOutput = fmt.Sprintf("%s °C", formattedOutput)
			case "clock":
				formattedOutput = strings.TrimSuffix(string(output), "\n")
				formattedOutput = strings.Split(formattedOutput, "=")[1]
				hz, err := strconv.ParseFloat(formattedOutput, 64)
				if err != nil {
					p.Publish(Event{event, fmt.Sprintf("Error: %v", err)})
					continue
				}
				ghz := hz / 1e9
				formattedOutput = fmt.Sprintf("%.2f GHz", ghz)
			case "volt":
				formattedOutput = strings.TrimSuffix(string(output), "V\n")
				formattedOutput = strings.TrimPrefix(formattedOutput, "volt=")
				formattedOutput = fmt.Sprintf("%s V", formattedOutput)
			}
			p.Publish(Event{event, formattedOutput})
			fmt.Printf("Event: %v\n", Event{event, formattedOutput})
		}
		var plural string
		if len(p.Subscribers) > 1 {
			plural = "people"
		} else {
			plural = "person"
		}
		p.Publish(Event{"observers", fmt.Sprintf("%d %s here", len(p.Subscribers), plural)})
		time.Sleep(time.Second)
	}
}

func piHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sub := p.Subscribe()
	defer p.Unsubscribe(sub)

	for {
		select {
		case event := <-sub.Events:
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Data)
			flusher.Flush()
		case <-r.Context().Done():
			return
			// default:
			// drop the event
		}
	}
}

var p = &Publisher{
	Subscribers: make(map[*Subscriber]struct{}),
}

func main() {
	http.Handle("/", http.FileServer(http.FS(views.Files)))
	http.HandleFunc("/status", piHandler)
	go monitor()
	http.ListenAndServe(":8080", nil)
}
