# Object Removal Proposal

## Idea

Remove user-defined `object` declarations from the language and rely on:

- global functions
- global values
- packages/modules for namespacing

This would simplify the language model by reducing the number of top-level declaration kinds.

Instead of:

```txt
object OS {
    def println(value Str) Unit = ...
}
```

the language could use a global value:

```txt
OS PrinterLike = ...
```

Instead of companion-style construction through objects:

```txt
object A {
    def apply(arg Int) A = ...
}
```

the language could use a same-named global function:

```txt
class A {
    ...
}

def A(arg Int) A = ...
```

## Why Remove `object`

- simpler surface language
- fewer overlapping data/value models
- more consistent with a language that already has global functions and global values
- closer to the original goal of being simpler than Scala-like designs

Potentially cleaner mental model:

- `class` = nominal instance type
- `record` = value/data type
- `enum` = tagged sum type
- globals = functions and singleton values

## What This Would Replace

### Singleton utility holders

Example:

```txt
OS.println("hello")
```

could continue to work if `OS` is just a global value rather than a special `object`.

### Companion-style factory helpers

Example:

```txt
class A {
    ...
}

def A(arg1 Int) A = ...
```

This keeps construction syntax concise without requiring a separate `object A`.

## Main Problems / Tradeoffs

### 1. Weaker namespacing

Without `object`, related functions and values are flatter at top level.

`object` naturally groups things like:

```txt
Math.sin()
Math.cos()
```

Without it, the language must rely more on:

- packages
- global naming conventions
- more top-level symbols

### 2. Companion/private access becomes less explicit

If a same-named global function is used as a factory:

```txt
class A {
    private value Int
}

def A(value Int) A = ...
```

the language needs a clear answer:

- can `def A(...)` access private members of class `A`?

If yes:
- this introduces a special privilege rule

If no:
- some companion/factory patterns become weaker or impossible

This is probably the biggest semantic issue in removing `object`.

### 3. Class/function name collisions need well-defined rules

If `A` is both:

- a class name in type position
- a function name in value/call position

the language needs clear name-resolution rules.

This is manageable, but still adds design work.

### 4. Singleton stateful values become less direct

`object` is a natural fit for:

- long-lived shared state
- grouped behavior
- singleton identity

A global mutable value can still model this, but the language loses a dedicated abstraction for it.

## Suggested Direction

If simplification is the goal, this proposal is worth serious consideration.

A plausible end state:

- remove user-defined `object`
- keep global functions
- keep global values
- allow same-named factory functions for classes
- decide explicitly whether same-named factory functions get companion-like private access

## Open Question

The key question is:

Should the language keep `object` as a dedicated singleton/namespace mechanism, or treat it as unnecessary now that:

- global functions exist
- global values exist
- anonymous records exist
- structural data no longer needs object literals

