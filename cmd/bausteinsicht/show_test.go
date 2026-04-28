package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestShowCmd_TextOutput(t *testing.T) {
	modelPath := writeFindModel(t)
	var buf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"show", "--model", modelPath, "payment-service"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "payment-service") {
		t.Errorf("expected element ID in output:\n%s", out)
	}
	if !strings.Contains(out, "Payment Service") {
		t.Errorf("expected title in output:\n%s", out)
	}
	if !strings.Contains(out, "service") {
		t.Errorf("expected kind in output:\n%s", out)
	}
}

func TestShowCmd_JSONOutput(t *testing.T) {
	modelPath := writeFindModel(t)
	var buf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"show", "--model", modelPath, "--format", "json", "payment-service"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\noutput:\n%s", err, buf.String())
	}
	if out["id"] != "payment-service" {
		t.Errorf("expected id=payment-service, got %v", out["id"])
	}
	if out["kind"] != "service" {
		t.Errorf("expected kind=service, got %v", out["kind"])
	}
}

func TestShowCmd_ShowsRelationships(t *testing.T) {
	modelPath := writeFindModel(t)
	var buf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"show", "--model", modelPath, "payment-service"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	// payment-service has an incoming relationship from order-service
	if !strings.Contains(out, "order-service") {
		t.Errorf("expected order-service in relationships:\n%s", out)
	}
}

func TestShowCmd_ShowsViews(t *testing.T) {
	modelPath := writeFindModel(t)
	var buf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"show", "--model", modelPath, "payment-service"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "context") || !strings.Contains(out, "payment") {
		t.Errorf("expected views listed in output:\n%s", out)
	}
}

func TestShowCmd_ElementNotFound(t *testing.T) {
	modelPath := writeFindModel(t)
	root := NewRootCmd()
	root.SetArgs([]string{"show", "--model", modelPath, "nonexistent-element"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error for unknown element ID")
	}
}

func TestShowCmd_JSONHasRelationshipsAndViews(t *testing.T) {
	modelPath := writeFindModel(t)
	var buf bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"show", "--model", modelPath, "--format", "json", "order-service"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\noutput:\n%s", err, buf.String())
	}
	rels, ok := out["relationships"].([]interface{})
	if !ok || len(rels) == 0 {
		t.Errorf("expected non-empty relationships array, got: %v", out["relationships"])
	}
	views, ok := out["views"].([]interface{})
	if !ok || len(views) == 0 {
		t.Errorf("expected non-empty views array, got: %v", out["views"])
	}
}
