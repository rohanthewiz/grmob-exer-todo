// Package app is Todo's root view. Every target mounts this package:
// wasm/main.go for the browser, and `grmob android` / `grmob ios`, which bind
// it into the native shells.
//
// Step 1 of the incremental build: a working todo list made only from core
// primitives (Input, Button, Checkbox, For, Keyed). Later steps swap pieces
// for comps widgets, so the hand-rolled versions here are the baseline each
// widget is measured against.
package app

import (
	"fmt"
	"strings"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/mobile"
)

// init registers the root view with the mobile bridge, which is the whole
// integration contract for the native shells: they call the bridge, and the
// bridge renders whatever registered here. The browser host mounts App
// directly (see wasm/main.go), so on that target this registration is unused.
func init() {
	mobile.Register(core.NewContext(), App)
}

// AppName gives gomobile a bindable symbol. gomobile links a bound package
// only if it exports something it can bind, and App — a function-typed value
// — is not, so without this the package and the init above would be dropped
// from the native library and the shell would start with no app.
func AppName() string { return "Todo" }

// Todo is a plain value, not a pointer. Mutations copy the slice and replace
// items wholesale, so the previous pass's tree never aliases the new state —
// the diff compares by pointer identity, and shared pointers would let an
// in-place edit go unseen.
type Todo struct {
	ID    int
	Title string
	Done  bool
}

// App is called again on every render pass: it reads state, returns a tree,
// and never mutates the screen itself.
//
// All state lives here at the root and flows down as values + closures. Hooks
// are positional slots, so a NewState call inside each row would shift onto a
// neighbour's slot as soon as a row is added or removed. Rows are therefore
// pure functions of their Todo.
func App(ctx *core.Context) core.View {
	// Hook order is fixed: these four calls run unconditionally, in this
	// order, on every pass.
	todos := core.NewState(ctx, []Todo{})
	draft := core.NewState(ctx, "") // the text currently in the input
	// IDs come from a monotonic counter rather than slice indices: deleting a
	// row shifts every index after it, which would break both list keys and
	// the closures that target a specific todo.
	nextID := core.NewState(ctx, 1)

	// --- Mutations ---------------------------------------------------------
	// Every write goes through one of these helpers, and each builds a fresh
	// slice before Set. Set is the whole update path: it marks the context
	// dirty, and the next pass's diff reaches the screen.

	addTodo := func() {
		title := strings.TrimSpace(draft.Get())
		if title == "" {
			return // ignore blank submissions instead of adding empty rows
		}
		next := append(append([]Todo(nil), todos.Get()...), Todo{ID: nextID.Get(), Title: title})
		todos.Set(next)
		nextID.Set(nextID.Get() + 1)
		draft.Set("") // clear the field so consecutive adds need no manual erase
	}

	setDone := func(id int, done bool) {
		next := append([]Todo(nil), todos.Get()...)
		for i := range next {
			if next[i].ID == id {
				next[i].Done = done
			}
		}
		todos.Set(next)
	}

	removeTodo := func(id int) {
		next := make([]Todo, 0, len(todos.Get()))
		for _, t := range todos.Get() {
			if t.ID != id {
				next = append(next, t)
			}
		}
		todos.Set(next)
	}

	// --- Derived values ----------------------------------------------------
	// Computed each pass from the single source of truth rather than stored,
	// so they can never drift out of sync with the list.
	remaining := 0
	for _, t := range todos.Get() {
		if !t.Done {
			remaining++
		}
	}

	// --- View --------------------------------------------------------------
	return core.SafeArea(
		core.Column(
			core.Gap(12),
			core.Padding(24),
			core.Text("Todo", core.FontSize(28), core.FontWeight(core.Bold)),

			// Entry row. The Input is controlled: it renders draft and reports
			// each keystroke through onChange. The keyboard's return key
			// (onSubmit) and the Add button are two paths to one handler.
			core.Row(
				// Rows get a theme inset; inside an already padded Column that
				// would indent this row, so Padding(0) clears it.
				core.Padding(0),
				core.Gap(8),
				core.InputWithSubmit(draft.Get(), "What needs doing?",
					func(v string) { draft.Set(v) },
					addTodo,
					core.FlexGrow(1), // the field takes the width the button leaves
				),
				core.Button("Add", addTodo),
			),

			// A conditional *view* is fine — only conditional hooks are not.
			core.IfElse(len(todos.Get()) == 0,
				core.Text("Nothing to do yet — add a task above."),
				core.Column(
					core.Padding(0),
					core.Gap(4),
					core.For(todos.Get(), func(t Todo, _ int) core.View {
						return todoRow(t, setDone, removeTodo)
					}),
				),
			),

			core.Text(fmt.Sprintf("%d left", remaining), core.FontSize(13)),
		),
	)
}

// todoRow is a pure function of its Todo: no hooks, so rows may come and go
// without disturbing the root's slot order.
//
// Keyed matters for a dynamic list. The diff matches children by position; a
// stable key per todo means deleting row 2 replaces only that row instead of
// re-propping every row beneath it.
func todoRow(t Todo, setDone func(int, bool), remove func(int)) core.View {
	// Capture the ID, not an index: the closures outlive this pass and must
	// still address the same todo after the slice is rebuilt.
	id := t.ID

	// Strikethrough isn't a style field, so "done" is shown by dimming.
	color := "#000000"
	if t.Done {
		color = "#8E8E93"
	}

	return core.Keyed(fmt.Sprintf("todo-%d", id), core.Row(
		core.Padding(0),
		core.Gap(10),
		// Cross-axis centring: the checkbox and button are taller than a
		// line of text, so without this the title would sit at the top.
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Checkbox(t.Done, func(v bool) { setDone(id, v) }),
		core.Text(t.Title, core.TextColor(color), core.FlexGrow(1)),
		core.Button("✕", func() { remove(id) },
			core.AccessibilityLabel("Delete "+t.Title)),
	))
}
