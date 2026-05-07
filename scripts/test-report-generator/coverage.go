package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// parseCoverageFile reads a Go coverage.out file and extracts coverage per package
func parseCoverageFile(path string) (map[string]*CoverageInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open coverage file: %w", err)
	}
	defer file.Close()

	coverage := make(map[string]*CoverageInfo)
	scanner := bufio.NewScanner(file)

	// Skip mode line (first line is "mode: <mode>")
	if !scanner.Scan() {
		return coverage, nil
	}

	// Parse coverage lines: "path/to/package/file:startLine.startCol,endLine.endCol numStmt count"
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		// Extract package from file path
		filePath := parts[0]
		pkg := extractPackage(filePath)

		stmtCount, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}

		count, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}

		if coverage[pkg] == nil {
			coverage[pkg] = &CoverageInfo{
				Package: pkg,
			}
		}

		coverage[pkg].StmtTotal += stmtCount
		if count > 0 {
			coverage[pkg].StmtCovered += stmtCount
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan coverage file: %w", err)
	}

	// Calculate coverage percentages and identify low coverage
	for _, info := range coverage {
		if info.StmtTotal > 0 {
			info.Coverage = float64(info.StmtCovered) * 100 / float64(info.StmtTotal)
			info.IsLowCoverage = info.Coverage < 80
		}
	}

	return coverage, nil
}

// extractPackage extracts the Go package name from a file path
// Examples:
//   github.com/docToolchain/Bausteinsicht/internal/sync/forward.go -> github.com/docToolchain/Bausteinsicht/internal/sync
//   github.com/docToolchain/Bausteinsicht/cmd/bausteinsicht/root.go -> github.com/docToolchain/Bausteinsicht/cmd/bausteinsicht
func extractPackage(filePath string) string {
	if idx := strings.LastIndex(filePath, "/"); idx != -1 {
		return filePath[:idx]
	}
	return filePath
}
