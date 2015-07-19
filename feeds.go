package main

import (
	"crypto/md5"
	"fmt"
	"time"

	"github.com/gorilla/feeds"
)

type Feed struct {
	Title       string    `json:"title"`
	Url         string    `json:"url"`
	CreateAt    time.Time `json:"create_at"` // auto generated
	Desc        string    `json:desc`
	Author      string    `json:"author"`
	Hash        string    `json:"hash"` // auto generated
	ChannelName string    `json:"channel"`
	Id          int       `json:"id"` // auto generated
}

func (f Feed) CalcHash() string {
	s := fmt.Sprintf("%s|%s|%s", f.Title, f.Desc, f.Url)
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func (f Feed) Valid() bool {
	return len(f.Title) > 0 && len(f.Url) > 0 && len(f.Desc) > 0
}

func (f Feed) ToGorillaFeedItem() *feeds.Item {
	return &feeds.Item{
		Title:       f.Title,
		Link:        &feeds.Link{Href: f.Url},
		Description: f.Url,
		Author:      &feeds.Author{f.Author, ""},
		Created:     f.CreateAt,
	}
}
