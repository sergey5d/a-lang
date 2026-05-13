# FAIL:
# pub is not allowed on enum members

enum Color {
    pub def label() Str = "red"
    case Red
}
