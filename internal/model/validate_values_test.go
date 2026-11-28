package model

import (
	"strings"
	"testing"
)

func hasError(errs []ValidationError, path, contains string) bool {
	for _, e := range errs {
		if e.Path == path && strings.Contains(e.Message, contains) {
			return true
		}
	}
	return false
}

// #726: these values were accepted although other values of the same kind
// (view tags, cardinality, layout) are checked.
func TestValidate_ElementTagMustBeDefined(t *testing.T) {
	m := buildValidModel()
	m.Specification.Tags = []TagDefinition{{ID: "pii"}, {ID: "public"}}
	customer := m.Model["customer"]
	customer.Tags = []string{"pii", "undefined-tag"}
	m.Model["customer"] = customer

	errs := Validate(m)

	if len(errs) != 1 || !hasError(errs, "model.customer.tags", `tag "undefined-tag" is not defined in specification.tags`) {
		t.Errorf("errors: %v", errs)
	}
}

func TestValidate_ChildElementTagMustBeDefined(t *testing.T) {
	m := buildValidModel()
	shop := m.Model["shop"]
	api := shop.Children["api"]
	api.Tags = []string{"nope"}
	shop.Children["api"] = api
	m.Model["shop"] = shop

	if errs := Validate(m); !hasError(errs, "model.shop.api.tags", `"nope"`) {
		t.Errorf("errors: %v", errs)
	}
}

func TestValidate_LastModifiedMustBeRFC3339(t *testing.T) {
	m := buildValidModel()
	customer := m.Model["customer"]
	customer.LastModified = "yesterday"
	m.Model["customer"] = customer

	errs := Validate(m)
	if len(errs) != 1 || !hasError(errs, "model.customer.lastModified", `"yesterday" is not an RFC3339 timestamp`) {
		t.Errorf("errors: %v", errs)
	}

	customer.LastModified = "2020-01-01T00:00:00Z"
	m.Model["customer"] = customer
	if errs := Validate(m); len(errs) != 0 {
		t.Errorf("a valid timestamp was rejected: %v", errs)
	}
}

func TestValidate_StaleThresholdDays(t *testing.T) {
	for name, tc := range map[string]struct {
		value   interface{}
		invalid bool
	}{
		"negative":     {float64(-5), true},
		"text":         {"thirty", true},
		"fraction":     {2.5, true},
		"zero":         {float64(0), false},
		"normal value": {float64(30), false},
	} {
		t.Run(name, func(t *testing.T) {
			m := buildValidModel()
			m.Meta = map[string]interface{}{"staleDetection": map[string]interface{}{"thresholdDays": tc.value}}
			errs := Validate(m)
			if tc.invalid != hasError(errs, "meta.staleDetection.thresholdDays", "non-negative integer") {
				t.Errorf("invalid=%v, errors: %v", tc.invalid, errs)
			}
		})
	}
}

func TestValidate_UnknownDocLinkTypeIsAWarning(t *testing.T) {
	m := buildValidModel()
	customer := m.Model["customer"]
	customer.DocLinks = []DocLink{{Type: "adr", Href: "a.html"}, {Type: "weird", Href: "w.html"}}
	m.Model["customer"] = customer
	view := m.Views["main"]
	view.DocLinks = []DocLink{{Type: "odd", Href: "o.html"}}
	m.Views["main"] = view

	res := ValidateWithWarnings(m)

	if len(res.Errors) != 0 {
		t.Errorf("an unknown docLinks type is no error: %v", res.Errors)
	}
	want := map[string]bool{"model.customer.docLinks[1].type": false, "views.main.docLinks[0].type": false}
	for _, w := range res.Warnings {
		if _, ok := want[w.Path]; ok && strings.Contains(w.Message, "unknown docLinks type") {
			want[w.Path] = true
		}
	}
	for path, found := range want {
		if !found {
			t.Errorf("no warning at %s, warnings: %v", path, res.Warnings)
		}
	}
}
