class Adder {
	private base Int

	def init(base Int) {
		this.base = base
	}

	def add(value Int) Int {
		return this.base + value
	}
}

third := "some string"

def suckItAll(val String) String {
    return val + " - hehe"
}

def run() Int {
	first Int = 5
	second Int = 7
	counter Int := first
	counter += second

	boost Int = 3
	addBoost = (value Int) -> value + boost
	lambdaResult Int = addBoost(counter)

	adder Adder = Adder(10)
	methodResult Int = adder.add(lambdaResult)

	Term.println("counter " + counter)
	Term.println("lambda " + lambdaResult)
	Term.println("method " + methodResult)

	Term.println("global " + third)

	Term.println(suckItAll("haha"))

	if first == 5 {
	    Term.println("YES it's 5!!!")
	} else {
	    Term.println("NONONO")
	}

	return methodResult
}
