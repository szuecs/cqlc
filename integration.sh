#!/bin/bash

set -e

PID_FILE=cassandra.pid
STARTUP_LOG=startup.log
ARCHIVE_BASE_URL=http://archive.apache.org/dist/cassandra

for v in 2.0.6 2.0.7 2.0.14
do
	TARBALL=apache-cassandra-$v-bin.tar.gz
	CASSANDRA_DIR=apache-cassandra-$v

	curl -L -O $ARCHIVE_BASE_URL/$v/$TARBALL
	
	if [ ! -f $CASSANDRA_DIR/bin/cassandra ]
	then
   		tar xzf $TARBALL
	fi
	
	CASSANDRA_LOG_DIR=`pwd`/v${v}/log/cassandra
	CASSANDRA_LOG=$CASSANDRA_LOG_DIR/system.log

	mkdir -p $CASSANDRA_LOG_DIR
	: >$CASSANDRA_LOG  # create an empty log file
	
	sed -i -e 's?/var?'`pwd`/v${v}'?' $CASSANDRA_DIR/conf/cassandra.yaml
	sed -i -e 's?/var?'`pwd`/v${v}'?' $CASSANDRA_DIR/conf/log4j-server.properties

	echo "Booting Cassandra ${v}, waiting for CQL listener to start ...."

	$CASSANDRA_DIR/bin/cassandra -p $PID_FILE &> $STARTUP_LOG

	{ tail -n +1 -f $CASSANDRA_LOG & } | sed -n '/Starting listening for CQL clients/q'
	
	PID=$(<"$PID_FILE")

	echo "Cassandra ${v} running (PID ${PID}), about to run test suite ...."

	make test

	echo "Test suite passed against Cassandra ${v}, killing server instance (PID ${PID})"
	
	kill -9 $PID
	rm $PID_FILE
done

for v in 2.1.4
do
	TARBALL=apache-cassandra-$v-bin.tar.gz
	CASSANDRA_HOME=`pwd`/apache-cassandra-$v
	PARENT_DIR=$(echo "$v" | sed 's/-rc[0-9]//')

	curl -L -O $ARCHIVE_BASE_URL/$PARENT_DIR/$TARBALL
	
	if [ ! -f $CASSANDRA_HOME/bin/cassandra ]
	then
   		tar xzf $TARBALL
	fi
	
	CASSANDRA_LOG_DIR=$CASSANDRA_HOME/logs
	CASSANDRA_LOG=$CASSANDRA_LOG_DIR/system.log

	mkdir -p $CASSANDRA_LOG_DIR
	: >$CASSANDRA_LOG  # create an empty log file
	
	sed -i -e 's?/var?'`pwd`/v${v}'?' $CASSANDRA_HOME/conf/cassandra.yaml

	echo "Booting Cassandra ${v}, waiting for CQL listener to start ...."

	CASSANDRA_HOME=$CASSANDRA_HOME $CASSANDRA_HOME/bin/cassandra -p $PID_FILE &> $STARTUP_LOG

	{ tail -n +1 -f $CASSANDRA_LOG & } | sed -n '/Starting listening for CQL clients/q'
	
	PID=$(<"$PID_FILE")

	echo "Cassandra ${v} running (PID ${PID}), about to run test suite ...."

	make test

	echo "Test suite passed against Cassandra ${v}, killing server instance (PID ${PID})"
	
	kill -9 $PID
	rm $PID_FILE
done
