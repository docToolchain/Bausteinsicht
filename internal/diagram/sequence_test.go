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
	out := RenderSequencePlantUML(testDynamicView(), testFlat(), nil)
	if !strings.Contains(out, "title Checkout Flow") {
		t.Errorf("expected title in output:\n%s", out)
	}
}

func TestRenderSequencePlantUML_ContainsParticipants(t *testing.T) {
	out := RenderSequencePlantUML(testDynamicView(), testFlat(), nil)
	if !strings.Contains(out, `"Web Frontend"`) {
		t.Errorf("expected participant title in output:\n%s", out)
	}
	if !strings.Contains(out, "web_frontend") {
		t.Errorf("expected sanitized participant ID in output:\n%s", out)
	}
}

func TestRenderSequencePlantUML_ArrowTypes(t *testing.T) {
	out := RenderSequencePlantUML(testDynamicView(), testFlat(), nil)
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

// Step numbers come from autonumber, not from the label text (#637).
func TestRenderSequencePlantUML_UsesAutonumber(t *testing.T) {
	out := RenderSequencePlantUML(testDynamicView(), testFlat(), nil)
	if !strings.Contains(out, "\nautonumber\n") {
		t.Errorf("expected autonumber:\n%s", out)
	}
	if !strings.Contains(out, ": POST /orders\n") || strings.Contains(out, "1. POST /orders") {
		t.Errorf("label must not carry the step number:\n%s", out)
	}
}

func TestRenderSequencePlantUML_StartsAndEndsCorrectly(t *testing.T) {
	out := RenderSequencePlantUML(testDynamicView(), testFlat(), nil)
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
	out := RenderSequencePlantUML(view, nil, nil)
	pos1 := strings.Index(out, ": first")
	pos2 := strings.Index(out, ": second")
	pos3 := strings.Index(out, ": third")
	if pos1 > pos2 || pos2 > pos3 {
		t.Errorf("steps not in order in output:\n%s", out)
	}
}

func TestRenderSequenceMermaid_ContainsTitle(t *testing.T) {
	out := RenderSequenceMermaid(testDynamicView(), testFlat(), nil)
	if !strings.Contains(out, "sequenceDiagram") {
		t.Errorf("expected sequenceDiagram header:\n%s", out)
	}
	if !strings.Contains(out, "title Checkout Flow") {
		t.Errorf("expected title:\n%s", out)
	}
}

func TestRenderSequenceMermaid_ArrowTypes(t *testing.T) {
	out := RenderSequenceMermaid(testDynamicView(), testFlat(), nil)
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
	out := RenderSequenceMermaid(testDynamicView(), testFlat(), nil)
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
	out := RenderSequenceMermaid(view, nil, nil)
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
	out := RenderSequencePlantUML(view, nil, nil)
	if !strings.Contains(out, "a -> b") {
		t.Errorf("expected sync arrow for omitted type:\n%s", out)
	}
}

func TestRenderSequenceMermaid_UsesAutonumber(t *testing.T) {
	out := RenderSequenceMermaid(testDynamicView(), testFlat(), nil)
	if !strings.Contains(out, "\n    autonumber\n") {
		t.Errorf("expected autonumber:\n%s", out)
	}
	if strings.Contains(out, "1. POST /orders") {
		t.Errorf("label must not carry the step number:\n%s", out)
	}
}

// Double quotes in a message label need no escaping (#637).
func TestRenderSequence_KeepsDoubleQuotesInLabels(t *testing.T) {
	view := model.DynamicView{Key: "q", Title: "Q", Steps: []model.SequenceStep{
		{Index: 1, From: "a", To: "b", Label: `submit "order"`},
	}}
	if out := RenderSequencePlantUML(view, nil, nil); !strings.Contains(out, `: submit "order"`) {
		t.Errorf("PlantUML changed the quotes:\n%s", out)
	}
	if out := RenderSequenceMermaid(view, nil, nil); !strings.Contains(out, `: submit "order"`) {
		t.Errorf("Mermaid changed the quotes:\n%s", out)
	}
}

func participantSpec() *model.Specification {
	return &model.Specification{Elements: map[string]model.ElementKind{
		"actor":    {Notation: "Actor"},
		"system":   {Notation: "Software System"},
		"database": {Notation: "Database"},
		"queue":    {Notation: "Queue"},
		"custom":   {Notation: "Custom", C4: "Person_Ext"},
	}}
}

func participantFlat() map[string]*model.Element {
	return map[string]*model.Element{
		"customer": {Kind: "actor", Title: "Customer"},
		"shop":     {Kind: "system", Title: "Shop"},
		"db":       {Kind: "database", Title: "Orders DB"},
		"mq":       {Kind: "queue", Title: "Events"},
		"partner":  {Kind: "custom", Title: "Partner"},
	}
}

func participantView() model.DynamicView {
	return model.DynamicView{Key: "p", Title: "P", Steps: []model.SequenceStep{
		{Index: 1, From: "customer", To: "shop", Label: "order"},
		{Index: 2, From: "shop", To: "db", Label: "save"},
		{Index: 3, From: "shop", To: "mq", Label: "publish", Type: model.StepAsync},
		{Index: 4, From: "partner", To: "shop", Label: "call"},
	}}
}

// Elements that C4 renders as Person are actors, data stores and queues get
// their own participant shapes (#637).
func TestRenderSequencePlantUML_ParticipantShapes(t *testing.T) {
	out := RenderSequencePlantUML(participantView(), participantFlat(), participantSpec())
	for _, want := range []string{
		`actor "Customer" as customer`,
		`participant "Shop" as shop`,
		`database "Orders DB" as db`,
		`queue "Events" as mq`,
		`actor "Partner" as partner`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestRenderSequenceMermaid_ActorParticipants(t *testing.T) {
	out := RenderSequenceMermaid(participantView(), participantFlat(), participantSpec())
	if !strings.Contains(out, "actor customer as Customer") {
		t.Errorf("expected an actor for the Person kind:\n%s", out)
	}
	if !strings.Contains(out, "participant shop as Shop") {
		t.Errorf("expected a plain participant for a system:\n%s", out)
	}
}

func blockView() model.DynamicView {
	return model.DynamicView{Key: "pay", Title: "Pay", Steps: []model.SequenceStep{
		{Index: 1, From: "customer", To: "shop", Label: "pay"},
		{Index: 2, Type: model.StepAlt, Label: "card accepted"},
		{Index: 3, From: "shop", To: "customer", Label: "receipt", Type: model.StepReturn},
		{Index: 4, Type: model.StepElse, Label: "card declined"},
		{Index: 5, From: "shop", To: "customer", Label: "error", Type: model.StepReturn},
		{Index: 6, Type: model.StepEnd},
		{Index: 7, Type: model.StepOpt, Label: "newsletter"},
		{Index: 8, From: "shop", To: "customer", Label: "offer"},
		{Index: 9, Type: model.StepEnd},
		{Index: 10, Type: model.StepGroup, Label: "cleanup"},
		{Index: 11, From: "shop", To: "shop", Label: "clear cart"},
		{Index: 12, Type: model.StepEnd},
	}}
}

func TestRenderSequencePlantUML_Blocks(t *testing.T) {
	out := RenderSequencePlantUML(blockView(), nil, nil)
	for _, want := range []string{
		"alt card accepted\n", "else card declined\n", "opt newsletter\n", "group cleanup\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if n := strings.Count(out, "\nend\n"); n != 3 {
		t.Errorf("want 3 end lines, got %d:\n%s", n, out)
	}
	// Block steps are not participants.
	if strings.Contains(out, "participant \"\"") || strings.Contains(out, " as \n") {
		t.Errorf("block step leaked into the participants:\n%s", out)
	}
	if !strings.Contains(out, "  shop --> customer : receipt") {
		t.Errorf("steps inside a block must be indented:\n%s", out)
	}
}

func TestRenderSequenceMermaid_Blocks(t *testing.T) {
	out := RenderSequenceMermaid(blockView(), nil, nil)
	for _, want := range []string{"alt card accepted\n", "else card declined\n", "opt newsletter\n", "rect rgb(240, 240, 240)\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if n := strings.Count(out, "    end\n"); n != 3 {
		t.Errorf("want 3 end lines, got %d:\n%s", n, out)
	}
	if strings.Contains(out, "participant  as") {
		t.Errorf("block step leaked into the participants:\n%s", out)
	}
}
