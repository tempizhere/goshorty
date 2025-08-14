// Package noexit содержит анализатор, запрещающий использование прямого вызова os.Exit в функции main пакета main.
package noexit

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// NoExitAnalyzer представляет анализатор, который проверяет отсутствие прямых вызовов os.Exit в функции main пакета main.
var NoExitAnalyzer = &analysis.Analyzer{
	Name: "noexit",
	Doc:  "запрещает использование прямого вызова os.Exit в функции main пакета main",
	Run:  run,
}

// run выполняет анализ AST для поиска вызовов os.Exit в функции main пакета main.
func run(pass *analysis.Pass) (interface{}, error) {
	// Проверяем только файлы нашего проекта, исключаем зависимости
	if !strings.HasPrefix(pass.Fset.Position(pass.Files[0].Pos()).Filename, pass.Pkg.Path()) {
		return nil, nil
	}

	// Проходим по всем файлам в пакете
	for _, file := range pass.Files {
		// Проверяем, что это пакет main
		if file.Name.Name != "main" {
			continue
		}

		// Проходим по всем объявлениям в файле
		ast.Inspect(file, func(node ast.Node) bool {
			// Ищем объявления функций
			funcDecl, ok := node.(*ast.FuncDecl)
			if !ok {
				return true
			}

			// Проверяем, что это функция main
			if funcDecl.Name.Name != "main" {
				return true
			}

			// Проверяем тело функции main на наличие вызовов os.Exit
			ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
				callExpr, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				// Проверяем, является ли вызов селектором (например, os.Exit)
				selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				// Проверяем, что селектор вызывает Exit
				if selExpr.Sel.Name != "Exit" {
					return true
				}

				// Проверяем, что это вызов из пакета os
				if ident, ok := selExpr.X.(*ast.Ident); ok {
					if obj := pass.TypesInfo.Uses[ident]; obj != nil {
						if pkg, ok := obj.(*types.PkgName); ok {
							if pkg.Imported().Path() == "os" {
								pass.Reportf(callExpr.Pos(), "прямой вызов os.Exit в функции main запрещен")
							}
						}
					}
				}

				return true
			})

			return true
		})
	}

	return nil, nil
}
