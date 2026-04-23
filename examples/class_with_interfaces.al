# EXPECT:
# hop
# jump 3
# true
# true

interface Hopper {
    def hop() Str
}

interface Jumper {
    def jump(steps Int) Str
}

class Rabbit with Hopper, Jumper {
    def hop() Str = "hop"

    def jump(steps Int) Str = "jump " + steps
}

def main() Unit {
    rabbit = Rabbit()
    Term.println(rabbit.hop())
    Term.println(rabbit.jump(3))
    Term.println(rabbit is Hopper)
    Term.println(rabbit is Jumper)
}
