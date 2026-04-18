class Adder {
	private base Int

	def init(base Int) {
		this.base = base
	}

	def add(value Int) Int {
	    Term.println("value added int " + value)
		this.base + value
	}

	def add(value String) Int {
	    Term.println("value added string " + value)
    	this.base + 4
    }

	def alterThemAll(items Int...) {
    	    for item <- items {
    	        increased = item + 5
    	        Term.println("increased!", increased)
    	        if increased != 9 {
    	            break
    	        }
    	    }
    }
}

class Bucket {
    base Int := deferred

    def init(a Int) {
        this.base = a
    }

    def print() {
        Term.println("base " + this.base)
    }

    def print2() = Term.println("base " + this.base)

    def get2() Int = 5
 }

third := "some string"

def suckItAll(val String) String {
    return val + " - hehe"
}

def suckItAll2(val String) String {
    val + " - hehe 2"
}

def suckItAll23(val String) Int {
    23
}

def run() Int {
	first Int = 5
	second Int = 7
	third2 = 56
	Term.println("third2 " + third2)

	def x(term Int) = Term.println("xexe" + term)

	def lala() = {
	    Term.println("lala")
	}

	x(5)
	lala()

	counter Int := first
	counter += second

	boost Int = 3
	addBoost = (value Int) -> value + boost

	addBoost2 = value Int -> {
	    value + boost
	}

	Term.println("boost2 " + addBoost2(4))

	lambdaResult Int = addBoost(counter)

	adder = Adder(10)
	adder.alterThemAll(4, 5, 9)
	adder.add(5)
	adder.add("hehe")

	methodResult Int = adder.add(lambdaResult)

	Term.println("counter " + counter)
	Term.println("lambda " + lambdaResult)
	Term.println("method " + methodResult)

	Term.println("global " + third)

	Term.println(suckItAll("haha"))
	Term.println(suckItAll2("haha 2"))
	Term.println("lalala " + suckItAll23("haha 2"))

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
        Term.println("item", item)
    }

    bucket Bucket = Bucket(10)

    bucket.print()

    Term.println("bucket.get2 " + bucket.get2())

	return methodResult
}
