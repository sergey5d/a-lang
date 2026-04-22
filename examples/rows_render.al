# SKIP: target sample for row rendering once sort/comparator support is added
#
# Python source:
#   print(rows)
#   rows.sort(key = lambda r: (-r[1], r[0]))
#   print(rows)
#   last_x = 0
#   last_y = rows[0][1]
#   for x, y, char in rows:
#     for _ in range(y, last_y):
#       print()
#     for _ in range(last_x, x):
#       print(" ", end = "")
#     print(char, end = "")
#     last_x = x + 1
#     last_y = y
#   print()
#
# Target a-lang shape:

def syntax1(rows List[(Int, Int, String)]) {

    lastX := 0
    if _, lastY, _ <- rows.get(0) {
        for x Int, y Int, char Rune <- rows {
            for _ <- Range(y, lastY) {
                Term.println()
            }
            for _ <- Range(lastX, x) {
                Term.print(" ")
            }

            Term.print(char)
            lastX := x + 1
            lastY := y
        }

        Term.println()
    }
}

def syntax2(rows List[(Int, Int, String)]) {

    lastX := 0
    if _, lastY Int, _ <- rows.get(0) {
        for x Int, y _, char Rune <- rows {
            for _ <- Range(y, lastY) {
                Term.println()
            }
            for _ <- Range(lastX, x) {
                Term.print(" ")
            }

            Term.print(char)
            lastX := x + 1
            lastY := y
        }

        Term.println()
    }
}

def syntax3(rows List[(Int, Int, String)]) {

    lastX := 0
    for {
        _, lastY Int, _ <- rows.get(0)
         x Int, y _, char Rune <- rows
    } yield {
        for _ <- Range(y, lastY) {
            Term.println()
        }
        for _ <- Range(lastX, x) {
            Term.print(" ")
        }
        Term.print(char)
        lastX := x + 1
        lastY := y
    }

    Term.println()
}

def main() Unit {
    rows = [
        (0, 0, '█'),
        (0, 1, '█'),
        (0, 2, '█'),
        (1, 1, '▀'),
        (1, 2, '▀'),
        (2, 1, '▀'),
        (2, 2, '▀'),
        (3, 2, '▀')
    ]

    Term.println(rows)

    # TODO: requires list sort with comparator / key support.
    #rows = rows.sort(r -> (-r[1], r[0]))
    Term.println(rows)

    syntax1(rows)
    syntax2(rows)
    syntax3(rows)
}
