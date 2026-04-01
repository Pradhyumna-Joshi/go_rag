package main

import "log"

func main() {

	config := Config{
		Addr: ":8080",
	}

	server := NewServer(config)
	if err := server.run(server.mount()); err != nil {
		log.Fatalf("Failed to run server : %v", err)
	}
}
