package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/boltdb/bolt"
)

type Channel struct {
	Name string `json:"name"`
}

func (c *Channel) Save() error {
	if len(c.Name) == 0 {
		return fmt.Errorf("Invalid channel")
	}
	if c, err := GetChannelByName(c.Name); c != nil && err == nil {
		return fmt.Errorf("Channel already exists")
	} else if err != nil {
		return err
	}
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(channelsBucketName)
		if err != nil {
			return err
		}
		key := fmt.Sprintf("channel:%s", c.Name)
		bb, _ := json.Marshal(c)
		if err = b.Put([]byte(key), bb); err != nil {
			return err
		}
		return nil
	})
}

func (c *Channel) AddFeed(f *Feed) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucketName := append(channelBucketNamePrefix, []byte(c.Name)...)
		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}
		keyBuf := new(bytes.Buffer)
		keyBuf.Write([]byte("item:id:"))
		err = binary.Write(keyBuf, binary.BigEndian, int64(-f.Id))
		if err != nil {
			return err
		}
		val, _ := json.Marshal(f)
		err = b.Put(keyBuf.Bytes(), val)
		if err != nil {
			return err
		}

		hashKey := fmt.Sprintf("item:hash:%s", f.CalcHash())
		idVal := fmt.Sprintf("%d", f.Id)
		err = b.Put([]byte(hashKey), []byte(idVal))
		if err != nil {
			return err
		}
		return nil
	})
}

func (c *Channel) HasFeed(f *Feed) (bool, error) {
	found := false
	err := db.Update(func(tx *bolt.Tx) error {
		bucketName := append(channelBucketNamePrefix, []byte(c.Name)...)
		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}

		hashKey := fmt.Sprintf("item:hash:%s", f.CalcHash())
		bb := b.Get([]byte(hashKey))
		if bb != nil {
			found = true
		}
		return nil
	})
	return found, err
}

func (c *Channel) RemoveFeed(f *Feed) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucketName := append(channelBucketNamePrefix, []byte(c.Name)...)
		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}
		key := fmt.Sprintf("item:id:%d", f.Id)
		err = b.Delete([]byte(key))
		if err != nil {
			return err
		}

		hashKey := fmt.Sprintf("item:hash:%s", f.CalcHash())
		err = b.Delete([]byte(hashKey))
		if err != nil {
			return err
		}
		return nil
	})
}

func (c *Channel) GetFeeds(offset int, limit int) ([]*Feed, error) {
	var feeds []*Feed
	err := db.Update(func(tx *bolt.Tx) error {
		bucketName := append(channelBucketNamePrefix, []byte(c.Name)...)
		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}
		cursor := b.Cursor()
		prefix := []byte("item:id:")
		for k, v := cursor.Seek(prefix); bytes.HasPrefix(k, prefix); k, v = cursor.Next() {
			var feed Feed
			err := json.Unmarshal(v, &feed)
			if err != nil {
				return err
			}
			feeds = append(feeds, &feed)
		}
		return nil

	})
	return feeds, err
}

func GetChannelByName(name string) (*Channel, error) {
	var ret *Channel
	err := db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(channelsBucketName)
		if err != nil {
			return err
		}
		key := fmt.Sprintf("channel:%s", name)
		bb := b.Get([]byte(key))
		if bb == nil {
			return nil
		}
		var chn Channel
		json.Unmarshal(bb, &chn)
		ret = &chn
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func GetChannels() ([]*Channel, error) {
	var channels []*Channel
	err := db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(channelsBucketName)
		if err != nil {
			return err
		}
		c := b.Cursor()
		prefix := []byte(fmt.Sprintf("channel:"))
		for k, v := c.Seek(prefix); bytes.HasPrefix(k, prefix); k, v = c.Next() {
			var channel Channel
			err := json.Unmarshal(v, &channel)
			if err != nil {
				return err
			}
			channels = append(channels, &channel)
		}
		return nil
	})
	return channels, err
}
