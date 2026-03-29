package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"coreblog/internal/blog"
	"coreblog/internal/server"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Загрузка .env файла (игнорируем ошибку, если файла нет) с перезаписью текущих переменных
	_ = godotenv.Load()

	// Строка подключения к БД (через переменную окружения или дефолт для локального Docker на Windows)
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://devlog_user:devlog_password@localhost:5432/devlog?sslmode=disable"
	}

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

	// Параметры OAuth из окружения
	vkClientID := os.Getenv("VK_CLIENT_ID")
	vkClientSecret := os.Getenv("VK_CLIENT_SECRET")
	vkRedirectURI := os.Getenv("VK_REDIRECT_URI")
	adminVKID := os.Getenv("ADMIN_VK_ID")

	yandexClientID := os.Getenv("YANDEX_CLIENT_ID")
	yandexClientSecret := os.Getenv("YANDEX_CLIENT_SECRET")
	yandexRedirectURI := os.Getenv("YANDEX_REDIRECT_URI")
	adminYandexID := os.Getenv("ADMIN_YANDEX_ID")

	srv := server.NewServer(store, vkClientID, vkClientSecret, vkRedirectURI, adminVKID, yandexClientID, yandexClientSecret, yandexRedirectURI, adminYandexID)

	// Настройка HTTP-сервера
	httpSrv := &http.Server{
		Addr:         ":8080",
		Handler:      srv.Router(),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// Если задан UNIX_SOCKET, используем его, иначе — стандартный TCP :8080
	sockPath := os.Getenv("UNIX_SOCKET")
	go func() {
		if sockPath != "" {
			fmt.Printf("Запуск на unix-сокете: %s\n", sockPath)
			os.Remove(sockPath)
			listener, err := net.Listen("unix", sockPath)
			if err != nil {
				log.Fatalf("Socket error: %v", err)
			}
			os.Chmod(sockPath, 0666)
			if err := httpSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Serve error: %v", err)
			}
		} else {
			fmt.Println("Запуск сервера на :8080 (TCP)")
			if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Serve error: %v", err)
			}
		}
	}()

	// Ожидание сигнала завершения от Docker (SIGTERM) или Ctrl+C (SIGINT)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("\nПолучен сигнал завершения. Плавная остановка сервера...")

	// Даем 5 секунд на завершение текущих HTTP-запросов
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Ошибка при остановке сервера: %v", err)
	}
	fmt.Println("Сервер успешно остановлен. Ресурсы освобождены.")
}
