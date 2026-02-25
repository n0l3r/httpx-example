// httpx-demo is a comprehensive demonstration of all httpx features.
//
// Run all examples:
//
//	go run main.go
//
// Run specific category:
//
//	go run main.go basic
//	go run main.go retry
//	go run main.go cache
//	go run main.go circuit-breaker
//	go run main.go rate-limiter
//	go run main.go middleware
//	go run main.go auth
//	go run main.go tracing
//	go run main.go singleflight
//	go run main.go mock
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/NTR3667/httpx-demo/examples/auth"
	"github.com/NTR3667/httpx-demo/examples/basic"
	"github.com/NTR3667/httpx-demo/examples/cache"
	cb "github.com/NTR3667/httpx-demo/examples/circuit_breaker"
	"github.com/NTR3667/httpx-demo/examples/middleware"
	mockdemo "github.com/NTR3667/httpx-demo/examples/mock_test"
	rl "github.com/NTR3667/httpx-demo/examples/rate_limiter"
	"github.com/NTR3667/httpx-demo/examples/retry"
	"github.com/NTR3667/httpx-demo/examples/singleflight"
	"github.com/NTR3667/httpx-demo/examples/tracing"
)

type demo struct {
	name string
	run  func()
}

var allDemos = []demo{
	{"basic", basic.Run},
	{"retry", retry.Run},
	{"cache", cache.Run},
	{"circuit-breaker", cb.Run},
	{"rate-limiter", rl.Run},
	{"middleware", middleware.Run},
	{"auth", auth.Run},
	{"tracing", tracing.Run},
	{"singleflight", singleflight.Run},
	{"mock", mockdemo.Run},
}

func main() {
	filter := ""
	if len(os.Args) > 1 {
		filter = strings.ToLower(os.Args[1])
	}

	if filter == "help" || filter == "--help" || filter == "-h" {
		printHelp()
		return
	}

	ran := 0
	for _, d := range allDemos {
		if filter == "" || d.name == filter {
			d.run()
			ran++
		}
	}

	if ran == 0 {
		fmt.Fprintf(os.Stderr, "Unknown demo: %q\n\n", filter)
		printHelp()
		os.Exit(1)
	}

	fmt.Println("\n✅ Done.")
}

func printHelp() {
	fmt.Println("httpx-demo — demonstrates all httpx features")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run main.go [category]")
	fmt.Println()
	fmt.Println("Categories:")
	for _, d := range allDemos {
		fmt.Printf("  %-20s\n", d.name)
	}
	fmt.Println()
	fmt.Println("Run all:  go run main.go")
}
