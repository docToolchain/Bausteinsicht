package diagram

import (
	"strings"
	"testing"

	"github.com/docToolchain/Bausteinsicht/internal/model"
)

func testDynamicView() model.DynamicView {
	return model.DynamicView{
		Key:   "checkout-flow",
		Title: "Checkout Flow",
		Steps: []model.SequenceStep{
			{Index: 1, From: "web-frontend", To: "api-gateway", Label: "POST /orders", Type: model.StepSync},
			{Index: 2, From: "api-gateway", To: "order-service", Label: "createOrder()", Type: model.StepSync},
			{Index: 3, From: "order-service", To: "payment-service", Label: "charge()", Type: model.StepSync},
			{Index: 4, From: "payment-service", To: "order-service", Label: "chargeResult", Type: model.StepReturn},
			{Index: 5, From: "order-service", To: "message-broker", Label: "OrderPlaced", Type: model.StepAsync},
			{Index: 6, From: "api-gateway", To: "web-frontend", Label: "201 Created", Type: model.StepReturn},
		},
	}
}

func testFlat() map[string]*model.Element {
	return map[string]*model.Element{
		"web-frontend":    {Kind: "container", Title: "Web Frontend"},
		"api-gateway":     {Kind: "container", Title: "API Gateway"},
		"order-service":   {Kind: "service", Title: "Order Service"},
		"payment-service": {Kind: "service", Title: "Payment Service"},
		"message-broker":  {Kind: "queue", Title: "Message Broker"},
	}
}

func TestRenderSequencePlantUML_ContainsTitle(t *testing.T) {
	out := RenderSequencePlantUML(testDynamicView(), testFlat())
	if !strings.Contains(out, "title Checkout Flow") {
		t.Errorf("expected title in output:\n%s", out)
	}
}

func TestRenderSequencePlantUML_ContainsParticipants(t *testing.T) {
	out := RenderSequencePlantUML(testDynamicView(), testFlat())
	if !strings.Contains(out, `"Web Frontend"`) {
		t.Errorf("expected participant title in output:\n%s", out)
	}
	if !strings.Contains(out, "web_frontend") {
		t.Errorf("expected sanitized participant ID in output:\n%s", out)
	}
}

func TestRenderSequencePlantUML_ArrowTypes(t *testing.T) {
	out := RenderSequencePlantUML(testDynamicView(), testFlat())
	// sync arrow
	if !strings.Contains(out, "web_frontend -> api_gateway") {
		t.Errorf("expected sync arrow:\n%s", out)
	}
	// return arrow
	if !strings.Contains(out, "payment_service --> order_service") {
		t.Errorf("expected return arrow:\n%s", out)
	}
	// async arrow
	if !strings.Contains(out, "order_service ->> message_broker") {
		t.Errorf("expected async arrow:\n%s", out)
	}
}

func TestRenderSequencePlantUML_StepLabelsNumbered(t *testing.T) {
	out := RenderSequencePlantUML(testDynamicView(), testFlat())
	if !strings.Contains(out, "1. POST /orders") {
		t.Errorf("expected numbered step label:\n%s", out)
	}
}

func TestRenderSequencePlantUML_StartsAndEndsCorrectly(t *testing.T) {
	out := RenderSequencePlantUML(testDynamicView(), testFlat())
	if !strings.HasPrefix(out, "@startuml") {
		t.Errorf("expected @startuml at start:\n%s", out)
	}
	if !strings.Contains(out, "@enduml") {
		t.Errorf("expected @enduml at end:\n%s", out)
	}
}

func TestRenderSequencePlantUML_StepsInOrder(t *testing.T) {
	// Provide steps out of order — output must still be sorted by index.
	view := model.DynamicView{
		Key:   "test",
		Title: "Test",
		Steps: []model.SequenceStep{
			{Index: 3, From: "c", To: "a", Label: "third", Type: model.StepSync},
			{Index: 1, From: "a", To: "b", Label: "first", Type: model.StepSync},
			{Index: 2, From: "b", To: "c", Label: "second", Type: model.StepSync},
		},
	}
	out := RenderSequencePlantUML(view, nil)
	pos1 := strings.Index(out, "1. first")
	pos2 := strings.Index(out, "2. second")
	pos3 := strings.Index(out, "3. third")
	if pos1 > pos2 || pos2 > pos3 {
		t.Errorf("steps not in order in output:\n%s", out)
	}
}

func TestRenderSequenceMermaid_ContainsTitle(t *testing.T) {
	out := RenderSequenceMermaid(testDynamicView(), testFlat())
	if !strings.Contains(out, "sequenceDiagram") {
		t.Errorf("expected sequenceDiagram header:\n%s", out)
	}
	if !strings.Contains(out, "title Checkout Flow") {
		t.Errorf("expected title:\n%s", out)
	}
}

func TestRenderSequenceMermaid_ArrowTypes(t *testing.T) {
	out := RenderSequenceMermaid(testDynamicView(), testFlat())
	// sync
	if !strings.Contains(out, "web_frontend->>api_gateway") {
		t.Errorf("expected sync arrow in mermaid:\n%s", out)
	}
	// return
	if !strings.Contains(out, "payment_service-->>order_service") {
		t.Errorf("expected return arrow in mermaid:\n%s", out)
	}
	// async
	if !strings.Contains(out, "order_service-)message_broker") {
		t.Errorf("expected async arrow in mermaid:\n%s", out)
	}
}

func TestRenderSequenceMermaid_ParticipantTitleFromFlat(t *testing.T) {
	out := RenderSequenceMermaid(testDynamicView(), testFlat())
	if !strings.Contains(out, "web_frontend as Web Frontend") {
		t.Errorf("expected participant with title from flat map:\n%s", out)
	}
}

func TestRenderSequenceMermaid_FallbackToIDWhenNoFlat(t *testing.T) {
	view := model.DynamicView{
		Key:   "test",
		Title: "Test",
		Steps: []model.SequenceStep{
			{Index: 1, From: "elem-a", To: "elem-b", Label: "call", Type: model.StepSync},
		},
	}
	out := RenderSequenceMermaid(view, nil)
	if !strings.Contains(out, "elem_a as elem-a") {
		t.Errorf("expected fallback to ID when flat is nil:\n%s", out)
	}
}

func TestRenderSequencePlantUML_DefaultTypeIsSync(t *testing.T) {
	view := model.DynamicView{
		Key:   "test",
		Title: "Test",
		Steps: []model.SequenceStep{
			{Index: 1, From: "a", To: "b", Label: "call"}, // no Type → sync
		},
	}
	out := RenderSequencePlantUML(view, nil)
	if !strings.Contains(out, "a -> b") {
		t.Errorf("expected sync arrow for omitted type:\n%s", out)
	}
}
