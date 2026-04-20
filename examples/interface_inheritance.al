# EXPECT:
# hop
# jump 3
# true
# true
# true

interface Hopper {
    def hop() String
}

interface Jumper {
    def jump(steps Int) String
}

interface Acrobat with Hopper, Jumper {
}

class Rabbit with Acrobat {
    def hop() String = "hop"

    def jump(steps Int) String = "jump " + steps
}

def main() Unit {
    rabbit = Rabbit()
    Term.println(rabbit.hop())
    Term.println(rabbit.jump(3))
    Term.println(rabbit is Acrobat)
    Term.println(rabbit is Hopper)
    Term.println(rabbit is Jumper)
}
