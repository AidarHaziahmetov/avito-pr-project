package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aidar/avito-pr-project/internal/app"
	"github.com/aidar/avito-pr-project/internal/config"
)

func main() {
	// Загружаем конфигурацию из переменных окружения
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Не удалось загрузить конфигурацию: %v", err)
	}

	// Создаем экземпляр приложения
	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Не удалось создать приложение: %v", err)
	}

	// Инициализируем приложение (подключение к БД, настройка роутинга)
	ctx := context.Background()
	if err := application.Initialize(ctx); err != nil {
		log.Fatalf("Не удалось инициализировать приложение: %v", err)
	}

	// Настраиваем graceful shutdown для корректного завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Запускаем HTTP сервер в отдельной горутине
	go func() {
		if err := application.Run(); err != nil {
			log.Printf("Ошибка сервера: %v", err)
		}
	}()

	fmt.Printf("Сервер запущен на порту %s\n", cfg.Server.Port)
	fmt.Println("Нажмите Ctrl+C для остановки")

	// Ожидаем сигнал прерывания (Ctrl+C или SIGTERM)
	<-sigChan
	fmt.Println("\nОстановка сервера...")

	// Создаем контекст с таймаутом для graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// Корректно останавливаем приложение
	if err := application.Shutdown(shutdownCtx); err != nil {
		cancel()
		log.Printf("Не удалось корректно остановить сервер: %v", err)
		os.Exit(1)
	}
	cancel()

	fmt.Println("Сервер остановлен")
}
