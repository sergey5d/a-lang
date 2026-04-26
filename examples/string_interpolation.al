# EXPECT:
# hello world
# next 3
# money $5
# mix world-4

def main() {
    name Str = "world"
    count Int = 2

    Term.println("hello $name")
    Term.println("next ${count + 1}")
    Term.println("money \$${count + 3}")
    Term.println("mix $name-${count + 2}")
}
