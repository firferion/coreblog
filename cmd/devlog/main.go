package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"coreblog/internal/blog"
	"coreblog/internal/server"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Строка подключения к БД
	connStr := "postgres://devlog_user:devlog_password@localhost:5432/devlog_db"

	// Парсинг конфигурации пула
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatalf("Ошибка парсинга конфига БД: %v", err)
	}

	// Настройка параметров пула
	config.MaxConns = 10
	config.MinConns = 2

	// Создание пула соединений
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer pool.Close()

	// Проверка подключения
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("БД недоступна: %v", err)
	}

	fmt.Println("Успешное подключение к базе данных")

	// Инициализация слоев
	store := blog.NewStore(pool)
	srv := server.NewServer(store)

	// Настройка HTTP-сервера
	httpSrv := &http.Server{
		Addr:         ":8080",
		Handler:      srv.Router(),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	fmt.Println("Сервер запускается на :8080")
	if err := httpSrv.ListenAndServe(); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
