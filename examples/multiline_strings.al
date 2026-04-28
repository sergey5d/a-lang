# EXPECT:
#
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

    text2 = """Another
multi-line
text"""

}
