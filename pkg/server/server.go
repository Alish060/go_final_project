package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"scheduler/pkg/api"
	"scheduler/pkg/db"
)

func StartServer() {
	if err := db.Init("./scheduler.db"); err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer db.Close()
	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}
	webDir := "./web"
	fs := http.FileServer(http.Dir(webDir))
	api.Init()
	http.Handle("/", fs)
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Сервер запущен на порту %s\n", port)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
