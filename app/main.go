package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/kafran/coconut-pi/app/views"
)

type Event struct {
	Type string
	Data string
}

type EventBroker struct {
	Publish     chan Event
	Subscribe   chan chan Event
	Unsubscribe chan chan Event
	subscribers map[chan Event]struct{} // in golang an empty struct takes up no memory
}

func (b *EventBroker) start() {
	for {
		select {
		case ch := <-b.Subscribe:
			b.subscribers[ch] = struct{}{}
			go publishSubscribersSize(b, len(b.subscribers))
			log.Println(ch, " subscribed.")
		case ch := <-b.Unsubscribe:
			delete(b.subscribers, ch)
			go publishSubscribersSize(b, len(b.subscribers))
			log.Println(ch, " unsubscribed.")
		case event := <-b.Publish:
			for ch := range b.subscribers {
				select {
				case ch <- event:
				default:
					// If the client is too slow to receive the event, we simply
					// skip it. Take note: the "observers" metric is recorded only once,
					// when someone subscribes or unsubscribes. Therefore, a slow client might
					// miss it. A possible solution to this problem could be using a
					// buffered channel.
					log.Println(ch, " skipped the event ", event)
				}
			}
		}
	}
}

func publishSubscribersSize(broker *EventBroker, size int) {
	var plural string
	if size > 1 {
		plural = "people"
	} else {
		plural = "person"
	}
	broker.Publish <- Event{"observers", fmt.Sprintf("%d %s here", size, plural)}
}

type PiMetric struct {
	Name         string
	Command      []string
	FormatOutput func(output string) (string, error)
}

func runPiMetric(metric PiMetric, broker *EventBroker) {
	for {
		output, err := exec.Command(metric.Command[0], metric.Command[1:]...).Output()
		if err != nil {
			broker.Publish <- Event{metric.Name, "-"}
			log.Println(Event{metric.Name, err.Error()})
			continue
		}
		formattedOutput, err := metric.FormatOutput(string(output))
		if err != nil {
			broker.Publish <- Event{metric.Name, "-"}
			log.Println(Event{metric.Name, err.Error()})
			continue
		}
		broker.Publish <- Event{metric.Name, formattedOutput}
		time.Sleep(time.Second)
	}
}

func streamHandler(broker *EventBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported!", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		streamChannel := make(chan Event)
		broker.Subscribe <- streamChannel
		// If for any reason the client connection is closed, we
		// unsubscribe from the broker's streamChannel.
		defer func() {
			broker.Unsubscribe <- streamChannel
		}()

		for {
			select {
			case event := <-streamChannel:
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Data)
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}
}

func main() {
	broker := &EventBroker{
		Subscribe:   make(chan chan Event),
		Unsubscribe: make(chan chan Event),
		Publish:     make(chan Event, 1),
		subscribers: make(map[chan Event]struct{}),
	}
	go broker.start()

	metrics := []PiMetric{
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

	for _, metric := range metrics {
		go runPiMetric(metric, broker)
	}

	http.Handle("/", http.FileServer(http.FS(views.Files)))
	http.HandleFunc("/status", streamHandler(broker))
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
