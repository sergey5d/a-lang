# TODO

## Builtin Descriptor Follow-Ups

- Revisit `Set[T]` / `Map[K, V]` collection API breadth.
  - `iterator`, `map`, `flatMap`, `filter`, `fold`, `reduce`, `exists`, `forAll`, and `forEach` now exist in `stdlib/predef` and are wired through the runtime.

- Revisit `Array[T]` collection API shape.
  - `map`, `exists`, `forAll`, and `forEach` now exist and fit fixed-size arrays reasonably well.
  - `flatMap`, `filter`, `zip`, and `zipWithIndex` still need more thought.
  - In particular, `zipWithIndex` producing another `Array[...]` may be the wrong result shape for a fixed-size collection surface.
  - Decide whether those APIs should instead return `List[...]`, `Iterable[...]`, or be omitted from `Array` entirely.

- Revisit `Option[T]` representation.
  - Current implementation still models `Option` as a class-like builtin for historical/runtime convenience.
  - Long-term direction should likely be an enum-based shape now that `match` exists and enum support is much better.
  - Clean up duplicated builtin declarations such as `Option` that currently exist in both `stdlib/` and `stdlib/predef/`, and settle on one source of truth.
  - Candidate surface shape:

```txt
enum Option[T] {
    case None
    case Some { value T }
}
```

## Syntax Follow-Ups

- Consider block-style trailing lambda syntax for call sites that take a function parameter.
  - Example target shape:

```txt
def fun((x Int, y Int) -> Int)

fun { x, y ->
    x + y
}
```

  - Main question: whether this reads as a natural extension of the current lambda syntax, or adds too much overlap with block expressions and existing `fun(x -> ...)` / `fun((x, y) -> ...)` call forms.
