# FAIL_REGEX:
# no_matching_overload at .*: class/record 'SecretBadge' requires an anonymous record with exactly matching field names and types

class SecretBadge {
    name Str
    private code Int
}

impl SecretBadge {
    def init(name Str) {
        this.name = name
        this.code = 99
    }
}

def main() Unit {
    _ SecretBadge = SecretBadge(record {
        name = "Mia"
    })
}
