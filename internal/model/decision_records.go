package model

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var driveLetter = regexp.MustCompile(`^[A-Za-z]:`)

// decisionFileLocation says how the file of a decision is to be read: a URL is
// not a file of the project, an absolute path or one that leaves the project is
// outside it; otherwise rel is the cleaned path below the model's directory.
func decisionFileLocation(file string) (rel string, isURL, outside bool) {
	if strings.Contains(file, "://") {
		return "", true, false
	}
	slashed := strings.ReplaceAll(file, `\`, "/")
	if strings.HasPrefix(slashed, "/") || driveLetter.MatchString(slashed) {
		return "", false, true
	}
	rel = path.Clean(slashed)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false, true
	}
	return rel, false, false
}

// DecisionFileExists reports whether the file of a decision exists below baseDir;
// checked is false if that cannot be said (no base directory, a URL, a path
// outside the project).
func DecisionFileExists(baseDir, file string) (exists, checked bool) {
	if baseDir == "" || file == "" {
		return false, false
	}
	rel, isURL, outside := decisionFileLocation(file)
	if isURL || outside {
		return false, false
	}
	_, err := os.Stat(filepath.Join(baseDir, filepath.FromSlash(rel)))
	return err == nil, true
}

// validateDecisionRecords checks the fields of the decisions in the specification:
// a status that is not known and a date that is not YYYY-MM-DD are warnings, a file
// that does not exist is a warning too, and a file outside the project is an error
// (BR-045 to BR-048, #707).
func validateDecisionRecords(m *BausteinsichtModel) ([]ValidationError, []ValidationWarning) {
	var errs []ValidationError
	var warns []ValidationWarning
	for i, d := range m.Specification.Decisions {
		where := fmt.Sprintf("specification.decisions[%d] (%s)", i, d.ID)
		if d.Status != "" && !isKnownADRStatus(d.Status) {
			warns = append(warns, ValidationWarning{Path: where, Message: fmt.Sprintf(
				"status %q is not one of %s", d.Status, joinStatuses(KnownADRStatuses))})
		}
		if d.Date != "" {
			if _, err := time.Parse("2006-01-02", d.Date); err != nil {
				warns = append(warns, ValidationWarning{Path: where, Message: fmt.Sprintf(
					"date %q is not a date (YYYY-MM-DD)", d.Date)})
			}
		}
		if d.FilePath == "" {
			continue
		}
		if _, _, outside := decisionFileLocation(d.FilePath); outside {
			errs = append(errs, ValidationError{Path: where, Message: fmt.Sprintf(
				"file %q is outside the project: use a path below the model's directory", d.FilePath)})
			continue
		}
		if exists, checked := DecisionFileExists(m.BaseDir, d.FilePath); checked && !exists {
			warns = append(warns, ValidationWarning{Path: where, Message: fmt.Sprintf(
				"file %q does not exist", d.FilePath)})
		}
	}
	return errs, warns
}

func isKnownADRStatus(s ADRStatus) bool {
	for _, k := range KnownADRStatuses {
		if s == k {
			return true
		}
	}
	return false
}

func joinStatuses(ss []ADRStatus) string {
	parts := make([]string, len(ss))
	for i, s := range ss {
		parts[i] = string(s)
	}
	return strings.Join(parts, ", ")
}
