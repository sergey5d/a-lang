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

	result = if first == 5 {
	    Term.println("YES it's 5!!!")
	    6
	} else {
	    Term.println("NONONO")
	    8
	}

	Term.println("result " + result)

	for {
	    if counter < 20 {
	        Term.println("counter " + counter)
	    } else {
	        break
	    }
	    counter += 1
	}

	list = [1, 2, 3, 8]

	for item <- list {
	    Term.println("item " + item)
	}

	Term.println("item end")

	newList = for {
	    item <- list
	    item2 <- list
	} yield {
	    item + item2
	}

	for item <- newList {
        Term.println("item " + item)
    }

	return methodResult
}
