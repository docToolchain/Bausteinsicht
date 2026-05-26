package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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

// CoverageBlock represents one instrumented block from coverage.out
type CoverageBlock struct {
	StartLine int `json:"start_line"`
	StartCol  int `json:"start_col"`
	EndLine   int `json:"end_line"`
	EndCol    int `json:"end_col"`
	NumStmt   int `json:"num_stmt"`
	Count     int `json:"count"`
}

// FileCoverage holds per-file coverage: block list + aggregated stats
type FileCoverage struct {
	ImportPath  string           `json:"import_path"`
	LocalPath   string           `json:"local_path"`
	StmtTotal   int              `json:"stmt_total"`
	StmtCovered int              `json:"stmt_covered"`
	Coverage    float64          `json:"coverage"`
	Blocks      []CoverageBlock  `json:"blocks"`
	SourceLines []string         `json:"source_lines,omitempty"`
}

// CoverageDetails holds file-level coverage information
type CoverageDetails struct {
	Files      map[string]*FileCoverage `json:"files"`
	ModuleName string                   `json:"module_name"`
}

// parseCoverageFileDetailed parses coverage.out and returns both package and file-level coverage
func parseCoverageFileDetailed(path string, sourceRoot string) (map[string]*CoverageInfo, *CoverageDetails, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open coverage file: %w", err)
	}
	defer file.Close()

	pkgCoverage := make(map[string]*CoverageInfo)
	fileCoverage := make(map[string]*FileCoverage)
	scanner := bufio.NewScanner(file)

	// Skip mode line
	if !scanner.Scan() {
		return pkgCoverage, &CoverageDetails{Files: fileCoverage}, nil
	}

	// Read module name once
	moduleName, _ := readModuleName(sourceRoot)

	// Parse coverage blocks
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		// Parse "path/file.go:startLine.col,endLine.col"
		locParts := strings.Split(parts[0], ":")
		if len(locParts) != 2 {
			continue
		}
		importPath := locParts[0]
		rangeParts := strings.Split(locParts[1], ",")
		if len(rangeParts) != 2 {
			continue
		}

		startParts := strings.Split(rangeParts[0], ".")
		endParts := strings.Split(rangeParts[1], ".")
		if len(startParts) != 2 || len(endParts) != 2 {
			continue
		}

		startLine, _ := strconv.Atoi(startParts[0])
		startCol, _ := strconv.Atoi(startParts[1])
		endLine, _ := strconv.Atoi(endParts[0])
		endCol, _ := strconv.Atoi(endParts[1])

		stmtCount, _ := strconv.Atoi(parts[1])
		count, _ := strconv.Atoi(parts[2])

		// Aggregate by package
		pkg := extractPackage(importPath)
		if pkgCoverage[pkg] == nil {
			pkgCoverage[pkg] = &CoverageInfo{Package: pkg}
		}
		pkgCoverage[pkg].StmtTotal += stmtCount
		if count > 0 {
			pkgCoverage[pkg].StmtCovered += stmtCount
		}

		// Store file-level block
		if fileCoverage[importPath] == nil {
			localPath := resolveModulePath(importPath, sourceRoot, moduleName)
			fileCoverage[importPath] = &FileCoverage{
				ImportPath: importPath,
				LocalPath:  localPath,
			}
		}
		fc := fileCoverage[importPath]
		fc.StmtTotal += stmtCount
		if count > 0 {
			fc.StmtCovered += stmtCount
		}
		fc.Blocks = append(fc.Blocks, CoverageBlock{
			StartLine: startLine,
			StartCol:  startCol,
			EndLine:   endLine,
			EndCol:    endCol,
			NumStmt:   stmtCount,
			Count:     count,
		})
	}

	// Read source lines for each file and embed in FileCoverage
	for _, fc := range fileCoverage {
		if fc.LocalPath != "" {
			if sourceLines := readSourceLines(fc.LocalPath); sourceLines != nil {
				fc.SourceLines = sourceLines
			}
		}
	}

	// Calculate percentages
	for _, info := range pkgCoverage {
		if info.StmtTotal > 0 {
			info.Coverage = float64(info.StmtCovered) * 100 / float64(info.StmtTotal)
			info.IsLowCoverage = info.Coverage < 80
		}
	}
	for _, fc := range fileCoverage {
		if fc.StmtTotal > 0 {
			fc.Coverage = float64(fc.StmtCovered) * 100 / float64(fc.StmtTotal)
		}
	}

	return pkgCoverage, &CoverageDetails{Files: fileCoverage, ModuleName: moduleName}, nil
}

// readModuleName reads the module name from go.mod
func readModuleName(searchDir string) (string, error) {
	dir := searchDir
	for {
		gomodPath := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(gomodPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "module ") {
					return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
				}
			}
		}
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			break
		}
		dir = parentDir
	}
	return "", fmt.Errorf("go.mod not found")
}

// resolveModulePath converts an import path to a filesystem path
func resolveModulePath(importPath string, searchDir string, moduleName string) string {
	if moduleName == "" {
		return ""
	}
	if !strings.HasPrefix(importPath, moduleName) {
		return ""
	}
	relPath := strings.TrimPrefix(importPath, moduleName+"/")
	// Convert forward slashes (from coverage.out) to OS-specific separators
	relPath = filepath.FromSlash(relPath)
	return filepath.Join(searchDir, relPath)
}

// readSourceLines reads a source file and returns lines (1-indexed, index 0 = "")
func readSourceLines(localPath string) []string {
	if localPath == "" {
		return nil
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil
	}
	lines := []string{""} // index 0 empty for 1-based indexing
	for _, line := range strings.Split(string(data), "\n") {
		lines = append(lines, line)
	}
	return lines
}

// htmlEscapeCode escapes HTML special chars in source code
func htmlEscapeCode(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// buildLineCoverageMap builds a map of line -> coverage status ("covered"/"uncovered"/"")
func buildLineCoverageMap(blocks []CoverageBlock, totalLines int) []string {
	status := make([]string, totalLines+1) // 1-indexed
	for _, block := range blocks {
		for lineNum := block.StartLine; lineNum <= block.EndLine; lineNum++ {
			if lineNum < len(status) {
				if block.Count > 0 {
					status[lineNum] = "covered"
				} else if status[lineNum] != "covered" {
					status[lineNum] = "uncovered"
				}
			}
		}
	}
	return status
}
