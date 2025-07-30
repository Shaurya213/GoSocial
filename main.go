package main

import (
	"fmt"
	"os"
	"github.com/joho/godotenv"
	"log"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Println(".env file not found, using system env variables")
	}
	fmt.Println(os.Getenv("MONGO_HOST"))
}
