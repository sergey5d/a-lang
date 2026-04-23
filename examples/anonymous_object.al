# SKIP
# Future syntax idea for anonymous objects / object literals.

# Possible direction:
#
# a = object {
#     amount = 5
#     def makeSomething() Str = "xaxa"
# }
#
# Or, if we ever decide bare braces are worth the grammar tradeoff:
#
# a = {
#     amount = 5
#     def makeSomething() Str = "xaxa"
# }
#
# Questions to settle later:
# - should this lower to a synthetic anonymous class + instance?
# - can it capture outer locals?
# - can it implement interfaces?
# - should `object { ... }` be preferred over bare `{ ... }` to avoid ambiguity?
