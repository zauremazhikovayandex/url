package main

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// noOsExitAnalyzer запрещает прямые вызовы os.Exit(...) в функции main пакета main.
var noOsExitAnalyzer = &analysis.Analyzer{
	Name: "noosexit",
	Doc: `forbid direct calls to os.Exit in main.main

Прямые вызовы os.Exit в main() затрудняют корректное завершение приложения,
нарушают инварианты тестирования и обходят defer. Вместо этого верните
ненулевой код возврата из main (или используйте обработку ошибок/логирование).

Срабатывает только для:
  - пакета "main"
  - функции "main"
  - вызова os.Exit(...) непосредственно в теле этой функции.
`,
	Run: runNoOsExit,
}

// Запуск анализатора
func runNoOsExit(pass *analysis.Pass) (interface{}, error) {
	// интересуемся только пакетами main
	if pass.Pkg == nil || pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, f := range pass.Files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Name.Name != "main" || fn.Body == nil {
				continue
			}
			// Обходим тело main() и ищем вызовы os.Exit
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				// Ищем вид: os.Exit(...)
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel == nil || sel.Sel.Name != "Exit" {
					return true
				}
				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				// Убедимся, что это именно пакет "os"
				if obj, ok := pass.TypesInfo.Uses[pkgIdent].(*types.PkgName); ok {
					if obj.Imported().Path() == "os" {
						pass.Reportf(call.Pos(), "do not call os.Exit in main; return an error/exit code instead")
					}
				}
				return true
			})
		}
	}
	return nil, nil
}
