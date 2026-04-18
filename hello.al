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

def suckItAll2(val String) String {
    val + " - hehe 2"
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
	Term.println(suckItAll2("haha 2"))

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

    crapper := 0
	for item <- list {
	    Term.println("item " + item)
	    crapper += item
	}

	Term.println("item end " + crapper)

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
