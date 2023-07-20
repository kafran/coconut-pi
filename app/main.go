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

// The initial implementation was using a []slice to hold the subscribers,
// but a map seems more efficient for Unsubscribing latter. An empty struct
// in golang is 0 bytes, so it's a good choice for the map value.
// The sync.Mutex was replaced by a sync.RWMutex, as we don't need to block
// the whole map for reading when publishing.
type Publisher struct {
	Subscribers map[*Subscriber]struct{}
	Mu          sync.RWMutex
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
	p.Mu.RLock()
	defer p.Mu.RUnlock()
	for subscriber := range p.Subscribers {
		select {
		case subscriber.Events <- event:
		default:
			fmt.Println("Skipping event ", event)
		}
	}
}

func (p *Publisher) Size() int {
	p.Mu.RLock()
	defer p.Mu.RUnlock()
	return len(p.Subscribers)
}

type PiMetric struct {
	Name         string
	Command      []string
	FormatOutput func(output string) (string, error)
}

var piMetrics = []PiMetric{
	{
		Name:    "temp",
		Command: []string{"vcgencmd", "measure_temp"},
		FormatOutput: func(output string) (string, error) {
			var formattedOutput string
			formattedOutput = strings.TrimSuffix(output, "'C\n")
			formattedOutput = strings.TrimPrefix(formattedOutput, "temp=")
			formattedOutput = fmt.Sprintf("%s °C", formattedOutput)
			return formattedOutput, nil
		},
	},
	{
		Name:    "clock",
		Command: []string{"vcgencmd", "measure_clock", "arm"},
		FormatOutput: func(output string) (string, error) {
			var formattedOutput string
			formattedOutput = strings.TrimSuffix(output, "\n")
			formattedOutput = strings.Split(formattedOutput, "=")[1]
			hz, err := strconv.ParseFloat(formattedOutput, 64)
			if err != nil {
				return "-", err
			}
			ghz := hz / 1e9
			formattedOutput = fmt.Sprintf("%.2f GHz", ghz)
			return formattedOutput, nil
		},
	},
	{
		Name:    "volt",
		Command: []string{"vcgencmd", "measure_volts", "core"},
		FormatOutput: func(output string) (string, error) {
			var formattedOutput string
			formattedOutput = strings.TrimSuffix(output, "V\n")
			formattedOutput = strings.TrimPrefix(formattedOutput, "volt=")
			formattedOutput = fmt.Sprintf("%s V", formattedOutput)
			return formattedOutput, nil
		},
	},
	{
		Name:    "mem",
		Command: []string{"free", "-h"},
		FormatOutput: func(output string) (string, error) {
			var formattedOutput string
			lines := strings.Split(output, "\n")
			if len(lines) < 2 {
				return "-", fmt.Errorf("unexpected output format")
			}
			memInfo := strings.Fields(lines[1])
			if len(memInfo) < 3 {
				return "-", fmt.Errorf("unexpected output format")
			}
			formattedOutput = fmt.Sprintf("%s / %s", memInfo[2], memInfo[1])
			return formattedOutput, nil
		},
	},
}

func runPiMetric(metric PiMetric) {
	for {
		output, err := exec.Command(metric.Command[0], metric.Command[1:]...).Output()
		if err != nil {
			p.Publish(Event{metric.Name, "-"})
			fmt.Println(Event{metric.Name, err.Error()})
			continue
		}
		formattedOutput, err := metric.FormatOutput(string(output))
		if err != nil {
			p.Publish(Event{metric.Name, "-"})
			fmt.Println(Event{metric.Name, err.Error()})
			continue
		}
		p.Publish(Event{metric.Name, formattedOutput})
		time.Sleep(time.Second)
	}
}

func runObserversMetric() {
	for {
		var plural string
		observers := p.Size()
		if observers > 1 {
			plural = "people"
		} else {
			plural = "person"
		}
		p.Publish(Event{"observers", fmt.Sprintf("%d %s here", observers, plural)})
		time.Sleep(time.Second)
	}
}

func monitor() {
	for _, metric := range piMetrics {
		go runPiMetric(metric)
	}
	go runObserversMetric()
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
