# EXPECT:
# hop
# jump 3
# true
# true
# true

interface Hopper {
    def hop() Str
}

interface Jumper {
    def jump(steps Int) Str
}

interface Acrobat with Hopper, Jumper {
}

class Rabbit with Acrobat {
    def hop() Str = "hop"

    def jump(steps Int) Str = "jump " + steps
}

def main() Unit {
    rabbit = Rabbit()
    Term.println(rabbit.hop())
    Term.println(rabbit.jump(3))
    Term.println(rabbit is Acrobat)
    Term.println(rabbit is Hopper)
    Term.println(rabbit is Jumper)
}
