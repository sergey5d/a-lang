# EXPECT:
# x
# closed
# solo

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

    single = Reader {
        impl def read() Str = "solo"
    }

    OS.println(handler.read())
    handler.close()
    OS.println(single.read())
}
