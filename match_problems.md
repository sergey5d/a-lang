# Match Problems

This note captures the main remaining work for `match` after the current generic type-pattern erasure decision.

## 1. Nested Patterns

The biggest remaining expressive gap is nested patterns.

Examples:

- enum case inside another constructor pattern
- tuple pattern inside enum/class/record pattern arguments
- class/record extractor patterns nested inside other patterns

This is the main feature still missing if `match` is meant to cover most structured destructuring use cases directly.

## 2. Unreachable-Case Detection

The checker handles basic exhaustiveness, but it still does not report unreachable later cases.

Examples:

- `_` first, then more specific cases after it
- `SomeX(_)` before `SomeX(x) if ...`
- duplicate case coverage that makes later branches impossible

This would make diagnostics much better and help `match` feel more complete.

## 3. Final Totality Story

The intended direction is mostly clear:

- `match` should stay exhaustive / total
- `try match` should stay partial
- plain `match` should not fall back to runtime "no match" behavior

But this should still be treated as a final language-design decision and documented clearly as part of the finished surface.

## 4. Deeper Exhaustiveness

Enum exhaustiveness exists, but it is still fairly shallow.

Remaining work:

- nested exhaustiveness
- stronger missing-case reporting
- better interaction with richer future pattern forms

This becomes more important once nested patterns are added.

## 5. Pattern-Lambda Sugar

Core `match` already works in placeholder-lambda form:

```txt
list.map(match _ {
    SomeX(x) => x + 1
    NoneX => 0
})
```

Possible later shorthand:

```txt
list.map(match {
    SomeX(x) => x + 1
    NoneX => 0
})
```

This is not a correctness blocker, but it is still open ergonomics work.

## 6. Generic Type-Pattern Policy

This part is now mostly settled:

- extractor patterns are statically generic-aware
- runtime type patterns are erased
- generic arguments inside runtime type patterns are rejected

What still remains is mostly documentation and examples, so the rule feels deliberate rather than incidental.
