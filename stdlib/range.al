record IntRange with Iterable[Int] {
    start Int
    end Int
    step Int

    def iterator() Iterator[Int] = RangeIterator(start = start, end = end, step = step)
}

private class RangeIterator with Iterator[Int] {
    private current Int := ?
    private end Int := ?
    private step Int := ?

    def this(start Int, end Int, step Int) {
        this.current := start
        this.end := end
        this.step := step
    }

    def hasNext() Bool {
        if this.step > 0 {
            return this.current < this.end
        }
        return this.current > this.end
    }

    def next() Int {
        value Int = this.current
        this.current := this.current + this.step
        value
    }
}

object Range {
    def (start Int, end Int) IntRange {
        if start < end {
            return IntRange(start = start, end = end, step = 1)
        }
        return IntRange(start = start, end = end, step = -1)
    }

    def (start Int, end Int, step Int) IntRange = IntRange(start = start, end = end, step = step)
}
