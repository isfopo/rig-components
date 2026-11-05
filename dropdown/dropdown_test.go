package dropdown

import (
	"testing"
)

func TestDropdownNavAndPick(t *testing.T) {
	d := New([]string{"a", "b", "c"}, 0)
	if d.Count() != 3 {
		t.Fatal("count")
	}
	d = d.Open()
	if !d.Focus || d.Cursor != 0 {
		t.Fatalf("open: focus=%v cursor=%d", d.Focus, d.Cursor)
	}
	d = d.Move(-1)
	if d.Cursor != 2 {
		t.Fatalf("wrap cursor=%d", d.Cursor)
	}
	nm, changed := d.Pick()
	if !changed || nm.Current != 2 || nm.Focus {
		t.Fatalf("pick changed=%v current=%d focus=%v", changed, nm.Current, nm.Focus)
	}
	// reopen and re-pick same
	d2 := nm.Open()
	if _, changed := d2.Pick(); changed {
		t.Fatal("re-pick should not change")
	}
	// close
	if d.Close().Focus {
		t.Fatal("close should unfocus")
	}
}
