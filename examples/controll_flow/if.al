# EXPECT:
# then
# picked 5
# binding 7
# 0

def main() Int {
    if true {
        Term.println("then")
    } else {
        Term.println("else")
    }

    picked = if false {
        1
    } else {
        5
    }
    Term.println("picked " + picked)

    values = [7]
    if value <- values.get(0) {
        Term.println("binding " + value)
    } else {
        Term.println("binding none")
    }

    0
}
