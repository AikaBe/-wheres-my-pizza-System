package main

import (
	"flag"
	"log/slog"
	"wheres-my-pizza/orderService/cmd"
)

func main() {
	mode := flag.String("mode", "", "Which mode to use")
	port := flag.String("port", "", "Port to listen on")
	maxConcurrent := flag.Int("max-concurrent", 50, "Maximum number of concurrent orders to process")
	flag.Parse()
	switch *mode {
	case "order-service":
		cmd.MainOrder(*port, *maxConcurrent)
	default:
		slog.Error("Unknown mode: %s", *mode)
	}

}
