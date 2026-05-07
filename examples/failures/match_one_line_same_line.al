# FAIL_REGEX:
# expected end of expression, got IDENT\("Some" @ .*?\)

def main() Int =
    match Some(1) Some(x) => x
