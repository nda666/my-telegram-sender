package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage:")
		fmt.Println("  migrate create create_users")
		return
	}

	switch os.Args[1] {
	case "create":
		createMigration(os.Args[2])
	}
}

func createMigration(name string) {
	dir := "migrations"

	_ = os.MkdirAll(dir, 0755)

	version := strconv.FormatInt(time.Now().Unix(), 10)

	up := filepath.Join(dir, fmt.Sprintf("%s_%s.up.sql", version, name))
	down := filepath.Join(dir, fmt.Sprintf("%s_%s.down.sql", version, name))

	_ = os.WriteFile(up, []byte("-- Write UP migration here\n"), 0644)
	_ = os.WriteFile(down, []byte("-- Write DOWN migration here\n"), 0644)

	fmt.Println("created:")
	fmt.Println(up)
	fmt.Println(down)
}
