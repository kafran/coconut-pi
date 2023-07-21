# Coconut Pi

A Raspberry Pi in Brazil + Go + HTMX + Tailwind.

![Web capture_21-7-2023_94948_coconutpi kafran codes](https://github.com/kafran/coconut-pi/assets/1889828/703981eb-87fe-4d46-8d3e-6eec205fb34e)

[Coconut Pi](https://coconutpi.kafran.codes) is my first Go project after getting a handle on the language's basics. This simple application was built to read a few metrics of interest from my Raspberry Pi 3 model B+, which I recently purchased.

This project is a result of my exploration with Go, following the [Tour of Go](https://go.dev/tour/) and solving a few [challenges](http://www.pythonchallenge.com/). The interactive parts of the front-end have been made possible using the HTMX Server-Sent Event (SSE) extension, and TailwindCSS has been used for styling.

In the initial stages, I implemented the solution naively, just reading the metrics and returning the events for the clients. However, after understanding the workings of net/http and how each client request would spawn its own goroutine reading the system metrics, I optimized the solution. This involved running the metrics on their own goroutine and notifying the clients using a basic pub/sub model.

Through this project, I've learnt about goroutines, the use of Mutex and RWMutex to protect the access to data structures from multiple goroutines, and improved my understanding of channels and pointers. I also discovered the nuances of variable scope and closures in Go [goroutine inside a for loop? 🥲], which was a challenging but enlightening experience.

The project is 24/7 running on my Raspberry Pi and available on https://coconutpi.kafran.codes (please don't hackme 🥲).

It's been a while since I've had so much fun learning a new language, and I look forward to continuing to explore Go through this project.

# Local Setup

To run the project you must have tailwind installed:

```bash
npx tailwindcss -i ./app/views/assets/css/src/style.css -o ./app/views/assets/css/dist/style.css --minify
```

and don't forget to configure the compiler for the Raspberry Pi:

```bash
GOOS=linux GOARCH=arm64 go build -o ./build/ ./app/main.go
```

# Future Plans

In the future, I aim to extend the capabilities of this project by adding new metrics. Specifically, I want to include the uptime of the Raspberry Pi, indicating how long it has been running since the last reboot. Additionally, I plan to implement a feature that keeps track of the maximum number of concurrent users on the dashboard since the last system reboot. This will provide useful insights into the usage patterns and capacity handling of the application.
