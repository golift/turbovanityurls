package handler

import "testing"

func TestJoinRedirect(t *testing.T) {
	t.Parallel()

	const (
		base   = "https://github.com/golift/xtractr"
		vanity = "/xtractr"
	)

	tests := []struct {
		name, req, want string
		ok              bool
	}{
		{name: "releases", req: "/xtractr/releases", want: base + "/releases", ok: true},
		{name: "wiki", req: "/xtractr/wiki/Home", want: base + "/wiki/Home", ok: true},
		{name: "dot-dot", req: "/xtractr/releases/../../../evil", ok: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, ok := joinRedirect(base, test.req, vanity)
			if ok != test.ok || (test.ok && got != test.want) {
				t.Fatalf("got %q ok=%v, want %q ok=%v", got, ok, test.want, test.ok)
			}
		})
	}

	if _, ok := joinRedirect("javascript:alert(1)", "/xtractr/releases", vanity); ok {
		t.Fatal("javascript scheme must be rejected")
	}

	if _, ok := joinRedirect("/relative", "/xtractr/releases", vanity); ok {
		t.Fatal("relative base must be rejected")
	}
}
