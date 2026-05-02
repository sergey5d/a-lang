# Class Shape Ideas

This note captures the current class-shape discussion and the main tradeoffs.

## Current Shape

Today the language leans toward putting everything in one place:

```txt
class A with SomeInterface {
    age Int
    name Str
    private malnutritioned Bool := false

    def this(maturity Int) = this(maturity - 15, "5")

    def this(name Str) {
        this.name = name
        age = 15
    }

    def trueAge() Int = age
    def trueName() Str = name
}
```

### Pros

- everything is local
- easy to understand a small type in one pass
- fields, constructors, and methods stay visually connected

### Cons

- class bodies get crowded quickly
- data shape and behavior are mixed together
- large classes become harder to scan

## Split Shape With `impl`

One proposed alternative is:

```txt
class A with SomeInterface {
    age Int
    name Str
    private {
        malnutritioned Bool := false
    }
}

impl A {
    def this(maturity Int) = this(maturity - 15, "5")

    def this(name Str) {
        this.name = name
        age = 15
    }

    def trueAge() Int = age
    def trueName() Str = name
}
```

### Mental Model

- `class` declares storage, fields, and implemented interfaces
- `impl` declares constructors and methods

### Pros

- separates data shape from behavior
- easier to scan medium and large types
- cleaner long-term model if the language adopts `impl` broadly

### Cons

- understanding one type requires looking in two places
- worse for very small classes
- feels most natural only if the language commits to the same model for other type kinds too

## More Formal Split With Signatures In `class`

Another possibility was:

```txt
class A with SomeInterface {
    age Int
    name Str
    private {
        malnutritioned Bool := false
    }

    def this(maturity Int)
    def this(name Str)

    def trueAge() Int
    def trueName() Str
}

impl A {
    def this(maturity Int) = this(maturity - 15, "5")

    def this(name Str) {
        this.name = name
        age = 15
    }

    def trueAge() Int = age
    def trueName() Str = name
}
```

### Verdict

This is likely too heavy:

- duplicates signatures and bodies across two places
- creates mismatch risk
- adds boilerplate
- feels more like header/source duplication than a lightweight language feature

## Readability Take

Readability depends on class size.

### Small classes

The current inline form is usually easier to read because everything is in one place.

### Medium or large classes

The `impl` split is often clearer because it separates:

- structure
- constructors
- behavior

## Coherence Concern

The `impl` direction is strongest only if it becomes a broader language model.

Possible broader rule:

- `class` / `record` / `enum` / `object` declare shape or cases
- `impl Type { ... }` declares behavior

If `impl` is added only for classes while other constructs keep inline methods, the language may become less uniform rather than more uniform.

## Current Leaning

Best candidate so far:

```txt
class A with SomeInterface {
    age Int
    name Str
    private {
        malnutritioned Bool := false
    }
}

impl A {
    def this(maturity Int) = this(maturity - 15, "5")

    def this(name Str) {
        this.name = name
        age = 15
    }

    def trueAge() Int = age
    def trueName() Str = name
}
```

Open questions:

- should multiple `impl A { ... }` blocks be allowed
- can `impl A` live in another file
- should `impl` also apply to `record`, `enum`, and `object`
- how should private access work from `impl`
- should interface conformance remain explicit in `class A with X`
