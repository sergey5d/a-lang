# EXPECT:
# ok

record Tag {
    name Str
}

record Route {
    path Str
}

@Tag(name = "service")
class Service {
    @Tag(name = "field")
    name Str
}

impl Service {
    @Route(path = "/health")
    def init() {
        this.name = "ok"
    }

    @Tag(name = "health")
    def health() Str = this.name
}

def main() Unit {
    println(Service().health())
}
