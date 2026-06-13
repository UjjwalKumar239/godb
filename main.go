package main

import (
	"fmt"
	"godb/server"
)

func main() {
	fmt.Println("🚀 GoDB - Database Engine Starting...")
	server.Start(":8080")
}
