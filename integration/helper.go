package integration

import (
	"fmt"
	"github.com/gocql/gocql"
	"github.com/relops/cqlc/cqlc"
	"log"
	"os"
)

func TestSession(host string, keyspace string) *gocql.Session {
	cluster := gocql.NewCluster(host)
	cluster.Keyspace = keyspace
	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("Could not connect to cassandra: %v", err)
		os.Exit(1)
	}
	return session
}

func ClusterTestSession(host string) *gocql.Session {
	cluster := gocql.NewCluster(host)
	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("Could not connect to cassandra: %v", err)
		os.Exit(1)
	}
	return session
}

// Deprecated in favor of cqlc.Truncate(*gocql.Session, cqlc.Table)
func Truncate(session *gocql.Session, table cqlc.Table) {
	stmt := fmt.Sprintf("truncate %s", table.TableName())
	err := session.Query(stmt).Exec()

	if err != nil {
		log.Fatalf("Could not %s: %v", stmt, err)
		os.Exit(1)
	}
}
