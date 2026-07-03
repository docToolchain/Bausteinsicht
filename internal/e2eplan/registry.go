// Package e2eplan maps E2E-Test-Plan.adoc line IDs (e.g. "5.1") to the Go
// e2e test(s) that automate them. It exists so scripts/e2e-test-report-gen
// can generate a line-accurate PASS/FAIL/SKIP report straight from `go test
// ./e2e/... -json`, instead of that report being hand-maintained (see #519).
//
// # Naming convention
//
// A subtest whose name is prefixed "<planID>_" — e.g.
// `t.Run("5.1_ExactMatch", ...)` — is detected automatically by the report
// generator via regex; it needs no entry here. This is the preferred style
// for all new e2e tests going forward, and is already used by the test
// files added in #519 (views_wildcards_lifting_test.go,
// security_injection_test.go, cli_flag_interactions_test.go, and others).
//
// Registry entries below are only for the exceptions: a top-level Test
// function (no subtest) covering one plan line, one subtest covering
// several plan lines, or a subtest name that predates this convention.
package e2eplan

// Registry maps a plan ID to the qualified go-test name(s) — in
// "TestFunc" or "TestFunc/Subtest" form, matching the "Test" field of `go
// test -json` events — that automate it. A plan ID mapped to more than one
// test name is considered PASS only if all of them passed.
var Registry = map[string][]string{
	// e2e/jsonc_comment_preservation_test.go — subtest names predate the
	// "<planID>_" convention and one subtest covers multiple plan lines.
	"4.16": {"TestJSONCCommentPreservation/ReverseSync"},
	"4.17": {"TestJSONCCommentPreservation/ReverseSync"},
	"4.18": {"TestJSONCCommentPreservation/ReverseSync"},
	"6.25": {"TestJSONCCommentPreservation/AddElement"},
	"6.26": {"TestJSONCCommentPreservation/AddElement"},
	"6.27": {"TestJSONCCommentPreservation/AddElement"},
	"6.28": {"TestJSONCCommentPreservation/AddElementMixedComments"},
	"6.29": {"TestJSONCCommentPreservation/TwoSequentialAdds"},
	"6.30": {"TestJSONCCommentPreservation/AddRelationship"},

	// e2e/label_html_handling_test.go — subtest names predate the
	// "<planID>_" convention.
	"4.4": {"TestReverseSyncLabelHandling/HTMLEntitiesDecoded"},
	"4.5": {"TestReverseSyncLabelHandling/HTMLTagsPreservedLiterally"},
	"4.6": {"TestReverseSyncLabelHandling/LineBreaksPreservedLiterally"},
	"4.7": {"TestReverseSyncLabelHandling/EmptyLabelRejected"},

	// e2e/views_wildcards_lifting_test.go — top-level Test functions (no
	// subtest, no planID in the name) for the cases that needed their own
	// isolated fixture (validation failures, multi-sync mutations).
	"5.13": {"TestViewScopeNonexistentElement"},
	"5.20": {"TestViewRemovedFromModel"},
	"5.21": {"TestViewRenamedKey"},
	"5.26": {"TestViewIncludeTrailingDot"},
	"5.27": {"TestViewIncludeJustDots"},
	"5.28": {"TestViewIncrementalElementAddedToAllViews"},
}
