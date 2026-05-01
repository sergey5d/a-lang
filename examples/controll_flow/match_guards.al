enum OptionX[T] {
    case NoneX
    case SomeX {
        value T
    }
}

def describe(value OptionX[Int]) Int =
    match value {
        SomeX(x) if x > 10 => x
        SomeX(_) => 10
        OptionX.NoneX => 0
    }

def run() Unit {
    OS.println(describe(OptionX.SomeX(12)))
    OS.println(describe(OptionX.SomeX(3)))
    OS.println(describe(OptionX.NoneX()))
}
