# EXPECT:
# x
# closed

interface Reader {
    def read() Str
}

interface Closer {
    def close() Unit
}

def main() Unit {
    handler = Reader with Closer {
        impl def read() Str = "x"
        impl def close() Unit = OS.println("closed")
    }

    OS.println(handler.read())
    handler.close()
}
