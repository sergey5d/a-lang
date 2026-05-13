# invalid_enum_field at .*: enum 'Token' cannot declare private field 'secret'

enum Token {
    priv secret Str

    case Ident {
        value Str
        secret = "ident"
    }
}

def main() Int = 0
