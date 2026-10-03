package app

import (
	"encoding/json"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// The fastest loop there is: render.Manager is the same engine every target
// runs, and it needs no browser, simulator or device. Taps go through the
// same dispatch path a finger does.

// TestMain turns on debug mode, which audits every render pass for hook-order
// drift and duplicate keys; assertNoConcerns fails the test on any finding.
func TestMain(m *testing.M) {
	core.SetDebugMode(true)
	m.Run()
}

// TestTodoLifecycle drives the full add → toggle → delete path through the
// same dispatch calls a host makes, then checks debug mode stayed quiet.
func TestTodoLifecycle(t *testing.T) {
	core.ClearConcerns()
	mgr := render.New(core.NewContext(), App)
	defer mgr.Close()

	if !shows(tree(t, mgr), "Nothing to do yet — add a task above.") {
		t.Fatal("an empty list should show the empty message")
	}

	// Add two todos: one via the Add button, one via the keyboard's submit.
	typeInto(t, mgr, "Buy milk")
	tap(t, mgr, "Add")
	typeInto(t, mgr, "Walk dog")
	submit(t, mgr)

	root := tree(t, mgr)
	if !shows(root, "Buy milk") || !shows(root, "Walk dog") || !shows(root, "2 left") {
		t.Fatal("both todos should be listed with 2 left")
	}

	// Blank input is ignored rather than adding an empty row.
	typeInto(t, mgr, "   ")
	tap(t, mgr, "Add")
	if !shows(tree(t, mgr), "2 left") {
		t.Fatal("a blank submission should not add a todo")
	}

	// Toggling the first checkbox completes "Buy milk".
	cb := find(tree(t, mgr), func(n *node) bool { return n.Type == "Checkbox" })
	mgr.DispatchBoolCallback(cb.Props["onToggle"].(string), true)
	if !shows(tree(t, mgr), "1 left") {
		t.Fatal("checking a todo should drop the remaining count")
	}

	// Deleting "Walk dog" leaves only the completed todo.
	tapLabeled(t, mgr, "Delete Walk dog")
	root = tree(t, mgr)
	if shows(root, "Walk dog") || !shows(root, "Buy milk") || !shows(root, "0 left") {
		t.Fatal("delete should remove only the targeted todo")
	}

	if cs := core.Concerns(); len(cs) != 0 {
		t.Fatalf("debug concerns raised:\n%s", core.DumpConcerns())
	}
}

// --- helpers ---------------------------------------------------------------

type node struct {
	Type     string
	Props    map[string]any
	Style    struct{ AccessibilityLabel string }
	Children []*node
}

// tree renders and parses the current tree. Callback IDs are assigned per
// pass, so each tap reads its ID from a fresh tree rather than reusing one.
func tree(t *testing.T, mgr *render.Manager) *node {
	t.Helper()
	var root node
	if err := json.Unmarshal([]byte(mgr.RenderInitial()), &root); err != nil {
		t.Fatalf("tree is not valid JSON: %v", err)
	}
	return &root
}

func find(n *node, pred func(*node) bool) *node {
	if n == nil || pred(n) {
		return n
	}
	for _, c := range n.Children {
		if f := find(c, pred); f != nil {
			return f
		}
	}
	return nil
}

func tap(t *testing.T, mgr *render.Manager, label string) {
	t.Helper()
	b := find(tree(t, mgr), func(n *node) bool { return n.Type == "Button" && n.Props["label"] == label })
	if b == nil {
		t.Fatalf("no Button labeled %q", label)
	}
	mgr.DispatchCallback(b.Props["onClick"].(string))
}

func shows(root *node, text string) bool {
	return find(root, func(n *node) bool { return n.Type == "Text" && n.Props["content"] == text }) != nil
}

// tapLabeled taps the Button whose accessibility label matches — the way to
// reach icon buttons like a row's "✕", whose visible label is not unique.
func tapLabeled(t *testing.T, mgr *render.Manager, a11y string) {
	t.Helper()
	b := find(tree(t, mgr), func(n *node) bool { return n.Type == "Button" && n.Style.AccessibilityLabel == a11y })
	if b == nil {
		t.Fatalf("no Button with accessibility label %q", a11y)
	}
	mgr.DispatchCallback(b.Props["onClick"].(string))
}

// typeInto sends text to the first Input, as a keystroke would.
func typeInto(t *testing.T, mgr *render.Manager, text string) {
	t.Helper()
	mgr.DispatchTextCallback(firstInput(t, mgr).Props["onChange"].(string), text)
}

// submit presses the keyboard's return key on the first Input.
func submit(t *testing.T, mgr *render.Manager) {
	t.Helper()
	mgr.DispatchCallback(firstInput(t, mgr).Props["onSubmit"].(string))
}

func firstInput(t *testing.T, mgr *render.Manager) *node {
	t.Helper()
	in := find(tree(t, mgr), func(n *node) bool { return n.Type == "Input" })
	if in == nil {
		t.Fatal("no Input in the tree")
	}
	return in
}
