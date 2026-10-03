# Plan: Incremental GrMob Todo App

**Created:** 2026-10-03
**Goal:** Turn the `grmob new` counter scaffold into a todo app one step at a
time. Each step swaps hand-rolled `core` primitives for `comps` widgets, so the
point of each widget is easy to see.

**Stack:** grmob v0.5.0 (needs Go ≥ 1.26.1; the toolchain auto-switches to
1.26.8 because the local Go is 1.25.4). Module `grmob-exer-todo`, app ID
`com.rohanthewiz.todo`.

**Reference implementation:** `examples/todoapp` in the module cache
(`$(go list -m -f '{{.Dir}}' github.com/rohanthewiz/grmob)/examples/todoapp`).
It covers most of steps 2–5 and is useful for checking work.

**API docs:** `$(go list -m -f '{{.Dir}}' github.com/rohanthewiz/grmob)/docs/api/comps-*.md`.
Read a widget's page for its exact fields before using it.

---

## Status

| Step | Status |
|---|---|
| 1. Core-primitives todo list | ✅ Done |
| 2. Swap in structural comps | ⬜ Next |
| 3. Filters + empty state + counts | ⬜ |
| 4. Scrolling list + app bar | ⬜ |
| 5. Persistence with bytdb | ⬜ |
| 6. Detail/edit screen with navigation + forms | ⬜ |

---

## Step 1: Core primitives only ✅

Files: `app/app.go`, `app/app_test.go`

- `Todo{ID int, Title string, Done bool}` is a value type. Mutations copy the
  slice and call `Set`, never edit in place.
- Root state, in fixed hook order: `todos []Todo`, `draft string`,
  `nextID int`. IDs come from a monotonic counter because indices shift on
  delete.
- Mutation helpers `addTodo`, `setDone`, `removeTodo` are the single choke
  point for writes. Step 5 adds write-through here.
- Derived `remaining` is recomputed each pass, never stored.
- Entry row: `core.InputWithSubmit(..., addTodo, core.FlexGrow(1))` +
  `core.Button("Add", addTodo)`, so the return key and the button share one
  handler.
- Rows: `core.For` + `core.Keyed("todo-<id>", ...)`. `todoRow` is a pure
  function with **no hooks**.
- `core.IfElse` switches between the empty message and the list.

Tests: add via button, add via submit, blank input ignored, toggle checkbox,
delete by accessibility label, and `core.Concerns()` empty.

## Step 2: Structural comps

Replace the hand-rolled layout:

| Before (core) | After (comps) | What the widget takes over |
|---|---|---|
| `SafeArea(Column(Padding, Gap, …))` | `comps.Screen{Fill, Gap, Children}` | safe area, padding, theme spacing |
| `Row(InputWithSubmit, Button)` | `comps.InputRow{Value, Placeholder, OnChange, OnSubmit, Button: comps.Button{Label: "Add"}}` | gap, flex-grow, button inherits `OnSubmit` |
| `Row(Checkbox, Text, Button)` | `comps.ListRow{Leading, Content, Trailing, AccessibilityLabel}` | trailing-edge pinning, vertical centring |
| `core.Button("✕")` | `comps.Button{Label: "✕", Variant: comps.VariantError}` | destructive color from the theme's Error role |

- A task checkbox goes in `ListRow.Leading`. `CheckboxRow` (trailing checkbox)
  is for named options and settings, not for the item being marked.
- Use `ListRow.Content` rather than `Title` because the title dims when done,
  and `Title` takes the theme Body style verbatim.
- Add a row accessibility label: "<title>, completed" / "<title>, not completed".
- Tests: the helpers find `Button`/`Input` by node type. Check whether comps
  emit the same node types (they render to core nodes) and adjust the helpers
  if needed.

## Step 3: Filters, empty state, counts

- Add a `filter int` state (`filterAll`/`filterActive`/`filterDone` as iota,
  doubling as indices into `[]string{"All","Active","Done"}`).
- `comps.SegmentedControl{Labels, Selected, OnSelect, KeyPrefix: "filter-",
  SegmentLabel}` for the filter bar.
- `comps.EmptyState` with a message for each filter in place of the plain
  `Text`.
- Footer `comps.ListRow{Content: "<n> items left", Trailing: clearButton}`,
  where `clearButton` is a nil `core.View` unless something is done. A nil
  slot emits no node, unlike `core.If`.
- `comps.Badge` for the remaining count (optional; compare with the plain text).
- Add `clearDone` mutation helper.

## Step 4: Scrolling list + app bar

- Move rows into `core.List(core.FlexGrow(1), core.For(...))`. It is
  virtualized (LazyColumn / LazyVStack natively).
- `comps.Screen{Fill: true}` is required for the List to grow, and
  `Scroll: false` because nested scroll views fight over the drag.
- Replace the title `Text` with `comps.AppBar{Title: "Todo", Subtitle: "<n> left"}`.
- `comps.Separator{}` between the controls and the list.

## Step 5: Persistence with bytdb

Follow `examples/todoapp/store.go`:

```
first render ──▶ openStore() ──▶ snapshot() ──▶ NewState initial values
tap / submit ──▶ mutation helper ──▶ State.Set (UI)
                               └───▶ store write-through (disk)
```

- New `app/store.go`. `openStore()` reads `mobile.DataDir()`; if it's empty
  (browser, bare tests), return nil and run in memory.
- Every store method is nil-receiver-safe, so the app calls them
  unconditionally.
- A mutex-guarded package singleton, because bytdb holds an exclusive file
  lock. Reopen when the dir changes (tests use a fresh `t.TempDir()`).
- **Never open in `init`.** The host calls `SetDataDir` after init.
- Seed `NewState` from the snapshot rather than loading in `hooks.UseEffect`,
  which runs on its own goroutine and would mount empty first.
- Write through synchronously in the mutation helpers. Persist `nextID` too.
- A failed open logs and degrades to in-memory.
- Tests: `mobile.SetDataDir(t.TempDir())` before `render.New`, then check
  that a second `Manager` sees the same todos.

## Step 6: Detail screen with navigation + forms

- Wrap the root in `core.Navigator`. Keep `todos` state **above** the
  navigator by capturing the outer `ctx`, so it survives navigation.
- Tapping a row opens `core.Push(ctx, detailScreen(id))`. Use
  `ListRow.OnTap` if available, otherwise `core.OnClick` +
  `AccessibilityRole(RoleButton)`.
- Detail screen: `forms.UseForm` with a `title` field
  (`Required`, `MaxLen`) and an optional `notes` field (add `Notes` to
  `Todo`), shown in `comps.FormField{Label, Error, Input}`.
- Save calls an `updateTodo` mutation helper (with write-through), then
  `core.Pop`. `comps.AppBar` gets a back button when `core.CanPop(ctx)`.
- Route state is discarded on pop, which is fine because the form re-seeds
  from the todo.

---

## Gotchas found so far

- `core.Align(...)` sets **text** alignment (`Style.Align`). For cross-axis
  centring in a Row, use `core.AlignItemsProp(core.AlignItemsCenter)`.
- The theme insets every `Column`/`Row`. A nested one needs `core.Padding(0)`.
- No strikethrough style field yet, so done todos are shown by dimming.
- JSON node shape for tests: `{Type, Key, Props, Style, Children}`.
  `AccessibilityLabel` lives under `Style`, not `Props`. Callback props are
  `onClick`, `onChange`, `onSubmit`, `onToggle`.
- `render.Manager` dispatchers: `DispatchCallback`, `DispatchTextCallback`,
  `DispatchBoolCallback`, `DispatchIntCallback`. Callback IDs are reissued
  every pass, so re-read the tree before each tap.

## Working loop

```sh
./dev.sh          # http://localhost:8080, hot-swaps on save
go test ./app     # fastest loop; debug mode on in TestMain
sh build.sh       # one-off browser build
```
