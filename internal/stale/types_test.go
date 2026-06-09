package stale

import (
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func TestMetaInt_Present(t *testing.T) {
	m := map[string]interface{}{"days": float64(30)}
	if got := metaInt(m, "days", 90); got != 30 {
		t.Errorf("metaInt = %d, want 30", got)
	}
}

func TestMetaInt_Missing(t *testing.T) {
	m := map[string]interface{}{}
	if got := metaInt(m, "days", 90); got != 90 {
		t.Errorf("metaInt missing = %d, want 90 (default)", got)
	}
}

func TestMetaInt_WrongType(t *testing.T) {
	m := map[string]interface{}{"days": "not-a-number"}
	if got := metaInt(m, "days", 90); got != 90 {
		t.Errorf("metaInt wrong type = %d, want 90 (default)", got)
	}
}

func TestMetaStringSlice_Present(t *testing.T) {
	m := map[string]interface{}{
		"kinds": []interface{}{"actor", "system"},
	}
	got := metaStringSlice(m, "kinds")
	if len(got) != 2 || got[0] != "actor" || got[1] != "system" {
		t.Errorf("metaStringSlice = %v, want [actor system]", got)
	}
}

func TestMetaStringSlice_Missing(t *testing.T) {
	m := map[string]interface{}{}
	if got := metaStringSlice(m, "kinds"); got != nil {
		t.Errorf("metaStringSlice missing = %v, want nil", got)
	}
}

func TestMetaStringSlice_WrongType(t *testing.T) {
	m := map[string]interface{}{"kinds": "not-a-slice"}
	if got := metaStringSlice(m, "kinds"); got != nil {
		t.Errorf("metaStringSlice wrong type = %v, want nil", got)
	}
}

func TestLoadConfigFromModel_Nil(t *testing.T) {
	cfg := LoadConfigFromModel(nil)
	if cfg.ThresholdDays != 90 {
		t.Errorf("nil model: ThresholdDays = %d, want 90", cfg.ThresholdDays)
	}
}

func TestLoadConfigFromModel_NoMeta(t *testing.T) {
	m := &model.BausteinsichtModel{Meta: nil}
	cfg := LoadConfigFromModel(m)
	if cfg.ThresholdDays != 90 {
		t.Errorf("no meta: ThresholdDays = %d, want 90", cfg.ThresholdDays)
	}
}

func TestLoadConfigFromModel_NoStaleDetection(t *testing.T) {
	m := &model.BausteinsichtModel{Meta: map[string]interface{}{"other": "value"}}
	cfg := LoadConfigFromModel(m)
	if cfg.ThresholdDays != 90 {
		t.Errorf("no staleDetection key: ThresholdDays = %d, want 90", cfg.ThresholdDays)
	}
}

func TestLoadConfigFromModel_WrongType(t *testing.T) {
	m := &model.BausteinsichtModel{
		Meta: map[string]interface{}{"staleDetection": "not-a-map"},
	}
	cfg := LoadConfigFromModel(m)
	if cfg.ThresholdDays != 90 {
		t.Errorf("wrong staleDetection type: ThresholdDays = %d, want 90", cfg.ThresholdDays)
	}
}

func TestLoadConfigFromModel_FullConfig(t *testing.T) {
	m := &model.BausteinsichtModel{
		Meta: map[string]interface{}{
			"staleDetection": map[string]interface{}{
				"thresholdDays": float64(60),
				"excludeKinds":  []interface{}{"actor"},
				"excludeTags":   []interface{}{"stable"},
			},
		},
	}
	cfg := LoadConfigFromModel(m)
	if cfg.ThresholdDays != 60 {
		t.Errorf("ThresholdDays = %d, want 60", cfg.ThresholdDays)
	}
	if len(cfg.ExcludeKinds) != 1 || cfg.ExcludeKinds[0] != "actor" {
		t.Errorf("ExcludeKinds = %v, want [actor]", cfg.ExcludeKinds)
	}
	if len(cfg.ExcludeTags) != 1 || cfg.ExcludeTags[0] != "stable" {
		t.Errorf("ExcludeTags = %v, want [stable]", cfg.ExcludeTags)
	}
}
