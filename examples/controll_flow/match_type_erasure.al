# EXPECT:
# list
# class
# record
# interface
# named Zoe
# 0

interface Named {
    def label() Str
}

class Worker with Named {
    name Str
}

impl Worker {
    def label() Str = name
}

record Amount {
    count Int
    label Str
}

def describeList(value List[Int]) {
    match value {
        _ List => OS.println("list")
    }
}

def describeClass(value Worker) {
    match value {
        _ Worker => OS.println("class")
    }
}

def describeRecord(value Amount) {
    match value {
        _ Amount => OS.println("record")
    }
}

def describeInterface(value Named) {
    match value {
        _ Named => OS.println("interface")
    }
}

def describeNamedWorker(value Named) {
    match value {
        worker Worker => OS.println("named " + worker.label())
        _ => OS.println("unknown")
    }
}

def main() Int {
    describeList([1, 2, 3])
    describeClass(Worker("Ada"))
    describeRecord(Amount(7, "usd"))
    describeInterface(Worker("Bob"))
    describeNamedWorker(Worker("Zoe"))
    0
}
