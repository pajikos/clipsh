package template

import (
	"strings"
	"testing"
)

func TestRender_Placeholders(t *testing.T) {
	ctx := Context{
		Timestamp: 1713657600,
		Ext:       "png",
		Basename:  "report",
		Hostname:  "mymac",
		User:      "pavel",
		Random:    "deadbeef",
	}
	cases := []struct {
		tmpl string
		want string
	}{
		{"/tmp/clipsh-{timestamp}.{ext}", "/tmp/clipsh-1713657600.png"},
		{"/uploads/{basename}-{random}.{ext}", "/uploads/report-deadbeef.png"},
		{"/home/{user}/{hostname}.log", "/home/pavel/mymac.log"},
		{"no-placeholders", "no-placeholders"},
	}
	for _, c := range cases {
		got, err := Render(c.tmpl, ctx)
		if err != nil {
			t.Fatalf("Render(%q): unexpected error: %v", c.tmpl, err)
		}
		if got != c.want {
			t.Errorf("Render(%q) = %q, want %q", c.tmpl, got, c.want)
		}
	}
}

func TestRender_UnknownPlaceholder(t *testing.T) {
	_, err := Render("/tmp/{nope}", Context{})
	if err == nil {
		t.Fatal("expected error for unknown placeholder")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("error %q should mention the bad placeholder", err)
	}
}

func TestRender_EmptyValues(t *testing.T) {
	got, err := Render("/tmp/{basename}.{ext}", Context{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/tmp/." {
		t.Errorf("empty context should render as literal dots: got %q", got)
	}
}

func TestAuto_FillsDefaults(t *testing.T) {
	var ctx Context
	ctx.Auto()
	if ctx.Timestamp == 0 {
		t.Error("Auto() should set Timestamp")
	}
	if ctx.Random == "" || len(ctx.Random) != 8 {
		t.Errorf("Auto() should set Random to 8 hex chars, got %q", ctx.Random)
	}
	if ctx.Hostname == "" {
		t.Error("Auto() should set Hostname")
	}
}

func TestAuto_PreservesExplicit(t *testing.T) {
	ctx := Context{Timestamp: 42, Random: "cafe"}
	ctx.Auto()
	if ctx.Timestamp != 42 {
		t.Errorf("Auto() overwrote explicit Timestamp: got %d", ctx.Timestamp)
	}
	if ctx.Random != "cafe" {
		t.Errorf("Auto() overwrote explicit Random: got %q", ctx.Random)
	}
}
