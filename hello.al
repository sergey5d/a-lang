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

    def tupleFun() (Int, String) = (1, "fun")

    def tupleFun2() (amount Int, descr String) = (2, "fun2")
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

	tuple1 = adder.tupleFun()
	t1, t2 := tuple1

	Term.println("tuple1", t1, t2)

	tuple2 = adder.tupleFun2()
    t21, t22 := tuple2
    Term.println("tuple2", t21, t22, tuple2.amount, tuple2.descr)

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

    var1, var2, var3 = 1, crapper, "5a7"

    var4 Int, var5 Float, var6 = 120, 100., "5aa7"

    var7, var8 = 800, adder.add(7) + 700

    var9, var10 := 800, adder.add(7) + 700

    Term.println(var1, var2, var3, var4, var5, var5, var6, var7, var8, var9, var10)

    var9, var10 := 801, adder.add(7) + 701

    Term.println(var1, var2, var3, var4, var5, var5, var6, var7, var8, var9, var10)

    bucket Bucket = Bucket(10)

    bucket.print()

    Term.println("bucket.get2 " + bucket.get2())

	return methodResult
}
