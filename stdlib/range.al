record Range with Iterable[Int] {
    start Int
    end Int

    def iterator() Iterator[Int] = RangeIterator(start = start, end = end)
}

class RangeIterator with Iterator[Int] {
    private current Int := ?
    private end Int := ?

    def this(start Int, end Int) {
        this.current := start
        this.end := end
    }

    def hasNext() Bool = this.current < this.end

    def next() Int {
        value Int = this.current
        this.current := this.current + 1
        value
    }
}
