package main

import (
	"gopkg.in/mgo.v2"
	"log"
	"github.com/joeshaw/envdecode"
	"sync"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	mongoSetupOnce sync.Once
	db             *mgo.Session
)

func dialdb() error {
	mongoSetupOnce.Do(func() {
		setupMongoConnection()
	})

	var err error
	log.Println("dialing mongodb: localhost")
	db, err = mgo.Dial(mgConnection)
	return err
}
func closedb() {
	db.Close()
	log.Println("closed database connection")
}

var mgConnection string

func setupMongoConnection() {
	var ms struct {
		ConnectionString string `env:"SP_MONGO_CONNECTION,required"`
	}
	if err := envdecode.Decode(&ms); err != nil {
		log.Fatalln(err)
	}

	mgConnection = ms.ConnectionString
}

type poll struct {
	Options []string
}

func loadOptions() ([]string, error) {
	var options []string
	iter := db.DB("ballots").C("polls").Find(nil).Iter()
	var p poll
	for iter.Next(&p) {
		options = append(options, p.Options...)
	}
	iter.Close()
	return options, iter.Err()
}

func main() {

	if err := dialdb(); err != nil {
		log.Fatalln("Failed to dial mongoDB:", err)
	}
	defer closedb()

	var stopLock sync.Mutex //Protects stop
	stop := false
	stopChan := make(chan struct{}, 1)
	signalChan := make(chan os.Signal, 1)
	go func() {
		<-signalChan
		stopLock.Lock()
		stop = true
		stopLock.Unlock()
		log.Println("Stopping...")
		stopChan <- struct{}{}
		closeConn()
	}()
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Start things
	votes := make(chan string) //chan for votes
	publisherStoppedChan := publishVotes(votes)
	twitterStoppedChan := startTwitterStream(stopChan, votes)
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			closeConn()
			stopLock.Lock()
			if stop {
				stopLock.Unlock()
				return
			}
			stopLock.Unlock()
		}
	}()
	<-twitterStoppedChan
	close(votes)
	<-publisherStoppedChan
}
