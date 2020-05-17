package main

import (
	"context"
	"doubleboiler/models"
	"log"
)

func main() {
	user := models.User{}

	user.New(
		"admin@example.com",
		"notasecret",
	)

	if err := user.Save(context.Background()); err != nil {
		log.Fatal(err)
	}

	log.Println("User added")
}
