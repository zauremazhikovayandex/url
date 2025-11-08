// Command staticlint — кастомный multichecker проекта.
//
// Запуск:
//
//	go run ./cmd/staticlint [...packages]
//	go build -o staticlint ./cmd/staticlint && ./staticlint ./...
//
// Что входит:
//   - стандартные passes (printf, shadow, shift, и др.);
//   - ВСЕ SA* из staticcheck (SA1000…SA9999);
//   - не менее одного из прочих классов staticcheck: S1000 (simple) и ST1003 (stylecheck);
//   - публичные анализаторы: bodyclose и nilerr;
//   - собственный анализатор noosexit: запрещает os.Exit в main.main.
package main

import (
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"

	// стандартные passes — при желании расширь список
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/findcall"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"

	// staticcheck наборы (в этой версии — map[string]*analysis.Analyzer)
	"honnef.co/go/tools/simple"      // S*
	"honnef.co/go/tools/staticcheck" // SA*
	"honnef.co/go/tools/stylecheck"  // ST*

	"github.com/gostaticanalysis/nilerr"
	// публичные анализаторы
	bodyclose "github.com/timakin/bodyclose/passes/bodyclose"
)

func main() {
	var analyzers []*analysis.Analyzer

	// 1) стандартные passes
	analyzers = append(analyzers,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		deepequalerrors.Analyzer,
		defers.Analyzer,
		errorsas.Analyzer,
		fieldalignment.Analyzer,
		findcall.Analyzer,
		framepointer.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		sortslice.Analyzer,
		stdmethods.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
	)

	// 2) ВСЕ SA* (staticcheck.Analyzers — map[string]*analysis.Analyzer)
	for name, a := range staticcheck.Analyzers {
		if strings.HasPrefix(name, "SA") {
			analyzers = append(analyzers, a)
		}
	}

	// 3) не менее одного анализатора других классов staticcheck
	//    S1000 (упрощения) и ST1003 (стилистика аббревиатур)
	if a, ok := simple.Analyzers["S1000"]; ok {
		analyzers = append(analyzers, a)
	}
	if a, ok := stylecheck.Analyzers["ST1003"]; ok {
		analyzers = append(analyzers, a)
	}

	// 4) публичные анализаторы
	analyzers = append(analyzers,
		bodyclose.Analyzer,
		nilerr.Analyzer,
	)

	// 5) собственный — запрет os.Exit в main.main
	analyzers = append(analyzers, noOsExitAnalyzer)

	multichecker.Main(analyzers...)
}
