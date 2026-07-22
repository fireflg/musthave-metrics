// Package main реализует точку входа для staticlint.
//
// Использование:
//
//	staticlint [flags] [packages]
//
// Флаги:
//
//	-c	печатать строки кода в дополнение к ошибкам
//	-d	выводить диагностику вместо проверки ошибок (для отладки)
//	-e	выводить только ошибки
//	-j	выводить результаты в формате JSON
//	-l	отключить вывод ошибок
//	-w	перезаписывать файлы результатами анализа
//
// Примеры:
//
//	# Анализ всех пакетов в текущем проекте
//	staticlint ./...
//
//	# Анализ конкретного пакета
//	staticlint ./internal/app
//
//	# Анализ с JSON выводом
//	staticlint -j ./...
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(Analyzers()...)
}