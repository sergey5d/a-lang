# FAIL_REGEX:
# invalid_match_pattern at .*: runtime type patterns cannot specify generic arguments; use the erased outer type

class Box[T] {
    value T
}

def main() Option[Int] =
    try match Box(7) {
        _ Box[Int] => 1
    }
