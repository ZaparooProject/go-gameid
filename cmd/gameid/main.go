package main

import "github.com/wizzomafizzo/go-gameid/pkg/database"

func main() {
	_, err := database.Load()
	if err != nil {
		panic(err)
	}
}
