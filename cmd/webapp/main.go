package main

import (
	"chantico/internal/webapp"
	"fmt"
)

func main() {
	app, err := webapp.New()
	if err != nil {
		panic(err)
	}
	err = app.Run()
	if err != nil {
		fmt.Println(err)
	}
}
