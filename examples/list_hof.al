# EXPECT:
# item 2
# item 4
# item 6
# doubled 2 6
# expanded 1 13
# 0

def main() Int {
    items List[Int] = List(1, 2, 3)
    doubled List[Int] = items.map((item Int) -> item * 2)
    doubled2 List[Int] = items.map(item -> item * 2)
    expanded List[Int] = items.flatMap((item Int) -> List(item, item + 10))

    doubled.forEach((item Int) -> Term.println("item " + item))

    Term.println("doubled " + doubled.get(0).getOr(0) + " " + doubled.get(2).getOr(0))
    Term.println("expanded " + expanded.get(0).getOr(0) + " " + expanded.get(5).getOr(0))
    0
}
