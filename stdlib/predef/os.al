object OS with Printer {
    out Printer = ?
    err Printer = ?

    impl def print(value Str...) Unit = ()
    impl def println(value Str...) Unit = ()
    impl def printf(format Str, value Str...) Unit = ()
    impl def panic(value Str...) Unit = ()
}

