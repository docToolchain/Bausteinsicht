package main

import "testing"

// TestExportCmd_DefaultScaleIsOne is a regression test for the headless GPU crash bug.
//
// The original default was 2.0 (retina quality). In headless container environments,
// draw.io's GPU process is disabled via ELECTRON_DISABLE_GPU. When --scale 2 is
// passed, draw.io attempts GPU-accelerated rendering, the GPU process crashes with
// exit code 9, and the export silently fails (exit 0, no output file created).
//
// Scale=1.0 uses software rendering and works in all environments. Scale > 1 is
// now an opt-in that requires hardware GPU.
func TestExportCmd_DefaultScaleIsOne(t *testing.T) {
	cmd := newExportCmd()
	scale, err := cmd.Flags().GetFloat64("scale")
	if err != nil {
		t.Fatalf("getting scale flag: %v", err)
	}
	if scale != 1.0 {
		t.Errorf("default scale = %v, want 1.0 (scale > 1 requires GPU and fails in headless containers)", scale)
	}
}
