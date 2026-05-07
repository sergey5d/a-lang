# FAIL_REGEX:
# expected end of expression, got IDENT\("Some" @ .*?\)

def main() Option[Int] =
    try match Some(1) Some(x) => x
