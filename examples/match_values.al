# EXPECT:
# exact value
# 0

def main() Int {
    exactValue = match 5 {
        4 => "nope"
        5 => "exact value"
        _ => "miss"
    }
    Term.println(exactValue)
    0
}
