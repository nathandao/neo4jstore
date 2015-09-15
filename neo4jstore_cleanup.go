package neo4jstore

import (
	"github.com/jmcvetta/neoism"
)

const dbUrl = "http://neo4j:foobar@localhost:7474"

func Cleanup() {
	db, _ := neoism.Connect(dbUrl)
	cq := neoism.CypherQuery{
		Statement: `MATCH (n) OPTIONAL MATCH (n)-[r]-() DELETE n,r`,
	}
	if err := db.Cypher(&cq); err != nil {
		panic(err)
	}
}
