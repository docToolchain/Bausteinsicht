package drawio

import (
	"testing"
)

func TestGenerateLabel(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		technology string
		want       string
	}{
		{
			name:       "standard label with title and technology",
			title:      "REST API",
			technology: "Spring Boot",
			want:       `<b>REST API</b><br><font color="#666666">Spring Boot</font>`,
		},
		{
			name:       "title only, no technology",
			title:      "My Service",
			technology: "",
			want:       `<b>My Service</b>`,
		},
		{
			name:       "special characters in title",
			title:      `A & B < C > D "E"`,
			technology: "",
			want:       `<b>A &amp; B &lt; C &gt; D &quot;E&quot;</b>`,
		},
		{
			name:       "special characters in technology",
			title:      "API",
			technology: `Go & <fast>`,
			want:       `<b>API</b><br><font color="#666666">Go &amp; &lt;fast&gt;</font>`,
		},
		{
			name:       "empty title and technology",
			title:      "",
			technology: "",
			want:       `<b></b>`,
		},
		{
			name:       "empty title with technology",
			title:      "",
			technology: "Java",
			want:       `<b></b><br><font color="#666666">Java</font>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateLabel(tt.title, tt.technology)
			if got != tt.want {
				t.Errorf("GenerateLabel(%q, %q)\n  got:  %q\n  want: %q", tt.title, tt.technology, got, tt.want)
			}
		})
	}
}

func TestGenerateActorLabel(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "simple actor label",
			title: "User",
			want:  `<b>User</b>`,
		},
		{
			name:  "actor with special chars",
			title: `Admin & <Root>`,
			want:  `<b>Admin &amp; &lt;Root&gt;</b>`,
		},
		{
			name:  "empty title",
			title: "",
			want:  `<b></b>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateActorLabel(tt.title)
			if got != tt.want {
				t.Errorf("GenerateActorLabel(%q)\n  got:  %q\n  want: %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestParseLabel(t *testing.T) {
	tests := []struct {
		name       string
		html       string
		wantTitle  string
		wantTech   string
	}{
		{
			name:      "standard label with title and technology",
			html:      `<b>REST API</b><br><font color="#666666">Spring Boot</font>`,
			wantTitle: "REST API",
			wantTech:  "Spring Boot",
		},
		{
			name:      "title only, no technology",
			html:      `<b>My Service</b>`,
			wantTitle: "My Service",
			wantTech:  "",
		},
		{
			name:      "non-standard format fallback",
			html:      "Plain text label",
			wantTitle: "Plain text label",
			wantTech:  "",
		},
		{
			name:      "non-standard HTML fallback strips tags",
			html:      "<div>Some label</div>",
			wantTitle: "Some label",
			wantTech:  "",
		},
		{
			name:      "escaped chars in title",
			html:      `<b>A &amp; B &lt;C&gt;</b>`,
			wantTitle: "A & B <C>",
			wantTech:  "",
		},
		{
			name:      "escaped chars in technology",
			html:      `<b>API</b><br><font color="#666666">Go &amp; &lt;fast&gt;</font>`,
			wantTitle: "API",
			wantTech:  "Go & <fast>",
		},
		{
			name:      "empty label",
			html:      "",
			wantTitle: "",
			wantTech:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotTech := ParseLabel(tt.html)
			if gotTitle != tt.wantTitle || gotTech != tt.wantTech {
				t.Errorf("ParseLabel(%q)\n  got:  (%q, %q)\n  want: (%q, %q)",
					tt.html, gotTitle, gotTech, tt.wantTitle, tt.wantTech)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		title string
		tech  string
	}{
		{"REST API", "Spring Boot"},
		{"My Service", ""},
		{`A & B < C > D "E"`, "Go"},
		{"", "Java"},
		{"", ""},
	}

	for _, tt := range tests {
		label := GenerateLabel(tt.title, tt.tech)
		gotTitle, gotTech := ParseLabel(label)
		if gotTitle != tt.title || gotTech != tt.tech {
			t.Errorf("round-trip GenerateLabel(%q, %q) -> ParseLabel:\n  got:  (%q, %q)\n  want: (%q, %q)",
				tt.title, tt.tech, gotTitle, gotTech, tt.title, tt.tech)
		}
	}
}
