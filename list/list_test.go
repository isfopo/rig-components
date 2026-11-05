package list

import "testing"

func TestListNav(t *testing.T) {
	l := New([]string{"a", "b"})
	l = l.SetActive(1)
	if l.Active != 1 {
		t.Fatal("active")
	}
	l = l.Move(1)
	if l.Cursor != 1 {
		t.Fatalf("cursor=%d", l.Cursor)
	}
	l2 := l.SetItems([]string{"x"})
	if l2.Cursor != 0 {
		t.Fatalf("clamp cursor=%d", l2.Cursor)
	}
	if l2.View() == "" {
		t.Fatal("view empty")
	}
}
