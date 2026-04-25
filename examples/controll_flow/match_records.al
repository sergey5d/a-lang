# EXPECT:
# 42-hello
# 0

record Amount {
    count Int
    label Str
}

def main() Int {
    amount Amount = Amount(42, "hello")
    Term.println(match amount {
        Amount(count, label) => count + "-" + label
    })
    0
}
