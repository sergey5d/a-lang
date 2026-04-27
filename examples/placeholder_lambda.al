# EXPECT:
# 2
# 3
# 4
# 4
# 4

def applyTwice(f (Int) -> Int, value Int) Int = f(f(value))

def main() Unit {
    inc (Int) -> Int = _ + 1
    items = List(1, 2, 3)
    mapped = items.map(_ + 1)
    str Str = "haha"
    words = [str]
    sizes = words.map(_.size())

    Term.println(mapped.get(0).getOr(0))
    Term.println(mapped.get(1).getOr(0))
    Term.println(mapped.get(2).getOr(0))
    Term.println(applyTwice(inc, 2))
    Term.println(sizes.get(0).getOr(0))
}
