package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/lestrrat-go/helium"
)

const usage = `helium-flame - Generate and view flamegraphs instantly

Usage:
  helium-flame [options] <xml-file>

Options:
  -iterations int    Number of parsing iterations (default: 2000)
  -port int         HTTP server port (default: 8080)
  -profile string   Profile type: cpu, mem (default: cpu)
  -help             Show this help message

This command will:
1. Generate a profile by running XML operations
2. Automatically open your browser to view the flamegraph
3. Keep the server running until you press Ctrl+C

Examples:
  helium-flame sample.xml                    # CPU flamegraph on port 8080
  helium-flame -profile mem sample.xml      # Memory flamegraph  
  helium-flame -port 9090 sample.xml        # Use different port
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

	if err := generateAndViewProfile(xmlFile, *iterations, *port, *profile); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func generateAndViewProfile(xmlFile string, iterations, port int, profileType string) error {
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

	// Check and install go-torch for real flamegraphs
	if err := ensureFlamegraphTools(); err != nil {
		fmt.Printf("⚠️  Warning: %v\n", err)
		fmt.Printf("   Falling back to pprof web interface (call graphs)\n\n")
		return startPprofServer(profileFile, port)
	}

	// Generate real flamegraph
	fmt.Printf("🔥 Generating actual flamegraph...\n")
	svgFile, err := generateSVGFlamegraph(profileFile, profileType)
	if err != nil {
		fmt.Printf("⚠️  Failed to generate flamegraph: %v\n", err)
		fmt.Printf("   Falling back to pprof web interface\n\n")
		return startPprofServer(profileFile, port)
	}

	// Serve the SVG flamegraph over HTTP
	fmt.Printf("✅ Flamegraph generated: %s\n", svgFile)
	fmt.Printf("🌐 Starting HTTP server to serve flamegraph...\n")
	
	return serveSVGFlamegraph(svgFile, port)
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

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch {
	case commandExists("xdg-open"): // Linux
		cmd = exec.Command("xdg-open", url)
	case commandExists("open"): // macOS
		cmd = exec.Command("open", url)
	case commandExists("cmd"): // Windows
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("no suitable browser opener found")
	}

	return cmd.Start()
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func ensureFlamegraphTools() error {
	// Install go-torch if not available
	if !commandExists("go-torch") {
		fmt.Printf("📦 Installing go-torch for real flamegraphs...\n")
		
		cmd := exec.Command("go", "install", "github.com/uber/go-torch@latest")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to install go-torch: %v\nOutput: %s", err, output)
		}
		
		if !commandExists("go-torch") {
			return fmt.Errorf("go-torch installation failed")
		}
		fmt.Printf("✅ go-torch installed successfully!\n")
	}

	// Check if flamegraph.pl is available
	if !commandExists("flamegraph.pl") && !fileExists("flamegraph.pl") {
		fmt.Printf("📦 Installing FlameGraph scripts...\n")
		
		if err := installFlameGraphScripts(); err != nil {
			return fmt.Errorf("failed to install FlameGraph scripts: %v", err)
		}
		
		fmt.Printf("✅ FlameGraph scripts installed successfully!\n")
	}

	return nil
}

func installFlameGraphScripts() error {
	// Clone the FlameGraph repository
	cmd := exec.Command("git", "clone", "https://github.com/brendangregg/FlameGraph.git", "flamegraph-tmp")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone FlameGraph repo: %v\nOutput: %s", err, output)
	}
	
	// Copy the scripts to current directory
	scripts := []string{"flamegraph.pl", "stackcollapse-perf.pl", "stackcollapse.pl"}
	for _, script := range scripts {
		src := filepath.Join("flamegraph-tmp", script)
		if fileExists(src) {
			if err := copyFile(src, script); err != nil {
				return fmt.Errorf("failed to copy %s: %v", script, err)
			}
			// Make executable
			if err := os.Chmod(script, 0755); err != nil {
				return fmt.Errorf("failed to make %s executable: %v", script, err)
			}
		}
	}
	
	// Clean up
	os.RemoveAll("flamegraph-tmp")
	
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func generateSVGFlamegraph(profileFile, profileType string) (string, error) {
	svgFile := fmt.Sprintf("flamegraph_%s.svg", profileType)
	
	var cmd *exec.Cmd
	if profileType == "cpu" {
		cmd = exec.Command("go-torch", "-b", profileFile, "-f", svgFile)
	} else {
		// For memory profiles, use different options
		cmd = exec.Command("go-torch", "--alloc_space", "-b", profileFile, "-f", svgFile)
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go-torch failed: %v\nOutput: %s", err, output)
	}
	
	return svgFile, nil
}

func serveSVGFlamegraph(svgFile string, port int) error {
	// Setup HTTP handler to serve the SVG
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Serve the SVG file
		w.Header().Set("Content-Type", "image/svg+xml")
		http.ServeFile(w, r, svgFile)
	})

	// Setup a simple index page
	http.HandleFunc("/index", func(w http.ResponseWriter, r *http.Request) {
		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Helium Flamegraph</title>
    <style>
        body { 
            font-family: Arial, sans-serif; 
            margin: 0; 
            padding: 20px; 
            background: #f5f5f5;
        }
        .container { 
            max-width: 100%%; 
            background: white; 
            border-radius: 8px; 
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        h1 { 
            background: #4CAF50; 
            color: white; 
            margin: 0; 
            padding: 20px; 
            text-align: center; 
        }
        .flamegraph { 
            width: 100%%; 
            height: calc(100vh - 120px); 
            border: none; 
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔥 Helium XML Library - Flamegraph</h1>
        <embed src="/" class="flamegraph" type="image/svg+xml">
    </div>
</body>
</html>`)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	})

	addr := fmt.Sprintf(":%d", port)
	url := fmt.Sprintf("http://localhost:%d/index", port)
	
	fmt.Printf("🚀 Opening flamegraph in browser...\n")
	fmt.Printf("📍 Server running at: http://localhost:%d/\n\n", port)

	// Start server in a goroutine
	server := &http.Server{Addr: addr}
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	// Wait a moment for server to start
	time.Sleep(1 * time.Second)

	// Open browser
	if err := openBrowser(url); err != nil {
		fmt.Printf("⚠️  Could not open browser automatically. Please open: %s\n", url)
	} else {
		fmt.Printf("✨ Browser opened! The flamegraph should appear shortly.\n")
	}

	fmt.Printf("\n📋 Instructions:\n")
	fmt.Printf("   • The flamegraph is now displayed in your browser\n")
	fmt.Printf("   • Direct SVG: http://localhost:%d/\n", port)
	fmt.Printf("   • With HTML wrapper: http://localhost:%d/index\n", port)
	fmt.Printf("   • SVG file saved as: %s\n", svgFile)
	fmt.Printf("   • Press Ctrl+C to stop the server\n\n")

	fmt.Printf("Press Ctrl+C to exit...\n")
	
	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	
	fmt.Printf("\n👋 Shutting down server...\n")
	
	// Shutdown server gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Server shutdown error: %v\n", err)
	}
	
	return nil
}

func startPprofServer(profileFile string, port int) error {
	fmt.Printf("🌐 Starting pprof server on port %d...\n", port)
	fmt.Printf("🚀 Opening browser automatically...\n\n")

	url := fmt.Sprintf("http://localhost:%d/ui/", port)
	
	// Start pprof server in background
	cmd := exec.Command("go", "tool", "pprof", "-http", fmt.Sprintf(":%d", port), profileFile)
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start pprof server: %w", err)
	}

	// Wait a moment for server to start
	time.Sleep(2 * time.Second)

	// Open browser
	if err := openBrowser(url); err != nil {
		fmt.Printf("⚠️  Could not open browser automatically. Please open: %s\n", url)
	} else {
		fmt.Printf("✨ Browser opened! The interface should appear shortly.\n")
	}

	fmt.Printf("\n📋 Instructions:\n")
	fmt.Printf("   • The pprof web interface is now loading in your browser\n")
	fmt.Printf("   • If it doesn't open, manually visit: %s\n", url)
	fmt.Printf("   • Press Ctrl+C to stop the server when done\n\n")

	// Wait for the server process
	return cmd.Wait()
}