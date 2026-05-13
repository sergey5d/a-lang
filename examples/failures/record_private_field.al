# invalid_record_field at .*: record 'Point' cannot declare private field 'secret'

record Point {
    x Int
    priv secret Int
}

def main() Int = 0
