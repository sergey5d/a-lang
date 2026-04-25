package things

interface Named {
    def label() Str
}

class A with Named {
    def label() Str = "A"
}

class B with Named {
    def label() Str = "B"
}

object C {
    def apply(value Int) Int = value + 1
}
