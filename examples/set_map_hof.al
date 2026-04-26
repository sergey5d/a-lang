# EXPECT:
# set 1
# set 2
# set 3
# pair a 1
# pair b 2
# doubled true 3
# expanded true 6
# mapped 10 20
# expandedValues 1 12
# total 9

def main() {
    seen = Set(1, 2, 3)
    doubled = seen.map((item Int) -> item * 2)
    expanded = seen.flatMap((item Int) -> Set(item, item + 10))
    seen.forEach((item Int) -> Term.println("set " + item))

    values = Map("a" : 1, "b" : 2)
    mapped = values.map((key Str, value Int) -> value * 10)
    expandedValues = values.flatMap((key Str, value Int) -> List(value, value + 10))
    values.forEach((key Str, value Int) -> Term.println("pair " + key + " " + value))

    Term.println("doubled " + doubled.contains(4) + " " + doubled.size())
    Term.println("expanded " + expanded.contains(12) + " " + expanded.size())
    Term.println("mapped " + mapped.get(0).getOr(0) + " " + mapped.get(1).getOr(0))
    Term.println("expandedValues " + expandedValues.get(0).getOr(0) + " " + expandedValues.get(3).getOr(0))

    total := 0
    for item Int <- seen {
        total += item
    }
    for key Str, value Int <- values {
        total += value
    }
    Term.println("total " + total)
}
