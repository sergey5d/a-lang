object OS with Printer {
    out Printer = ?
    err Printer = ?

    impl def print(value Str...) Unit = ()
    impl def println(value Str...) Unit = ()
}
