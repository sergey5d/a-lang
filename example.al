class Counter {
	private count Int := deferred

	def init(count Int) {
		this.count = count
	}

	def inc() Int {
		this.count += 1
		return this.count
	}
}

seed Int = 1

def run(input Int) Int {
	counter Counter = Counter(input)
	if input > 0 {
		return counter.inc()
	}
	return seed
}
