package main

import (
	"fmt"
	"os"

	"./data"
	
	"./logger"
	"./routes"
	"github.com/joho/godotenv"
)

var err error

func init() {
	//Начало работы, читаем настроечный фаил
	if err = godotenv.Load(); err != nil {
		fmt.Println("Can't load enc file - ", err.Error())
	}
}

func main() {
	//Загружаем модуль логирования
	if err = logger.Init(os.Getenv("logger_path")); err != nil {
		fmt.Println("Error opening logger subsystem ", err.Error())
		return
	}

	//Подключение к базе данных
	if err = data.ConnectDB(); err != nil {
		logger.Info.Println("Error open DB", err.Error())
		fmt.Println("Error open DB", err.Error())
		return
	}
	defer data.GetDB().Close() // не забывает закрыть подключение

	logger.Info.Println("Start work...")
	fmt.Println("Start work...")

	//раз в час обновляем данные регионов, и состояний
	go data.CacheDataUpdate()
	//----------------------------------------------------------------------

	//запуск сервера
	routes.StartServer()

	logger.Info.Println("Exit working...")
	fmt.Println("Exit working...")
}
