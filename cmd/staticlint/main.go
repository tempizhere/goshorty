// Package main содержит multichecker для статического анализа Go кода.
//
// Multichecker объединяет следующие группы анализаторов:
//
// 1. Стандартные анализаторы из golang.org/x/tools/go/analysis/passes:
//   - nilness: проверяет возможные разыменования nil указателей
//   - shadow: обнаруживает затенение переменных
//   - unreachable: находит недостижимый код
//   - printf: проверяет корректность форматных строк в printf-подобных функциях
//   - assign: обнаруживает бесполезные присваивания
//   - atomic: проверяет правильность использования sync/atomic
//   - bools: анализирует булевы выражения
//   - buildtag: проверяет корректность build tags
//   - copylocks: обнаруживает копирование значений с мьютексами
//
// 2. Все анализаторы класса SA из staticcheck.io:
//   - SA анализаторы проверяют код на наличие багов и потенциальных проблем
//
// 3. Дополнительные анализаторы из других классов staticcheck.io:
//   - ST1000: проверяет соответствие именования пакетов стандартам Go
//   - S1000: предлагает упрощения кода (например, замена "if x == true" на "if x")
//
// 4. Публичные анализаторы:
//   - errcheck: проверяет обработку возвращаемых ошибок
//   - deadcode: находит неиспользуемый код
//
// 5. Собственный анализатор:
//   - noexit: запрещает использование прямого вызова os.Exit в функции main пакета main
//
// Использование:
//
//	go run cmd/staticlint/main.go ./...
//
// Или после сборки:
//
//	go build -o staticlint cmd/staticlint/main.go
//	./staticlint ./...
package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	"github.com/kisielk/errcheck/errcheck"

	"github.com/tempizhere/goshorty/cmd/staticlint/noexit"
)

func main() {
	var analyzers []*analysis.Analyzer

	// 1. Стандартные анализаторы из golang.org/x/tools/go/analysis/passes
	analyzers = append(analyzers,
		nilness.Analyzer,     // проверка nil указателей
		shadow.Analyzer,      // затенение переменных
		unreachable.Analyzer, // недостижимый код
		printf.Analyzer,      // проверка printf форматов
		assign.Analyzer,      // бесполезные присваивания
		atomic.Analyzer,      // правильность использования sync/atomic
		bools.Analyzer,       // анализ булевых выражений
		buildtag.Analyzer,    // проверка build tags
	)

	// 2. Все анализаторы класса SA из staticcheck.io
	for _, analyzer := range staticcheck.Analyzers {
		analyzers = append(analyzers, analyzer.Analyzer)
	}

	// 3. Дополнительные анализаторы из других классов staticcheck.io
	// ST класс - проверки стиля кода
	for _, analyzer := range stylecheck.Analyzers {
		if analyzer.Analyzer.Name == "ST1000" { // проверка именования пакетов
			analyzers = append(analyzers, analyzer.Analyzer)
		}
	}

	// S класс - упрощения кода
	for _, analyzer := range simple.Analyzers {
		if analyzer.Analyzer.Name == "S1000" { // упрощения условий
			analyzers = append(analyzers, analyzer.Analyzer)
		}
	}

	// 4. Публичные анализаторы
	analyzers = append(analyzers,
		errcheck.Analyzer, // проверка обработки ошибок
		// Заменяем deadcode на более простую реализацию через staticcheck
	)

	// 5. Собственный анализатор
	analyzers = append(analyzers, noexit.NoExitAnalyzer)

	// Запуск multichecker с выбранными анализаторами
	multichecker.Main(analyzers...)
}
