# TODO

## Builtin Descriptor Follow-Ups

- Expand `stdlib/predef` collection APIs so `Set` and `Map` are not lagging behind `List`.
  - Decide whether `Set[T]` should expose `map`, `flatMap`, `forEach`, `filter`, and related higher-order helpers.
  - Decide what `Map[K, V]` transformation APIs should look like.
    - Possible directions:
      - `map((key, value) -> X) List[X]`
      - `mapValues(value -> X) Map[K, X]`
      - `mapEntries((key, value) -> (K2, V2)) Map[K2, V2]`

- Revisit `Option[T]` representation.
  - Current implementation still models `Option` as a class-like builtin for historical/runtime convenience.
  - Long-term direction should likely be an enum-based shape now that `match` exists and enum support is much better.
  - Candidate surface shape:

```txt
enum Option[T] {
    case None
    case Some { value T }
}
```

- After the above API decisions, finish wiring runtime/native builtin behavior to the same `predef` descriptor source so checker and interpreter share one builtin API definition path.
