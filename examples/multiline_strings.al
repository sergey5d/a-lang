# EXPECT:
# hello
# world
# $name
# done
#
# escaped

def main() {
    text Str = """
hello
world
$name
done
\nescaped
"""

    Term.println(text)
}
