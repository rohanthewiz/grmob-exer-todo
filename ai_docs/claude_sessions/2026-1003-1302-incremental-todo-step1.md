# Session: Scaffold GrMob app + incremental todo, step 1

**Session ID:** `41bd1255-e5b5-4a0a-85c8-bbd6ba560f69`
**Date:** 2026-10-03
**Project:** `~/projs/go/grmob-exer-todo`
**Plan:** `ai_docs/plans/2026-1003-incremental-todo-app.md`

## Goal

1. Create a barebones GrMob app (following the grmob-native-mobile-go skill).
2. Turn it into a todo app incrementally, to learn how GrMob's core primitives
   and `comps` widgets work.

## What was done

### 1. Scaffolded the app

- Fetched the latest skill from
  `https://raw.githubusercontent.com/rohanthewiz/grmob/refs/heads/master/ai_docs/SKILL.md`.
  It was identical to the local fallback copy.
- Ran the following in the (empty) project dir:
  ```sh
  go run github.com/rohanthewiz/grmob/cmd/grmob@latest new . -name "Todo" -id com.rohanthewiz.todo
  ```
- Result: module `grmob-exer-todo`, grmob **v0.5.0**. grmob needs Go ≥ 1.26.1;
  local Go is 1.25.4, so the toolchain auto-switches to go1.26.8.
- Generated files: `app/app.go`, `app/app_test.go`, `wasm/main.go`,
  `wasm/index.html`, `build.sh`, `dev.sh`, `grmob.json`, `README.md`,
  `.gitignore`. The browser build products under `wasm/` are gitignored.

### 2. Step 1: todo list from core primitives only

The counter in `app/app.go` was replaced with a todo list built only from
`core` constructors. The hand-rolled version is the baseline each `comps`
widget gets compared against in later steps.

- `Todo{ID, Title, Done}` is a value type. Mutations copy the slice and call
  `Set`.
- Root state, in fixed hook order: `todos`, `draft`, `nextID`. A monotonic ID
  counter is used because indices shift on delete.
- Mutation helpers `addTodo` / `setDone` / `removeTodo` are the single write
  choke point. Step 5 will add bytdb write-through there.
- Derived `remaining` is recomputed each pass.
- Entry row: `core.InputWithSubmit(..., addTodo, core.FlexGrow(1))` +
  `core.Button("Add", addTodo)`, so one handler serves two paths.
- Rows: `core.For` + `core.Keyed("todo-<id>", ...)`. `todoRow` is a pure
  function with no hooks. The row is
  `Row(Checkbox, Text(FlexGrow 1), Button("✕"))` with
  `AlignItemsProp(AlignItemsCenter)`.
- `core.IfElse` switches between the empty message and the list.

### 3. Tests (`app/app_test.go`)

- `TestCounter` was replaced with `TestTodoLifecycle`: add via button, add via
  submit, blank ignored, checkbox toggle, delete by accessibility label, and
  `core.Concerns()` empty.
- New helpers: `tapLabeled` (finds a Button by `Style.AccessibilityLabel`),
  `typeInto`, `submit`, `firstInput`. The `node` struct gained a
  `Style{AccessibilityLabel}` field.
- `go test ./app` passes and `sh build.sh` builds `wasm/main.wasm`. Not yet
  checked by eye in a browser.

### 4. Saved the plan

`ai_docs/plans/2026-1003-incremental-todo-app.md` holds the six-step plan with
a status table, the widget swaps for each step, and the gotchas found so far.

## Gotchas discovered

- `core.Align(...)` sets `Style.Align`, which is **text** alignment. For
  cross-axis centring in a Row, use `core.AlignItemsProp(core.AlignItemsCenter)`.
- Rendered-node JSON is `{Type, Key, Props, Style, Children}`.
  `AccessibilityLabel` is under `Style`. Callback prop names are `onClick`,
  `onChange`, `onSubmit` and `onToggle`.
- Callback IDs are reissued every pass, so test helpers re-read the tree
  before each dispatch.
- The theme insets every Column/Row. Nested ones need `core.Padding(0)`.
- There's no strikethrough style field, so done todos are shown by dimming.

## References

- Reference implementation: `examples/todoapp` in
  `~/go/pkg/mod/github.com/rohanthewiz/grmob@v0.5.0/`
- Widget API pages: `.../grmob@v0.5.0/docs/api/comps-*.md`

## Next

- Step 2: swap in structural comps (`comps.Screen`, `comps.InputRow`,
  `comps.ListRow`, `comps.Button{Variant: VariantError}`). Check whether the
  test helpers' node-type lookups still match.
- Step 3: All/Active/Done filter with `comps.SegmentedControl`,
  `comps.EmptyState`, footer count / `comps.Badge`, and a "Clear completed"
  mutation.
- Step 4: virtualized `core.List(FlexGrow(1))` with `Screen{Fill: true}`,
  `comps.AppBar`, and `comps.Separator`.
- Step 5: bytdb persistence via `app/store.go` (lazy open, nil-safe,
  mutex singleton, write-through, seeded `NewState`).
- Step 6: detail/edit screen with `core.Navigator`/`Push`/`Pop`,
  `forms.UseForm`, and `comps.FormField`. Check `ListRow` tap handling,
  `EmptyState` and `Badge` fields against the docs.
- Check step 1 by eye in the browser (`./dev.sh`).
- Set up a git remote. The repo was initialized locally this session with no
  remote, so the push could not happen.
