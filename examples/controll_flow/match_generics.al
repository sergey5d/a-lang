# EXPECT:
# 0

enum OptionX[T] {
    case NoneX
    case SomeX {
        value T
    }
}

class Box[T] {
    value T
}

def unwrapSome(value OptionX[Int]) Int = match value {
    SomeX(x) => x + 1
    OptionX.NoneX => 0
}

def unwrapBox(value Box[Int]) Int = match value {
    Box(x) => x + 2
}

def main() Int {
    0
}
