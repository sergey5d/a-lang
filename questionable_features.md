# Questionable Features

This file tracks language ideas that are intentionally not supported right now, but may still be worth revisiting later.

## Interface Method Markers

The language used to experiment with a dedicated interface-implementation marker:

```txt
impl Box {
    impl def label() Str = "box"
}
```

Current decision:
- do not support `impl def`
- interface methods inside `impl Type { ... }` blocks use ordinary `def`

Question still worth revisiting later:
- whether a special marker for interface-method overrides/implementations adds enough readability in large types to justify the extra syntax
- whether explicit markers help enough with refactors and intent signaling
- or whether plain `def` plus normal conformance checking is the cleaner long-term surface
