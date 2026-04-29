# EXPECT:
# hello world
# fmt 7
# pair left 9
# out line
# done

def main() Unit {
    OS.print("hello")
    OS.println(" world")
    OS.printf("fmt %d\n", 7)
    OS.out.printf("pair %s %d\n", "left", 9)
    OS.out.print("out")
    OS.out.println(" line")
    OS.println("done")
}
