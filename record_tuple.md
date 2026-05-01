# Record / Tuple Conversion Note

## Idea

Instead of keeping a dedicated positional anonymous-record constructor like:

```txt
record(mergedP, mergedQ, mergedAncestor)
```

we could allow a plain tuple to convert to an anonymous record when the record
shape is already known from context and the element types match.

That would make this valid:

```txt
return (mergedP, mergedQ, mergedAncestor)
```

when the expected return type is something like:

```txt
{ foundP Option[TreeNode], foundQ Option[TreeNode], ancestor Option[TreeNode] }
```

## Why this is attractive

- removes a special-purpose `record(...)` syntax
- keeps tuples as the compact positional literal form
- keeps records as the named structural type form
- works well in:
  - typed bindings
  - function arguments
  - return expressions

Examples:

```txt
value { a Int, b Str } = (1, "x")
```

```txt
def run() { a Int, b Str } = (1, "x")
```

```txt
takePair((1, "x"))
```

where `takePair` expects:

```txt
{ a Int, b Str }
```

## Main concerns

- it makes tuple/record boundaries more contextual
- field order becomes important for record construction in these cases
- it may blur the difference between positional products and named products

## Recommended restrictions

If we do this, it should stay narrow:

- conversion only when the expected record shape is known
- arity must match exactly
- element types must match exactly or be assignable
- no implicit conversion for unconstrained tuple expressions
- preferably one-way only: tuple -> record

## Recommendation

This is probably cleaner than keeping a separate `record(...)` positional
constructor, as long as the conversion stays strictly contextual.
