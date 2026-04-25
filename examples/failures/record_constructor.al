# FAIL_REGEX:
# invalid_record_method at .*: record 'Bad': records cannot declare constructors

record Bad {
    value Int

    def this(value Int) {
        this(value = value)
    }
}

def main() Int = 0
