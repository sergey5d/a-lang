# EXPECT:
# exact tuple
# exact tuple 2-3
# exact tuple 4-5
# tuple 1 2
# 0

def main() Int {
    exactPair = match (1, 2) {
        (1, 2) => "exact tuple"
        _ => "miss"
    }
    OS.println(exactPair)

    exactPair2 = match (2, 3) {
        (a Int, b Int) => "exact tuple " + a + "-" + b
        _ => "miss"
    }
    OS.println(exactPair2)

    exactPair3 = match (4, 5) {
        (a, b Int) => "exact tuple " + a + "-" + b
        _ => "miss"
    }
    OS.println(exactPair3)

    pair = (1, 2)
    match pair {
        (left, right) => {
            OS.println("tuple " + left + " " + right)
        }
    }

    0
}
