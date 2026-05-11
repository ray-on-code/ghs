package ui

import (
	"strings"
	"testing"

	ghapi "github.com/ray-on-code/ghs/pkg/github"
)

func TestFormatRepoLabel_Basic(t *testing.T) {
	r := &ghapi.Repository{
		Owner:    "octo",
		Name:     "hello",
		FullName: "octo/hello",
	}
	got := FormatRepoLabel(r)
	if got != "octo/hello" {
		t.Errorf("got %q, want %q", got, "octo/hello")
	}
}

func TestFormatRepoLabel_Nil(t *testing.T) {
	if got := FormatRepoLabel(nil); got != "" {
		t.Errorf("nil repo should return empty, got %q", got)
	}
}

func TestFormatRepoLabel_FullNameMissing(t *testing.T) {
	r := &ghapi.Repository{Owner: "octo", Name: "hello"}
	got := FormatRepoLabel(r)
	if got != "octo/hello" {
		t.Errorf("got %q, want %q", got, "octo/hello")
	}
}

func TestFormatRepoLabel_Flags(t *testing.T) {
	r := &ghapi.Repository{
		FullName: "octo/hello",
		Private:  true,
		Fork:     true,
		Archived: true,
	}
	got := FormatRepoLabel(r)
	want := "octo/hello [private,fork,archived]"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatRepoLabel_TruncatesLongDescription(t *testing.T) {
	long := strings.Repeat("a", 100)
	r := &ghapi.Repository{
		FullName:    "octo/hello",
		Description: long,
	}
	got := FormatRepoLabel(r)
	if !strings.Contains(got, "—") {
		t.Errorf("expected separator '—' in %q", got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("long description should be truncated with …, got %q", got)
	}
}

func TestFormatRepoLabel_ShortDescriptionUntouched(t *testing.T) {
	r := &ghapi.Repository{
		FullName:    "octo/hello",
		Description: "a short one",
	}
	got := FormatRepoLabel(r)
	if !strings.HasSuffix(got, "a short one") {
		t.Errorf("expected description preserved, got %q", got)
	}
}

// SurveyPrompter が Prompter インターフェースを満たすことのコンパイル時保証。
func TestSurveyPrompter_ImplementsPrompter(t *testing.T) {
	var _ Prompter = SurveyPrompter{}
}

func TestBuildRepoLookup(t *testing.T) {
	repos := []ghapi.Repository{
		{Owner: "octo", Name: "a", FullName: "octo/a"},
		{Owner: "octo", Name: "b", FullName: "octo/b", Private: true},
	}
	labels, lookup := BuildRepoLookup(repos)
	if len(labels) != 2 || len(lookup) != 2 {
		t.Fatalf("labels=%d lookup=%d", len(labels), len(lookup))
	}
	if labels[0] != "octo/a" {
		t.Errorf("labels[0] = %q", labels[0])
	}
	if labels[1] != "octo/b [private]" {
		t.Errorf("labels[1] = %q", labels[1])
	}
	if lookup["octo/a"].Name != "a" {
		t.Errorf("lookup mismatch: %+v", lookup["octo/a"])
	}
	if lookup["octo/b [private]"].Name != "b" {
		t.Errorf("lookup mismatch: %+v", lookup["octo/b [private]"])
	}
}

func TestResolveRepoSelection_Found(t *testing.T) {
	repos := []ghapi.Repository{{Owner: "o", Name: "r", FullName: "o/r"}}
	_, lookup := BuildRepoLookup(repos)
	got, err := ResolveRepoSelection("o/r", lookup)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "r" {
		t.Errorf("got %+v", got)
	}
}

func TestResolveRepoSelection_NotFound(t *testing.T) {
	lookup := map[string]*ghapi.Repository{}
	if _, err := ResolveRepoSelection("missing", lookup); err == nil {
		t.Fatal("expected error for missing label")
	}
}

func TestFilterByLabel(t *testing.T) {
	cases := []struct {
		filter, option string
		want           bool
	}{
		{"", "anything", true},
		{"foo", "Foobar", true},
		{"FOO", "foobar", true},
		{"baz", "foobar", false},
		{"oct", "octo/repo", true},
	}
	for _, tc := range cases {
		if got := FilterByLabel(tc.filter, tc.option, 0); got != tc.want {
			t.Errorf("FilterByLabel(%q, %q) = %v, want %v", tc.filter, tc.option, got, tc.want)
		}
	}
}
