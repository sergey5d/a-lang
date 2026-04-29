# EXPECT:
# 2
# 3
# 4
# 4
# 4
# 2
# 4
# 5
# 4
# 10

def applyTwice(f (Int) -> Int, value Int) Int = f(f(value))

def main() Unit {
    inc (Int) -> Int = _ + 1
    items = List(1, 2, 3)
    mapped = items.map(_ + 1)
    str Str = "haha"
    words = [str]
    sizes = words.map(_.size())
    pairs = List(("a", 1), ("bb", 2))
    pairMapped = pairs.map((key, value) -> key.size() + value)
    pairIgnored = pairs.map((_, value) -> value * 2)
    entries = Map("a": 1, "bbb": 2)
    mapMapped = entries.map((key, value) -> key.size() + value)
    tuple4s = List((1, 2, 3, 4), (4, 5, 6, 7))
    tuple4Mapped = tuple4s.map((first, _, third, _) -> first + third)

    Term.println(mapped.get(0).getOr(0))
    Term.println(mapped.get(1).getOr(0))
    Term.println(mapped.get(2).getOr(0))
    Term.println(applyTwice(inc, 2))
    Term.println(sizes.get(0).getOr(0))
    Term.println(pairMapped.get(0).getOr(0))
    Term.println(pairMapped.get(1).getOr(0))
    Term.println(mapMapped.get(1).getOr(0))
    Term.println(pairIgnored.get(1).getOr(0))
    Term.println(tuple4Mapped.get(1).getOr(0))
}
