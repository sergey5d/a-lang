class Counter with Eq[Counter] {
	private count Int
	private ticks Int := 0

	def init(count Int) {
		this.count = count
	}

	def equals(other Counter) Bool {
		return this.count == other.count
	}

	def inc() Int {
		this.ticks += 1
		return this.count + this.ticks
	}
}

class Box[T] {
	private value T

	def init(value T) {
		this.value = value
	}

	def apply(mapper T -> T) T {
		return mapper(this.value)
	}
}

def sumFirstTwo(values Array[Int]) Int {
	first Int = values[0]
	second Int = values[1]
	return first + second
}

def run() Bool {
	addOne Int -> Int = x -> x + 1
	addTwo = (x Int) -> {
		y Int = x + 2
		return y
	}

	values = [1, 2, 3]
	values[1] := values[0] + 4

	box Box[Int] = Box[Int](10)
	transformed Int = box.apply(item -> item + 5)

	left Counter = Counter(3)
	right Counter = Counter(3)

	first Int = addOne(6)
	second Int = addTwo(5)
	total Int = sumFirstTwo(values)

	if first == 7 && second == 7 {
		return left == right && total == 6 && transformed == 15
	}

	return false
}

