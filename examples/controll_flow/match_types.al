# EXPECT:
# worker 7
# other 3
# 0

interface WorkerLike {
    def doWork() Int
}

class Worker with WorkerLike {
    impl def doWork() Int = 7
}

class Other with WorkerLike {
    impl def doWork() Int = 3
}

def describe(value WorkerLike) {
    match value {
        worker Worker => {
            OS.println("worker " + worker.doWork())
        }
        other Other => {
            OS.println("other " + other.doWork())
        }
        _ => {
            OS.println("unknown")
        }
    }
}

def main() Int {
    describe(Worker())
    describe(Other())
    0
}
