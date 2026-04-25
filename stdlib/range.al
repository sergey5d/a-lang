record IntRange with Iterable[Int] {
    start Int
    end Int
    step Int

    impl def iterator() Iterator[Int] = RangeIterator(start = start, end = end, step = step)
}

private class RangeIterator with Iterator[Int] {
    current Int := ?
    end Int := ?
    step Int := ?

    def this(start Int, end Int, step Int) {
        this.current := start
        this.end := end
        this.step := step
    }

    impl def hasNext() Bool {
        if this.step > 0 {
            return this.current < this.end
        }
        return this.current > this.end
    }

    impl def next() Int {
        value Int = this.current
        this.current := this.current + this.step
        value
    }
}

object Range {
    def apply(start Int, end Int) IntRange {
        if start < end {
            return IntRange(start = start, end = end, step = 1)
        }
        return IntRange(start = start, end = end, step = -1)
    }

    def apply(start Int, end Int, step Int) IntRange = IntRange(start = start, end = end, step = step)
}
