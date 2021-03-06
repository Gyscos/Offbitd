package main

import (
	"encoding/json"
	diffbot "github.com/diffbot/diffbot-go-client"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var maxFileLength int = 60

type Source struct {
	Title string
	URL   string

	// TRUE if this source has some unread articles
	NewMessages int

	// Overrides default refresh duration if non-null.
	RefreshPeriod time.Duration

	// List of articles from this source.
	// Not JSON-ified since it's stored separately
	Articles []*Article `json:"-"`

	// Mainly used by html template. No need to store in json.
	SaneTitle string `json:"-"`

	SyncNeeded chan *Article `json:"-"`
}

func makeSource(url string) *Source {
	source := &Source{URL: url, Title: url}
	source.prepare()

	return source
}

func (s *Source) prepare() {
	s.SaneTitle = sanify(s.Title)
	s.SyncNeeded = make(chan *Article)

	go s.syncArticlesOnDisk()

	if fileExists(s.getDataDir()) {
		s.loadArticles()
	} else {
		err := os.Mkdir(s.getDataDir(), os.ModeDir|0700)
		if err != nil {
			log.Println("Error creating source directory:", err)
		}
	}

}

// Phase 1 - Load article list from storage
func (s *Source) loadArticles() {
	files, err := ioutil.ReadDir(s.getDataDir())
	if err != nil {
		log.Println("Could not open articles for source "+s.Title+":", err)
	}

	for _, file := range files {
		body, err := ioutil.ReadFile(s.getDataDir() + "/" + file.Name())
		if err != nil {
			log.Println("Cannot read article file:", err)
			continue
		}

		var article Article
		err = json.Unmarshal(body, &article)
		if err != nil {
			log.Println("Cannot unmarshal article:", err)
			continue
		}
		s.Articles = append(s.Articles, &article)
	}
}

func (s *Source) getDataDir() string {
	return "data/" + sanify(s.Title)
}

// Select the articles from the list not corresponding to an existing article.
// Uses URLs to compare them.
func (s *Source) filterNewArticles(items []FrontPageItem) []FrontPageItem {
	var result []FrontPageItem

	for _, item := range items {
		ok := true
		for _, article := range s.Articles {
			if article.Url == item.URL {
				ok = false
				break
			}
		}
		if ok {
			result = append(result, item)
		}
	}

	return result
}

func (s *Source) setURL(url string) {
	s.URL = url
}

func (s *Source) rename(newTitle string) {
	if s.Title == newTitle {
		return
	}

	oldDataDir := s.getDataDir()
	s.Title = newTitle
	s.SaneTitle = sanify(s.Title)

	err := os.Rename(oldDataDir, s.getDataDir())
	if err != nil {
		log.Println("Error renaming source datadir:", err)
	}
}

func (s *Source) getArticle(url string) *Article {
	for _, a := range s.Articles {
		if a.Url == url {
			return a
		}
	}

	return nil
}

func (s *Source) markArticleRead(url string) {
	article := s.getArticle(url)
	if article == nil {
		log.Println("Error: could not find article " + s.Title + "/" + url)
		return
	}

	if !article.Read {
		s.NewMessages--
		article.Read = true
		s.SyncNeeded <- article
	}
}

func (s *Source) syncArticlesOnDisk() {
	for article := range s.SyncNeeded {

		// Write article to file
		bytes, err := json.Marshal(article)
		if err != nil {
			log.Println("Error marshaling article:", err)
		}
		err = ioutil.WriteFile(s.getDataDir()+"/"+sanify(article.Url), bytes, 0600)
		if err != nil {
			log.Println("Error writing article file:", err)
		}
	}
}

// Phase 2 - Add an article during runtime. Also save it to disk.
func (s *Source) addArticle(rawArticle *diffbot.Article) {
	article := wrapArticle(rawArticle)
	s.Articles = append(s.Articles, article)
	s.NewMessages++
	s.SyncNeeded <- article
}
