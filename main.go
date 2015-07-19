package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	gfeeds "github.com/gorilla/feeds"
	"github.com/gorilla/mux"
	"github.com/ngaut/log"
)

var (
	addr   = flag.String("addr", ":10086", "http server listen port")
	dbFile = flag.String("dbfile", ".anyrss.db", "db file path")
)

var (
	// bucket names
	settingsBucketName      = []byte("settings")
	channelsBucketName      = []byte("channels")
	channelBucketNamePrefix = []byte("channel:")
	// const keys
	settingsIdKey = []byte("id")
)

var db *bolt.DB
var idChan chan int

func genGlobalId() int {
	return <-idChan
}

func init() {
	flag.Parse()
	var err error
	db, err = bolt.Open(*dbFile, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	// load global id
	var globalId int
	if err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(settingsBucketName)
		if err != nil {
			return err
		}
		r := b.Get(settingsIdKey)
		if r == nil {
			globalId = 1000
		} else {
			globalId, err = strconv.Atoi(string(r))
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}
	// run gen id routine
	idChan = make(chan int)
	go func(globalId int) {
		for {
			if err := db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket(settingsBucketName)
				globalId += 1
				err := b.Put(settingsIdKey, []byte(strconv.Itoa(globalId)))
				if err != nil {
					return err
				}
				return nil
			}); err != nil {
				log.Fatal(err)
			}
			idChan <- globalId
		}
	}(globalId)
}

// channel handler
// PUT: create new channel
func channelPutHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var channel Channel
	err := decoder.Decode(&channel)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	err = channel.Save()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	b, _ := json.MarshalIndent(channel, "", "  ")
	w.Write(b)
}

// POST: post feed to channel
func channeldPostHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelName := vars["channel"]
	c, err := GetChannelByName(channelName)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	decoder := json.NewDecoder(r.Body)
	var feed Feed
	err = decoder.Decode(&feed)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	if !feed.Valid() {
		w.WriteHeader(500)
		w.Write([]byte("invalid feed"))
		return
	}
	exists, err := c.HasFeed(&feed)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	if exists {
		w.WriteHeader(500)
		w.Write([]byte("feed already exists"))
		return
	}
	feed.Id = genGlobalId()
	feed.CreateAt = time.Now()
	feed.Hash = feed.CalcHash()
	feed.ChannelName = channelName

	err = c.AddFeed(&feed)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	b, _ := json.MarshalIndent(feed, "", "  ")
	w.Write(b)
}

// GET: get channel feed list
func channelGetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelName := vars["channel"]
	c, err := GetChannelByName(channelName)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	if c == nil {
		w.WriteHeader(500)
		w.Write([]byte("no such channel"))
		return
	}

	feeds, err := c.GetFeeds(0, 100)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	b, _ := json.MarshalIndent(feeds, "", "  ")
	w.Write(b)
}

func channelRssHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelName := vars["channel"]
	c, err := GetChannelByName(channelName)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	if c == nil {
		w.WriteHeader(500)
		w.Write([]byte("no such channel"))
		return
	}

	feeds, err := c.GetFeeds(0, 100)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	f := &gfeeds.Feed{
		Title:  channelName,
		Link:   &gfeeds.Link{},
		Author: &gfeeds.Author{},
	}
	var fs []*gfeeds.Item
	for _, feed := range feeds {
		fs = append(fs, feed.ToGorillaFeedItem())
	}
	f.Items = fs
	rss, err := f.ToRss()
	w.Write([]byte(rss))
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/c/{channel:[a-zA-Z0-9]+}", channelGetHandler).Methods("GET")
	r.HandleFunc("/c/{channel:[a-zA-Z0-9]+}", channeldPostHandler).Methods("POST")
	r.HandleFunc("/rss/{channel:[a-zA-Z0-9]+}", channelRssHandler).Methods("GET")
	r.HandleFunc("/c", channelPutHandler).Methods("PUT")
	http.Handle("/", r)
	http.ListenAndServe(*addr, nil)
}
