package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime/pprof"

	"github.com/lestrrat-go/helium"
)

const usage = `helium-flamegraph - Generate flamegraphs for helium XML operations

Usage:
  helium-flamegraph [options] <xml-file>

Options:
  -iterations int    Number of parsing iterations (default: 2000)
  -port int         HTTP server port (default: 8080)
  -profile string   Profile type: cpu, mem (default: cpu)
  -help             Show this help message

This command will:
1. Generate a profile by running XML operations
2. Start pprof web server to view the flamegraph

Examples:
  helium-flamegraph sample.xml                    # CPU flamegraph on port 8080
  helium-flamegraph -profile mem sample.xml      # Memory flamegraph  
  helium-flamegraph -port 9090 sample.xml        # Use different port
`

func main() {
	var (
		iterations = flag.Int("iterations", 2000, "Number of parsing iterations")
		port       = flag.Int("port", 8080, "HTTP server port")
		profile    = flag.String("profile", "cpu", "Profile type: cpu, mem")
		help       = flag.Bool("help", false, "Show help message")
	)

	flag.Parse()

	if *help {
		fmt.Print(usage)
		os.Exit(0)
	}

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Error: XML file argument required\n\n")
		fmt.Print(usage)
		os.Exit(1)
	}

	xmlFile := flag.Arg(0)

	if _, err := os.Stat(xmlFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: XML file does not exist: %s\n", xmlFile)
		os.Exit(1)
	}

	if *profile != "cpu" && *profile != "mem" {
		fmt.Fprintf(os.Stderr, "Error: profile must be 'cpu' or 'mem'\n")
		os.Exit(1)
	}

	if err := generateAndStartServer(xmlFile, *iterations, *port, *profile); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func generateAndStartServer(xmlFile string, iterations, port int, profileType string) error {
	fmt.Printf("🔥 Helium Flamegraph Generator\n")
	fmt.Printf("XML file: %s\n", xmlFile)
	fmt.Printf("Profile type: %s\n", profileType)
	fmt.Printf("Iterations: %d\n", iterations)
	fmt.Printf("Server port: %d\n\n", port)

	// Read XML file
	xmlData, err := os.ReadFile(xmlFile)
	if err != nil {
		return fmt.Errorf("failed to read XML file: %w", err)
	}

	profileFile := fmt.Sprintf("helium_%s.prof", profileType)

	// Generate profile
	fmt.Printf("📊 Generating %s profile...\n", profileType)
	if err := generateProfile(xmlData, iterations, profileType, profileFile); err != nil {
		return fmt.Errorf("failed to generate profile: %w", err)
	}

	fmt.Printf("✅ Profile generated: %s\n\n", profileFile)

	// Start pprof web server
	return startPprofServer(profileFile, port)
}

func generateProfile(xmlData []byte, iterations int, profileType, profileFile string) error {
	// Disable tracing for performance
	helium.SetTracingEnabled(false)
	ctx := context.Background()
	parser := helium.NewParser()

	switch profileType {
	case "cpu":
		return generateCPUProfile(ctx, parser, xmlData, iterations, profileFile)
	case "mem":
		return generateMemProfile(ctx, parser, xmlData, iterations, profileFile)
	default:
		return fmt.Errorf("unsupported profile type: %s", profileType)
	}
}

func generateCPUProfile(ctx context.Context, parser *helium.Parser, xmlData []byte, iterations int, profileFile string) error {
	f, err := os.Create(profileFile)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		return err
	}
	defer pprof.StopCPUProfile()

	// Run workload
	for i := range iterations {
		doc, err := parser.Parse(ctx, xmlData)
		if err != nil {
			return fmt.Errorf("parse failed at iteration %d: %w", i, err)
		}

		// Also test serialization to get complete picture
		if err := doc.XML(ctx, io.Discard); err != nil {
			return fmt.Errorf("serialization failed at iteration %d: %w", i, err)
		}
	}

	return nil
}

func generateMemProfile(ctx context.Context, parser *helium.Parser, xmlData []byte, iterations int, profileFile string) error {
	// Create many documents to trigger allocations
	var docs []*helium.Document
	for range iterations {
		doc, err := parser.Parse(ctx, xmlData)
		if err != nil {
			return fmt.Errorf("parse failed: %w", err)
		}
		docs = append(docs, doc)
	}

	// Write memory profile
	f, err := os.Create(profileFile)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := pprof.WriteHeapProfile(f); err != nil {
		return err
	}

	// Prevent optimization of docs slice
	_ = len(docs)
	return nil
}

func startPprofServer(profileFile string, port int) error {
	fmt.Printf("🌐 Starting pprof server on port %d...\n", port)
	fmt.Printf("📍 Open browser to: http://localhost:%d/ui/\n\n", port)

	// Start pprof server
	cmd := exec.Command("go", "tool", "pprof", "-http", fmt.Sprintf(":%d", port), profileFile)

	fmt.Printf("📋 Instructions:\n")
	fmt.Printf("   • The pprof web interface will start shortly\n")
	fmt.Printf("   • Open http://localhost:%d/ui/ in your browser\n", port)
	fmt.Printf("   • Click 'Flame Graph' for flamegraph view\n")
	fmt.Printf("   • Press Ctrl+C to stop the server when done\n\n")

	// Run server (blocks until interrupted)
	return cmd.Run()
}
