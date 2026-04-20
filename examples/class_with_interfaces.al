# EXPECT:
# hop
# jump 3
# true
# true

interface Hopper {
    def hop() String
}

interface Jumper {
    def jump(steps Int) String
}

class Rabbit with Hopper, Jumper {
    def hop() String = "hop"

    def jump(steps Int) String = "jump " + steps
}

def main() Unit {
    rabbit = Rabbit()
    Term.println(rabbit.hop())
    Term.println(rabbit.jump(3))
    Term.println(rabbit is Hopper)
    Term.println(rabbit is Jumper)
}
