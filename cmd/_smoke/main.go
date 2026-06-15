package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/anurag925/crank/internal/bootstrap"
	"github.com/anurag925/crank/internal/bootstrap/scaffold"

	_ "github.com/anurag925/crank/internal/bootstrap/features/base"
	_ "github.com/anurag925/crank/internal/bootstrap/features/postgres"
)

func main() {
	tmp, err := os.MkdirTemp("", "smoke")
	if err != nil {
		panic(err)
	}
	fmt.Println("TMP:", tmp)
	defer func() {
		if os.Getenv("KEEP_TMP") != "" {
			fmt.Println("keeping", tmp)
		} else {
			os.RemoveAll(tmp)
		}
	}()

	res, err := bootstrap.Generate(bootstrap.GlobalRegistry, bootstrap.Options{
		ProjectName: "demo",
		ModulePath:  "github.com/example/demo",
		TargetDir:   tmp,
		Features:    []string{"base", "postgres"},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Generated base project at", res.ProjectDir)

	rs, err := scaffold.Generate(scaffold.Options{
		ProjectDir: res.ProjectDir,
		Kind:       scaffold.KindHandler,
		Name:       "Order",
		Fields:     []string{"customer:string", "total:float"},
		Tests:      true,
	})
	if err != nil {
		fmt.Println("Error:", err)
		for _, c := range rs.Created {
			fmt.Println("  +", c)
		}
		os.Exit(1)
	}
	fmt.Println("Scaffold result:")
	for _, c := range rs.Created {
		fmt.Println("  +", c)
	}
	for _, s := range rs.Skipped {
		fmt.Println("  =", s)
	}
	fmt.Println("Wired:", rs.Wired, "Hint:", rs.WireHint)

	// Verify expected file paths exist
	expected := []string{
		"internal/domain/order/order.go",
		"internal/domain/order/events.go",
		"internal/domain/order/errors.go",
		"internal/domain/order/repository.go",
		"internal/application/order/commands.go",
		"internal/application/order/command_handler.go",
		"internal/application/order/queries.go",
		"internal/application/order/query_handler.go",
		"internal/adapters/persistence/postgres/order_repository.go",
		"internal/adapters/persistence/memory/order_repository.go",
		"internal/adapters/http/web/order_handler.go",
		"internal/adapters/http/web/order_handler_test.go",
	}
	for _, p := range expected {
		full := filepath.Join(res.ProjectDir, p)
		if _, err := os.Stat(full); err != nil {
			fmt.Printf("MISSING: %s\n", p)
		} else {
			fmt.Printf("OK: %s\n", p)
		}
	}

	// Now try to compile the generated project.
	// 1. Resolve module deps. The generated project has no go.sum; we
	//    skip go mod tidy (which needs network) and just check the
	//    generated go files compile syntactically by running gofmt.
	if err := runGofmt(res.ProjectDir); err != nil {
		fmt.Println("GOFMT FAILED:", err)
		os.Exit(1)
	}
	fmt.Println("GOFMT: OK")
}

func runGofmt(dir string) error {
	cmd := exec.Command("gofmt", "-l", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gofmt: %w: %s", err, string(out))
	}
	if len(out) > 0 {
		return fmt.Errorf("unformatted files: %s", string(out))
	}
	return nil
}
