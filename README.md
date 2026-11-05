# rig components (library module)

Reusable, generic Bubble Tea components extracted from `rig`. This is a
separate Go module (`github.com/isfopo/rig/components`) so it can be
versioned and consumed by other apps independently of `rig`.

## Packages

- `dropdown` — keyboard-driven option list with open/closed focus, cursor
  wrap-around, and change reporting.
- `list` — keyboard-navigable list with cursor and active/open marker.
- `lamps` — activity indicators that light up briefly after traffic.
- `meter` — terminal capture-level meter (pure render function).

## Component convention

Each component is a self-contained `tea.Model`:

- lives in its own package
- owns its state (pointer fields for shared atomics where needed)
- implements `Init()`, `Update()`, `View()`
- defines its own messages (e.g. `dropdown.PickedMsg`, `lamps.LampOffMsg`)
- has no imports from `rig` internals

Parents compose components by holding them as fields, delegating input in
`Update`, and rendering via `View()`.

## Developing a component in isolation

Run an example program:

    go run ./examples/dropdown
    go run ./examples/lamps

Each example is a tiny Bubble Tea program wrapping one component. The
`demo` package (`demo.Run` / `demo.Fatal`) factors out the program
boilerplate (alt screen, mouse). To add a new component's dev harness, copy
`examples/_template/` and fill in the placeholders.

## Component gallery (Storybook-like)

Browse and interact with every component variant in one TUI:

    go run ./examples/gallery

Tab toggles focus between the story list (left) and the component (right).

## Snapshot (golden) tests

The `dropdown` package includes a `teatest`-based golden test
(`TestDropdownGolden`) that captures the rendered terminal output and compares
it against `testdata/*.golden`. Regenerate with `-update`:

    go test ./dropdown/ -run TestDropdownGolden -update

## Extraction

To spin this off into its own repository, move this directory to a new repo,
update the module path in `go.mod`, and update import paths in consumers.
No `rig`-internal dependencies exist, so extraction is mechanical.
