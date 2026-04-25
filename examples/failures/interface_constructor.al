# FAIL_REGEX:
# invalid_interface_method at .*: interface 'Bad': interfaces cannot declare constructors

interface Bad {
    def this() Int
}

def main() Int = 0
