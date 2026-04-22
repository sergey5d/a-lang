# Feature Notes

This file captures the main language gaps and near-term design directions.

## Highest Priority

### 1. Match / Pattern Matching

This is the biggest missing expression feature.

Why it matters:
- enums become much more useful
- `Option` becomes more natural
- tuples, records, and destructuring become more complete
- future `Result` / `Either` style APIs get much nicer

Good first scope:
- enum case matching
- tuple patterns
- wildcard `_`
- literal patterns

Good later scope:
- guards
- nested patterns
- record/class destructuring by field
- exhaustiveness checking

### 2. Generics With Type Bounds

Generic bounds are the other major missing feature.

Why it matters:
- stdlib constraints like `Ordering`, `Eq`, `Hashed`
- generic collection APIs
- safer reusable abstractions

This unlocks things like:
- bounded sort helpers
- constrained collection methods
- conditional derivations

## Important Next Tier

### 3. Enum Ergonomics

Enums exist, but they still want:
- pattern matching support
- eventual exhaustiveness checking

Once `match` exists, enums become much more complete.

### 4. Derived Protocols

Records should eventually support auto-derived protocols.

Likely targets:
- `Eq`
- `Hashed` when all fields are hashable
- maybe `Show` / `Stringify` later

This reduces boilerplate and helps stdlib types feel native.

### 5. Collection / Query APIs

The language now has `for ... yield`, `map`, `flatMap`, and `forEach`, but stdlib collection ergonomics still need growth.

Likely missing methods:
- `filter`
- `fold` / `reduce`
- `any`
- `all`

These can mostly live in the stdlib, but may still need runtime support in places.

## Medium Priority

### 6. Operator Overloading

This is not a core blocker, but it would help:
- numeric wrappers
- small value types
- vector-like records
- domain-specific types

### 7. Anonymous Objects / Object Literals

Still useful for:
- one-off adapters
- inline protocol implementations
- small configuration objects
- temporary helper instances

There is already an example reminder for this idea.

### 8. Module / Visibility Polish

Current package/import support is usable, but possible future additions include:
- import aliases
- selective imports
- explicit exports later if needed

## Longer-Term Ideas

### 9. Result / Either Style Error Values

`Option` exists, but a richer success/error enum would likely be useful later.

This becomes much more attractive once `match` exists.

### 10. Smarter Type Narrowing

Later improvements could include:
- better narrowing after `is`
- exhaustiveness analysis
- unreachable branch detection

## Suggested Priority Order

1. `match`
2. generic bounds
3. enum + pattern ergonomics
4. derived `Eq` / `Hashed`
5. stdlib collection/query growth
6. operator overloading
7. anonymous objects
