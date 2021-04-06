package main

import (
	"fmt"
	"github.com/solarmicrobe/vesync-go"
	"os"
)

func main() {
	username := os.Getenv("USERNAME")
	password := os.Getenv("FOO")

	manager := vesync.NewVeSync(username, password, "America/Chicago")
	manager.Login()
	fmt.Printf("Token: %s", manager.Token)
}
