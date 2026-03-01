package drawio

import (
	"testing"
)

func TestGenerateLabel(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		technology  string
		description string
		want        string
	}{
		{
			name:       "standard label with title and technology",
			title:      "REST API",
			technology: "Spring Boot",
			want:       `<b>REST API</b><br><font color="#666666">[Spring Boot]</font>`,
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
			want:       `<b>API</b><br><font color="#666666">[Go &amp; &lt;fast&gt;]</font>`,
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
			want:       `<b></b><br><font color="#666666">[Java]</font>`,
		},
		{
			name:        "title, technology and description",
			title:       "REST API",
			technology:  "Spring Boot",
			description: "Handles business logic",
			want:        `<b>REST API</b><br><font color="#666666">[Spring Boot]</font><br><font color="#999999">Handles business logic</font>`,
		},
		{
			name:        "title and description, no technology",
			title:       "Customer",
			description: "End user of the system",
			want:        `<b>Customer</b><br><font color="#999999">End user of the system</font>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateLabel(tt.title, tt.technology, tt.description)
			if got != tt.want {
				t.Errorf("GenerateLabel(%q, %q, %q)\n  got:  %q\n  want: %q", tt.title, tt.technology, tt.description, got, tt.want)
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
		name        string
		html        string
		wantTitle   string
		wantTech    string
		wantDesc    string
	}{
		{
			name:      "standard label with title and technology",
			html:      `<b>REST API</b><br><font color="#666666">[Spring Boot]</font>`,
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
			html:      `<b>API</b><br><font color="#666666">[Go &amp; &lt;fast&gt;]</font>`,
			wantTitle: "API",
			wantTech:  "Go & <fast>",
		},
		{
			name:      "empty label",
			html:      "",
			wantTitle: "",
			wantTech:  "",
		},
		{
			name:      "label with title, tech and description",
			html:      `<b>REST API</b><br><font color="#666666">[Spring Boot]</font><br><font color="#999999">Handles requests</font>`,
			wantTitle: "REST API",
			wantTech:  "Spring Boot",
			wantDesc:  "Handles requests",
		},
		{
			name:      "label with title and description only",
			html:      `<b>Customer</b><br><font color="#999999">End user</font>`,
			wantTitle: "Customer",
			wantDesc:  "End user",
		},
		{
			name:      "legacy label without brackets (backward compat)",
			html:      `<b>REST API</b><br><font color="#666666">Spring Boot</font>`,
			wantTitle: "REST API",
			wantTech:  "Spring Boot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotTech, gotDesc := ParseLabel(tt.html)
			if gotTitle != tt.wantTitle || gotTech != tt.wantTech || gotDesc != tt.wantDesc {
				t.Errorf("ParseLabel(%q)\n  got:  (%q, %q, %q)\n  want: (%q, %q, %q)",
					tt.html, gotTitle, gotTech, gotDesc, tt.wantTitle, tt.wantTech, tt.wantDesc)
			}
		})
	}
}

func TestParseLabel_StripsNestedHTMLTags(t *testing.T) {
	html := `<b><i>Styled</i> Customer</b>`
	gotTitle, gotTech, gotDesc := ParseLabel(html)
	if gotTitle != "Styled Customer" {
		t.Errorf("expected title %q, got %q", "Styled Customer", gotTitle)
	}
	if gotTech != "" {
		t.Errorf("expected empty technology, got %q", gotTech)
	}
	if gotDesc != "" {
		t.Errorf("expected empty description, got %q", gotDesc)
	}
}

func TestParseLabel_StripsUnderlineTags(t *testing.T) {
	html := `<b><u>Important</u> System</b><br><font color="#666666">[Go]</font>`
	gotTitle, gotTech, gotDesc := ParseLabel(html)
	if gotTitle != "Important System" {
		t.Errorf("expected title %q, got %q", "Important System", gotTitle)
	}
	if gotTech != "Go" {
		t.Errorf("expected technology %q, got %q", "Go", gotTech)
	}
	if gotDesc != "" {
		t.Errorf("expected empty description, got %q", gotDesc)
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		title string
		tech  string
		desc  string
	}{
		{"REST API", "Spring Boot", ""},
		{"My Service", "", ""},
		{`A & B < C > D "E"`, "Go", ""},
		{"", "Java", ""},
		{"", "", ""},
		{"REST API", "Spring Boot", "Handles requests"},
		{"Customer", "", "End user of the system"},
	}

	for _, tt := range tests {
		label := GenerateLabel(tt.title, tt.tech, tt.desc)
		gotTitle, gotTech, gotDesc := ParseLabel(label)
		if gotTitle != tt.title || gotTech != tt.tech || gotDesc != tt.desc {
			t.Errorf("round-trip GenerateLabel(%q, %q, %q) -> ParseLabel:\n  got:  (%q, %q, %q)\n  want: (%q, %q, %q)",
				tt.title, tt.tech, tt.desc, gotTitle, gotTech, gotDesc, tt.title, tt.tech, tt.desc)
		}
	}
}
