//
// Generate an RSS feed for my blog.
//
// I get the posts and info about them by examining markdown files.
//
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"sort"
	"time"

	"github.com/horgh/gorse/gorselib"
)

// URI is the url to the blog root.
const URI = "https://blog.summercat.com"

// Post holds information about a post.
type Post struct {
	Title       string
	Description string
	PubDate     time.Time
	URI         string
}

// ByPubDate implements sort.Interface
// Reverse chronologically.
type ByPubDate []Post

func (p ByPubDate) Len() int           { return len(p) }
func (p ByPubDate) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ByPubDate) Less(i, j int) bool { return p[j].PubDate.Before(p[i].PubDate) }

func main() {
	log.SetFlags(0)
	gorselib.SetQuiet(true)

	outputFile := flag.String("output-file", "", "Output XML file to write.")

	flag.Parse()

	if len(*outputFile) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	posts, err := getPosts()
	if err != nil {
		log.Fatal(err)
	}

	rss := gorselib.RSSFeed{}
	rss.Name = "The one and the many"
	rss.URI = URI
	rss.Description = "A personal blog with articles discussing programming, GNU/Linux, technology, and other interests."
	rss.LastUpdateTime = posts[0].PubDate

	for _, post := range posts {
		rss.Items = append(rss.Items, gorselib.RSSItem{
			Title:           post.Title,
			URI:             post.URI,
			Description:     post.Description,
			PublicationDate: post.PubDate,
		})
	}

	if len(rss.Items) > 10 {
		rss.Items = rss.Items[0:10]
	}

	err = gorselib.WriteFeedXML(&rss, *outputFile)
	if err != nil {
		log.Printf("Failed to write XML: %s", err.Error())
		os.Exit(1)
	}
}

// Return posts reverse chronologically.
func getPosts() ([]Post, error) {
	dh, err := os.Open("pages")
	if err != nil {
		return nil, err
	}

	fis, err := dh.Readdir(0)
	if err != nil {
		_ = dh.Close()
		return nil, fmt.Errorf("Readdir: %s", err)
	}

	posts := []Post{}

	for _, fi := range fis {
		if fi.Name()[0] == '.' {
			continue
		}

		// Not a post.
		if fi.Name() == "index.md" {
			continue
		}

		// Should only have regular files.
		postPath := path.Join("pages", fi.Name())

		post, err := getPost(postPath, fi.Name())
		if err != nil {
			_ = dh.Close()
			return nil, fmt.Errorf("Unable to retrieve post: %s: %s", fi.Name(), err)
		}

		posts = append(posts, post)
	}

	err = dh.Close()
	if err != nil {
		return nil, fmt.Errorf("Close: %s", err)
	}

	sort.Sort(ByPubDate(posts))

	return posts, nil
}

func getPost(path, name string) (Post, error) {
	fh, err := os.Open(path)
	if err != nil {
		return Post{}, err
	}

	scanner := bufio.NewScanner(fh)

	post := Post{}

	metadata := map[string]string{}
	metaName := ""
	metaValue := ""

	for scanner.Scan() {
		metaRe := regexp.MustCompile("^META (\\S+) (.*)$")
		matches := metaRe.FindStringSubmatch(scanner.Text())
		if matches != nil {
			if len(metaName) > 0 {
				metadata[metaName] = metaValue
				metaName = ""
				metaValue = ""
			}
			metaName = matches[1]
			metaValue = matches[2]
			continue
		}

		if len(metaName) > 0 {
			if len(scanner.Text()) == 0 {
				metadata[metaName] = metaValue
				metaName = ""
				metaValue = ""
			} else {
				metaValue += " " + scanner.Text()
			}
			continue
		}

		titleRe := regexp.MustCompile("^# (.+)$")
		matches = titleRe.FindStringSubmatch(scanner.Text())
		if matches != nil {
			post.Title = matches[1]
		}
	}

	if scanner.Err() != nil {
		_ = fh.Close()
		return Post{}, fmt.Errorf("Scanner: %s", scanner.Err())
	}

	err = fh.Close()
	if err != nil {
		return Post{}, fmt.Errorf("Close: %s", err)
	}

	desc, ok := metadata["description"]
	if !ok {
		return Post{}, fmt.Errorf("No description found")
	}
	post.Description = desc

	dateRaw, ok := metadata["pubdate"]
	if !ok {
		return Post{}, fmt.Errorf("No pub date found")
	}

	locn, err := time.LoadLocation("America/Vancouver")
	if err != nil {
		return Post{}, err
	}

	pubdate, err := time.ParseInLocation("2006-01-02", dateRaw, locn)
	if err != nil {
		pubdate, err = time.ParseInLocation("2006-01-02 15:04:05", dateRaw, locn)
		if err != nil {
			return Post{}, err
		}
	}

	post.PubDate = pubdate

	post.URI = fmt.Sprintf("%s/%s.html", URI, name)

	return post, nil
}
