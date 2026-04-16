class Counter {
	private var count Int

	def init(count Int) {
		this.count = count
	}

	def inc() Int {
		this.count += 1
		return this.count
	}
}

let seed Int = 1

def run(input Int) Int {
	let counter Counter = Counter(input)
	if input > 0 {
		return counter.inc()
	}
	return seed
}

