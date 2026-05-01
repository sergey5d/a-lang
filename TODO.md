# TODO

## Builtin Descriptor Follow-Ups

- Revisit `Set[T]` / `Map[K, V]` collection API breadth.
  - `iterator`, `map`, `flatMap`, `filter`, `fold`, `reduce`, `exists`, `forAll`, and `forEach` now exist in `stdlib/predef` and are wired through the runtime.

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
