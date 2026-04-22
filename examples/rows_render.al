# EXPECT:
# 3578
# 246
# 1
# 3578
# 246
# 1
# 3578
# 246
# 1
# 3578
# 246
# 1

object RowOrdering with Ordering[(Int, Int, String)] {
    def compare(left (Int, Int, String), right (Int, Int, String)) Int {
        leftX Int, leftY Int, _ = left
        rightX Int, rightY Int, _ = right

        if leftY > rightY {
            return -1
        }
        if leftY < rightY {
            return 1
        }
        if leftX < rightX {
            return -1
        }
        if leftX > rightX {
            return 1
        }
        return 0
    }
}

def syntax1(rows List[(Int, Int, String)]) Unit {
    lastX := 0
    if _, initialY, _ <- rows.get(0) {
        lastY := initialY
        for x Int, y Int, char String <- rows {
            for lineStep <- Range(y, lastY) {
                Term.println()
            }
            if lastX < x {
                for spaceStep <- Range(lastX, x) {
                    Term.print(" ")
                }
            }
            Term.print(char)
            lastX := x + 1
            lastY := y
        }

        Term.println()
    }
}

def syntax2(rows List[(Int, Int, String)]) Unit {
    lastX := 0
    if _, initialY Int, _ <- rows.get(0) {
        lastY := initialY
        for x, y, char <- rows {
            for lineStep <- Range(y, lastY) {
                Term.println()
            }
            if lastX < x {
                for spaceStep <- Range(lastX, x) {
                    Term.print(" ")
                }
            }
            Term.print(char)
            lastX := x + 1
            lastY := y
        }

        Term.println()
    }
}

def syntax3(rows List[(Int, Int, String)]) Unit {
    lastX := 0
    if first <- rows.get(0) {
        _, initialY Int, _ = first
        lastY := initialY
        for row <- rows {
            x, y, char String = row
            for lineStep <- Range(y, lastY) {
                Term.println()
            }
            if lastX < x {
                for spaceStep <- Range(lastX, x) {
                    Term.print(" ")
                }
            }
            Term.print(char)
            lastX := x + 1
            lastY := y
        }

        Term.println()
    }
}

def syntax4(rows List[(Int, Int, String)]) Unit {
    lastX := 0
    for {
        _, initialY Int, _ <- rows.get(0)
        lastY := initialY
        x, y, char String <- rows
        # just as an example
        xImmutable = x
    } yield {
        for lineStep <- Range(y, lastY) {
            Term.println()
        }
        if lastX < xImmutable {
            for spaceStep <- Range(lastX, xImmutable) {
                Term.print(" ")
            }
        }
        Term.print(char)
        lastX := x + 1
        lastY := y
        char
    }
    Term.println()
}

def main() Unit {
    rows List[(Int, Int, String)] = [
        (0, 0, "1"),
        (0, 1, "2"),
        (0, 2, "3"),
        (1, 1, "4"),
        (1, 2, "5"),
        (2, 1, "6"),
        (2, 2, "7"),
        (3, 2, "8")
    ]
    rows.sort(RowOrdering)

    syntax1(rows)
    syntax2(rows)
    syntax3(rows)
    syntax4(rows)
}
