# Feature Notes

This file captures the main language gaps and near-term design directions.

## Highest Priority

### 1. Match / Pattern Matching

`match` exists now, including:
- enum case patterns
- tuple patterns
- literal/value patterns
- class/record extractor patterns
- simple type patterns

Still missing:
- generic type-aware matching / extraction
- guards
- nested patterns
- unreachable-case detection
- a final decision on statement-vs-expression totality semantics

## Important Next Tier

### 2. Enum Ergonomics

Enums exist, but they still want:
- better generic pattern ergonomics

Now that `match` exists and enum exhaustiveness is checked, the biggest remaining enum work is more expressive generic-pattern support.

### 3. Derived Protocols

Records should eventually support auto-derived protocols.

Likely targets:
- `Eq`
- `Hashed` when all fields are hashable
- maybe `Show` / `Stringify` later

This reduces boilerplate and helps stdlib types feel native.

### 4. Collection / Query APIs

The language now has `for ... yield`, `map`, `flatMap`, `filter`, `fold`, `reduce`, `exists`, `forAll`, and `forEach`, but stdlib collection ergonomics still need growth.

Likely missing methods:
- clearer `Map` indexing ergonomics:
  - `map[key]` should likely act as lookup and return `Option[V]`
  - `map[key] := value` should likely act as set/update
  - this is intentionally different from list/array indexing, where `[]` may still return the element directly
- maybe collection partitioning helpers
- maybe zip / unzip style helpers later

These can mostly live in the stdlib, but may still need runtime support in places.

## Medium Priority

### 5. Operator Overloading

Operator overloading now exists and is mainly intended for compact value-oriented declared types such as:
- numeric-like wrappers
- vectors / matrices / geometry values
- record/class domain values like `Money`, `Distance`, or `Duration`
- interface-driven abstractions that want symbolic operators over implementing types

Constraint:
- keep it same-line only; no newline-based implicit body after `:`

Finalized policy:
- operator overloading is limited to interfaces, classes, records, and enums
- objects do not participate
- top-level functions do not participate

### 6. Module / Visibility Polish

Current package/import support is usable.

Still open:
- decide what to do with package variables and package-scoped methods
- private by default vs public by default
- whether package variables should ever be exposed through imports
- whether package-scoped methods should be exposed by default or require an explicit rule
- whether object members should ever be importable directly, for example importing `OS.println`-style names without importing the whole object surface
- if both a wide package import and a renamed selective import target the same package, the wide import should come first and the `as` import should come after it

## Longer-OS Ideas

### 9. Result / Either Style Error Values

`Option`, `Result`, `Either`, and `<-` short-circuit extraction now exist.

Still open:
- whether there should also be a Rust-style `?`-like propagation form
- whether failure conversion between result families should ever be supported
- whether the current `Unwrappable[T]` surface is enough, or needs a richer protocol later
- whether custom-shape early exit should move from single-binding `guard` syntax toward a block form such as:
  - `guard { a <- b; c <- d } else { ... }`
  - `guard { a <- b; c <- d } fail { ... }`
- if a block-style `guard` is added:
  - whether successful `<-` bindings stay visible after the guard block
  - whether only `<-` failures should trigger the fallback block
  - whether the fallback block should implicitly return its final value

One possible follow-up is a Rust-style propagation form that:
- extracts the success value from `Ok`
- returns early on `Err`
- possibly allows error conversion through a protocol or conversion rule

Important design constraint:
- expressing "same container family, different success type" is hard without higher-kinded types
- so the current propagation model still relies on compiler help for failure rewrapping

Clarification on "failure conversion":
- this does not necessarily mean superclass/subclass conversion
- the more likely model is wrapper-style conversion into a broader application error type
- example:
  - `readFile() Result[Str, IoError]`
  - enclosing function returns `Result[Int, AppError]`
  - failure conversion would mean allowing `IoError` to be turned into something like `AppError.Io(...)` during propagation

### 10. Smarter Type Narrowing

Later improvements could include:
- better narrowing after `is`
- exhaustiveness analysis
- unreachable branch detection

### 11. Deferred Cleanup

A Go-like `defer` construct is still a possible future feature.

Potential shape:
- `defer close()`
- `defer { cleanup() }`

Main use cases:
- resource cleanup
- structured teardown
- keeping setup and cleanup close together in imperative code

Open questions:
- whether it should run at function exit only
- whether it should support block scope
- how it should interact with `return`, `break`, and runtime errors

## TBD

### Interface Implementation Syntax

`impl def` is implemented now, but the language still needs a final decision on whether this should remain the required syntax for interface method implementations.

Open question:
- keep `impl def` as the explicit implementation marker
- or simplify back to plain `def` and rely on interface conformance checking only

Current leaning:
- keep `impl def` for now because it makes interface implementation explicit
- but this is still a language-design decision, not fully settled

### Match Totality / Partial Match Behavior

`match` now exists, but the language still needs a clear rule for what happens when no case matches.

Open options under discussion:

1. Keep exhaustive `match` expressions
   Shape:
   - `match value { ... }` returns `T`
   - missing cases are a compile error
   Good fit:
   - safest long-term design
   - strongest enum / `Option` ergonomics
   - needs exhaustiveness checking

2. Add `try match` as the partial form
   Possible shapes:
   - `match value { ... }` returns `T`
   - `try match value { ... }` returns `Option[T]`
   Good fit:
   - explicit
   - very compact
   - keeps plain `match` as the total form

3. Add `try match` as the partial form
   Possible shapes:
   - `match value { ... }` returns `T`
   - `try match value { ... }` returns `Option[T]`
   Good fit:
   - explicit
   - keyword-only
   - keeps plain `match` as the total form

Current leaning:
- avoid runtime "no match" exceptions as a normal language outcome
- keep `match` as the exhaustive / total form
- partial matching now uses `try match`

Related lambda-syntax discussion:
- today placeholder-based forms like `list.map(match _ { ... })` work
- possible future shorthand: allow implicit-input match lambdas without `_`
  - block form: `list.map(match { ... })`
  - single-expression shorthand form: `list.map(match: Some(x) => x + 1)`
- this would only make sense in contexts where a one-argument lambda is expected
- open question:
  - is this worthwhile readability improvement
  - or unnecessary contextual magic compared to the explicit `match _ { ... }` form

### Constructor / Companion Design

These are still open design options that need a decision.

Context:
- keep autogenerated primary constructors for the common case
- use same-named objects as privileged companions
- companions may access private members of their class when construction requires it

Open options under discussion:

1. Keep autogenerated constructors and add a way to make the generated primary constructor private
   Possible shape:
   - `private def this(*) = ?`

2. Same as above, but using `...` instead of `*`
   Possible shape:
   - `private def this(...) = ?`
   Concern:
   - `...` already means variadic parameters, so this may be misleading

3. Remove user-declared constructors entirely and rely on:
   - autogenerated primary constructors when fields allow them
   - companion object factories when a class has private uninitialized members

4. If a class defines a custom secondary construction path through its companion object, suppress the autogenerated primary constructor
   Concern:
   - this may be too implicit and surprising during refactors

Current leaning:
- autogenerated constructors still make sense
- companion objects are the special construction path for classes that cannot expose a normal public autogenerated constructor
- if explicit control is needed, `private def this(*) = ?` currently looks clearer than the `...` variant

## Suggested Priority Order

1. `match`
2. enum + pattern ergonomics
3. derived `Eq` / `Hashed`
4. stdlib collection/query growth
5. operator overloading
6. anonymous objects
