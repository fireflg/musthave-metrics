/*
staticlint — multichecker для статического анализа кода

Этот инструмент объединяет стандартные анализаторы Go, анализаторы из
staticcheck.io и кастомные анализаторы для комплексной проверки качества кода.

Стандартные анализаторы (golang.org/x/tools/go/analysis/passes)
 включают проверки на:
  - Согласованность assembly объявлений (asmdecl)
  - Проблемы с присваиваниями (assign)
  - Атомарные операции (atomic)
  - Выравнивание данных (atomicalign)
  - Булевые операции (bools)
  - Build tags (buildtag)
  - Cgo вызовы (cgocall)
  - Составные литералы (composite)
  - Блокировки копирования (copylock)
  - errors.As проверки (errorsas)
  - Frame pointers (framepointer)
  - HTTP ответы (httpresponse)
  - Утверждения интерфейсов (ifaceassert)
  - Замыкания в циклах (loopclosure)
  - Отмена контекста (lostcancel)
  - Nil функции (nilfunc)
  - Форматы printf (printf)
  - Сдвиги (shift)
  - Сигнальные каналы (sigchanyzer)
  - Стандартные методы (stdmethods)
  - Теги структур (structtag)
  - Тестирование goroutines (testinggoroutine)
  - Unmarshal операции (unmarshal)
  - Недостижимый код (unreachable)
  - unsafe.Pointer (unsafeptr)
  - Неиспользуемые результаты (unusedresult)

Анализаторы staticcheck.io (SA, ST, V)
 включают сотни дополнительных проверок для стиля, производительности
 и потенциальных ошибок.

Кастомные анализаторы проекта:
  - osexitanalyzer — запрещает os.Exit() в функции main
  - naminganalyzer — проверяет PascalCase для экспортируемых имен
  - contextanalyzer — проверяет наличие context.Context в функциях БД

Запуск анализа:
  go run ./cmd/staticlint ./...

Сборка:
  go build -o staticlint ./cmd/staticlint
  ./staticlint ./...
*/
package main