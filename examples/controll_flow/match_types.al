# EXPECT:
# worker 7
# other 3
# slacker -1
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

class Slacker with WorkerLike {
    impl def doWork() Int = -1
}

def describe(value WorkerLike) {
    match value {
        worker Worker => {
            OS.println("worker " + worker.doWork())
        }
        other Other => {
            OS.println("other " + other.doWork())
        }
        _ Slacker =>
            OS.println("slacker " + value.doWork())
        _ => {
            OS.println("unknown")
        }
    }
}

def main() Int {
    describe(Worker())
    describe(Other())
    describe(Slacker())
    0
}
