# EXPECT:
# valid0 true
# loc0 0:0
# valid1 true
# loc1 0:2
# valid2 true
# loc2 0:2
# valid3 true
# loc3 1:1
# valid4 false
# loc4 -1:-1
# 0

record Location {
    docId Int
    position Int
}

class Cursor {
    private term Str
    private documents List[Str]
    private current Location := ?
    private valid Bool := false
}

impl Cursor {
    def this(term Str, documents List[Str]) {
        this.term = term
        this.documents = documents
        this.current := Location(-1, -1)
        this.valid := false
        this.seek(Location(-1, -1))
    }

    def isValid() Bool = valid

    def get() Location = current

    def advance() Unit {
        if valid {
            this.seek(Location(current.docId, current.position + 1))
        }
    }

    def seek(location Location) Unit {
        docId Int := 0
        found Bool := false

        if location.docId >= 0 {
            docId := location.docId
        }

        while docId < documents.size() && !found {
            docTerms = documents[docId].split(" ")
            pos Int := 0

            if docId == location.docId && location.position >= 0 {
                pos := location.position
            }

            while pos < docTerms.size() && !found {
                if docTerms[pos] == term {
                    this.current := Location(docId, pos)
                    this.valid := true
                    found := true
                }
                if !found {
                    pos += 1
                }
            }

            if !found {
                docId += 1
            }
        }

        if !found {
            this.current := Location(-1, -1)
            this.valid := false
        }
    }
}

def toBeOrNotToBe() Unit {

    documents List[Str] = List(
        "to be"
    )

    c1 = Cursor("to", documents)
    c2 = Cursor("be", documents)
    
    loc1 := c1.get()
    loc2 := c2.get()

    result List[Int] = []

    while true {

        loc1 := c1.get()
        loc2 := c2.get()

        if loc1.docId > loc2.docId {
            c2.seek(Location(loc1.docId, 0))
        } else if loc1.docId < loc2.docId {
            c1.seek(Location(loc2.docId, 0))
        } else {

            currDocId = loc1.docId
            c2Stack List[Int] = []

            #c2.seek(loc1 with { pos = loc1.pos + 1})

            while c2.isValid() && loc2.docId == currDocId {
                c2Stack.append(loc2.position)
                c2.advance()
                loc2 := c2.get()
            }

            c1Orig := loc1

            failed := false

            while c1.isValid() && loc1.docId == currDocId && c2Stack.size() != 0 {
                unwrap c2Loc <- c2Stack.removeLast() else ()
                if c2Loc < loc1.position {
                    failed := true
                    break
                }
                c1.advance()
                loc1 := c1.get()
            }

            if !failed && c2Stack.isEmpty() && loc1.docId > c1Orig.docId {
                result.append(c1Orig.docId)
            } else {
                c1.seek(Location(loc2.docId, 0))
            }
        }
    }

    for res <- result {
        OS.println("document: " + res)
    }

}

def main() Int {
    documents List[Str] = List(
        "cat dog cat",
        "bird cat",
        "dog"
    )

    cursor = Cursor("cat", documents)

    OS.println("valid0 " + cursor.isValid())
    loc0 = cursor.get()
    OS.println("loc0 " + loc0.docId + ":" + loc0.position)

    cursor.advance()
    OS.println("valid1 " + cursor.isValid())
    loc1 = cursor.get()
    OS.println("loc1 " + loc1.docId + ":" + loc1.position)

    cursor.seek(loc1)
    OS.println("valid2 " + cursor.isValid())
    loc2 = cursor.get()
    OS.println("loc2 " + loc2.docId + ":" + loc2.position)

    cursor.advance()
    OS.println("valid3 " + cursor.isValid())
    loc3 = cursor.get()
    OS.println("loc3 " + loc3.docId + ":" + loc3.position)

    cursor.advance()
    OS.println("valid4 " + cursor.isValid())
    loc4 = cursor.get()
    OS.println("loc4 " + loc4.docId + ":" + loc4.position)

    0
}
