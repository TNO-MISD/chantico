package main

import (
	"fmt"

	"example.com/m/internal/lifecycle"
)

func main() {
	app, err := lifecycle.New()
	if err != nil {
		panic(err)
	}
	err = app.Run()
	if err != nil {
		fmt.Println(err)
	}
}
