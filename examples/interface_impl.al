# EXPECT:
# hop
# jump 2
# 0

interface Hopper {
    def hop() Str
}

interface Jumper {
    def jump(steps Int) Str
}

class Rabbit with Hopper, Jumper {
    impl def hop() Str = "hop"

    impl def jump(steps Int) Str = "jump " + steps
}

def main() Int {
    rabbit Hopper = Rabbit()
    jumper Jumper = Rabbit()
    Term.println(rabbit.hop())
    Term.println(jumper.jump(2))
    0
}
