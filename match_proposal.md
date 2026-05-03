# Match Proposal

This note captures the main remaining directions for improving `match`.

## Main Improvement Areas

### 1. Guards

Potential shape:

```txt
match value {
    Some(x) if x > 10 => ...
}
```

This is probably the highest-value next step, because it makes pattern matching much more practical without changing the core model.

## 2. Nested Patterns

Examples:

- tuple inside enum case
- record/class patterns inside other patterns
- deeper destructuring in one shot

This would let `match` express more of the current destructuring story directly inside patterns.

## 3. Generic-Aware Type Patterns

Constructor and extractor patterns already carry substituted field types correctly, but type-pattern matching still needs a clearer generic story, especially for:

- generic classes behind interface-typed values
- generic enums behind wider typed values
- distinguishing `Box[Int]` from `Box[Str]` when the runtime currently does not preserve explicit type arguments on instances

This is now mostly about deciding whether generic type patterns should:

- behave with erased runtime semantics
- inspect payload or field values structurally where possible
- or preserve concrete type arguments on runtime instances

## 4. Unreachable-Case Detection

Examples:

- wildcard case first
- later specific case that can never run

This would improve diagnostics and make `match` feel more complete as a checked language feature.

## 5. Partial-Match Story

`try match` already exists, so the main open question is whether that is the final shape.

Open questions:

- is `try match` the final partial-match syntax?
- should there ever be something like `try match` instead?
- should partial matching get better fallback ergonomics?

This is more about language-shape polish than basic capability.

## 6. Pattern-Lambda / Collection Ergonomics

This is a smaller refinement, but still useful.

Examples already discussed:

```txt
list.map(match _ {
    Some(x) => x + 1
    None => 0
})
```

Possible later shorthand:

```txt
list.map(match {
    Some(x) => x + 1
    None => 0
})
```

This should come after the core `match` model is finished.

## 7. Exhaustiveness Depth

Enum exhaustiveness already exists in a basic form.

Possible next improvements:

- better reporting
- support with nested patterns
- more complete analysis across richer pattern shapes

## Suggested Priority

If we want `match` to feel finished, the best order is probably:

1. guards
2. nested patterns
3. unreachable-case detection
4. generic-aware extraction
5. partial-match polish
6. pattern-lambda sugar

That order gives the biggest practical readability gains first while keeping syntax churn lower.
