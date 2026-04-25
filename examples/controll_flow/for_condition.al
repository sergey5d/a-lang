# EXPECT:
# block 0
# block 1
# block 2
# short 0
# short 1
# short 2
# 0

def next(label Str, value Int) Int {
    Term.println(label + " " + value)
    value + 1
}

def main() Int {
    count Int := 0
    for count < 3 {
        Term.println("block " + count)
        count += 1
    }

    shortCount Int := 0
    for shortCount < 3: shortCount = next("short", shortCount)

    0
}
