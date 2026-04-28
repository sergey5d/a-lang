# FAIL_REGEX:
# expected expression on same line after "<-"

def main() Option[Int] {
    value <-
        Some(1)
    Some(value)
}
