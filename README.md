# Coconut Pi

A Raspberry Pi + Golang + SSE (Server Sent Events) + HTMX + Tailwind.

![Coconut Pi Dashboard](https://github.com/kafran/coconut-pi/assets/1889828/703981eb-87fe-4d46-8d3e-6eec205fb34e)

[Coconut Pi](https://coconutpi.kafran.codes) is a compact SSE (Server Sent Events) showcase built with Golang. This simple application is designed to read and display metrics from a Raspberry Pi 3 Model B+ in real time.

The interactive front-end is powered by the HTMX SSE extension, with TailwindCSS for styling. The solution implements a publish-subscribe model using goroutines and channels in Golang.

The entire application is embedded into a single binary for easy deployment.

This project is currently hosted on my own Raspberry Pi and is available 24/7 at [coconutpi.kafran.codes](https://coconutpi.kafran.codes) (please don't hackme 🥲).

## Local Setup

To run the project, make sure you have Golang, Node.js and npm installed on your machine.

Compile the Tailwind CSS using the following command:

```bash
npx tailwindcss -i ./app/views/assets/css/src/style.css -o ./app/views/assets/css/dist/style.css --minify
```

Configure the Go compiler for the Raspberry Pi with the command:

```bash
GOOS=linux GOARCH=arm64 go build -o ./build/ ./app/main.go
```

And deploy to your Raspberry Pi:

```bash
scp -i '~/.ssh/id_ed25519' ./build/main <username>@<ip-address>:<path on the raspberry>
```

## Future Plans

* Add an uptime metric 
* Add the number of max concurrent users on the dashboard since the last system reboot
* Add the ability for users to leave a message on the page that will persist until the next reboot
