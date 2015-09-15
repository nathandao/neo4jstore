# Neo4jStore - gorilla/sessions

A session storage backend for [gorilla/sessions](http://www.gorillatoolkit.org/pkg/sessions) - [src](https://github.com/gorilla/sessions).

This implementation is based on the [postgresql implementation](https://github.com/antonlindstrom/pgstore). In fact, I copy the whole pgs backend implementation tests with a few modifications to fit the context of Neo4j.

# Usage

```go
package main()

import(
  "github.com/gorilla/securecookie"
  "github.com/gorilla/sessions"
  "github.com/jmcvetta/neoism"
  "github.com/nathandao/neo4jstore"
)

const(
  DbUrl = "http://user:password@localhost:7474"
  SecretKey = "something very secret"
)

// Fetch new store.
store := NewNeo4jStore("http://user:password@localhost:7474", []byte(SecretKey))

// Get a session.
session, err = store.Get(req, "session-key")
if err != nil {
  log.Error(err.Error())
}

// Add a value.
session.Values["foo"] = "bar"

// Save.
if err = sessions.Save(req, rsp); err != nil {
  t.Fatalf("Error saving session: %v", err)
}

// Delete session.
session.Options.MaxAge = -1
if err = sessions.Save(req, rsp); err != nil {
  t.Fatalf("Error saving session: %v", err)
}
```

## Thanks

I've stolen, borrowed and gotten inspiration from the other backends available - especially pgstore

* [pgstore](https://github.com/antonlindstrom/pgstore)
* [redistore](https://github.com/boj/redistore)
* [mysqlstore](https://github.com/srinathgs/mysqlstore)
* [babou dbstore](https://github.com/drbawb/babou/blob/master/lib/session/dbstore.go)

Thank you all for sharing your code!
