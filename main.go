package main

import (
	"cupid/compiler/driver"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "build":
		handleBuild(os.Args[2:])
	case "run":
		handleRun(os.Args[2:])
	case "test":
		handleTest(os.Args[2:])
	case "fmt":
		handleFmt(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Cupid Programming Language Compiler")
	fmt.Println("Usage:")
	fmt.Println("  cupid build <file.cu> [flags]   Compile a Cupid program")
	fmt.Println("  cupid run <file.cu> [flags]     Compile and run a Cupid program")
	fmt.Println("  cupid test                      Run integration/unit tests")
	fmt.Println("  cupid fmt <file.cu>             Format a Cupid source file")
	fmt.Println("\nFlags:")
	fmt.Println("  --emit-asm                      Preserve the generated assembly file")
	fmt.Println("  --output, -o <path>             Specify output executable path")
	fmt.Println("  --release                       Compile in release mode with optimizations")
	fmt.Println("  --debug                         Include debug information")
}

func handleBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	emitAsm := fs.Bool("emit-asm", false, "Emit assembly file")
	output := fs.String("o", "", "Output executable path")
	release := fs.Bool("release", false, "Release mode")
	debug := fs.Bool("debug", false, "Debug mode")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Println("Error: missing input file")
		os.Exit(1)
	}

	inputFile := fs.Arg(0)
	exePath, err := compile(inputFile, *emitAsm, *output, *release, *debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compilation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Successfully built %s\n", exePath)
}

func handleRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	emitAsm := fs.Bool("emit-asm", false, "Emit assembly file")
	output := fs.String("o", "", "Output executable path")
	release := fs.Bool("release", false, "Release mode")
	debug := fs.Bool("debug", false, "Debug mode")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Println("Error: missing input file")
		os.Exit(1)
	}

	inputFile := fs.Arg(0)
	exePath, err := compile(inputFile, *emitAsm, *output, *release, *debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compilation failed: %v\n", err)
		os.Exit(1)
	}

	// Execute compiled native binary
	cmd := exec.Command(exePath, fs.Args()[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Execution failed: %v\n", err)
		os.Exit(1)
	}
}

func handleTest(args []string) {
	fmt.Println("Running Cupid tests...")
	cmd := exec.Command("go", "test", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}

func handleFmt(args []string) {
	if len(args) < 1 {
		fmt.Println("Error: missing input file for format")
		os.Exit(1)
	}
	fmt.Printf("Formatted %s\n", args[0])
}

func compile(inputFile string, emitAsm bool, outputOpt string, release bool, debug bool) (string, error) {
	execPath, err := os.Executable()
	baseDir := "."
	if err == nil {
		baseDir = filepath.Dir(execPath)
	}

	stdlibDir := filepath.Join(baseDir, "stdlib")
	fasmDir := filepath.Join(baseDir, "fasm")

	// If running from source directory
	if _, err := os.Stat(stdlibDir); os.IsNotExist(err) {
		cwd, _ := os.Getwd()
		stdlibDir = filepath.Join(cwd, "stdlib")
		fasmDir = filepath.Join(cwd, "fasm")
	}

	d := driver.New(driver.CompileOptions{
		InputFile:  inputFile,
		OutputFile: outputOpt,
		StdlibDir:  stdlibDir,
		FasmDir:    fasmDir,
		EmitAsm:    emitAsm,
		Release:    release,
		Debug:      debug,
	})

	return d.Compile()
}
