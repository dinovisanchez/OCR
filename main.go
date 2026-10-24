package main

import (
	"ia-extraction/router"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")

	router.NewRouter()
}