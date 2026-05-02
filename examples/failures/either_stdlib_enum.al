# FAIL_REGEX:
# parse stdlib "either.al": expected '\{', got DEF\("def" @ 4:5\)

def main() Unit {
    value Either[Str, Int] = Right(7)
    OS.println(value.isRight())
}
