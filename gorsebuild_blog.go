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

	"summercat.com/gorse/gorselib"
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

	outputFile := flag.String("output-file", "rss.xml", "Output XML file to write.")
	pagesDir := flag.String("pages-dir", "pages", "Directory containing pages.")

	flag.Parse()

	if len(*outputFile) == 0 {
		log.Printf("You must provide an output file.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if len(*pagesDir) == 0 {
		log.Printf("You must provide a pages directory.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	posts, err := getPosts(*pagesDir)
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
		log.Fatalf("Failed to write XML: %s", err)
	}
}

// Return posts reverse chronologically.
func getPosts(dir string) ([]Post, error) {
	dh, err := os.Open(dir)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := dh.Close()
		if err != nil {
			log.Printf("close: %s: %s", dir, err)
		}
	}()

	fis, err := dh.Readdir(0)
	if err != nil {
		return nil, fmt.Errorf("readdir: %s", err)
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
		postPath := path.Join(dir, fi.Name())

		post, err := getPost(postPath, fi.Name())
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve post: %s: %s", fi.Name(), err)
		}

		posts = append(posts, post)
	}

	sort.Sort(ByPubDate(posts))

	return posts, nil
}

func getPost(path, name string) (Post, error) {
	fh, err := os.Open(path)
	if err != nil {
		return Post{}, err
	}

	defer func() {
		err := fh.Close()
		if err != nil {
			log.Printf("close: %s: %s", name, err)
		}
	}()

	// Read page for title and meta information.

	scanner := bufio.NewScanner(fh)

	metaRe := regexp.MustCompile("^META (\\S+) (.*)$")
	metadata := map[string]string{}
	metaName := ""
	metaValue := ""

	titleRe := regexp.MustCompile("^# (.+)$")
	title := ""

	for scanner.Scan() {
		// New META begins.
		matches := metaRe.FindStringSubmatch(scanner.Text())
		if matches != nil {
			if len(metaName) > 0 {
				metadata[metaName] = metaValue
			}
			metaName = matches[1]
			metaValue = matches[2]
			continue
		}

		// If we're in a meta, then we end at a blank line, or append a line to the
		// meta's value.
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

		// Title.
		matches = titleRe.FindStringSubmatch(scanner.Text())
		if matches != nil {
			title = matches[1]
		}
	}

	if scanner.Err() != nil {
		return Post{}, fmt.Errorf("scanner: %s", scanner.Err())
	}

	// We must always have a title, description, and publication date.

	if len(title) == 0 {
		return Post{}, fmt.Errorf("no title found")
	}

	if len(metadata["description"]) == 0 {
		return Post{}, fmt.Errorf("no description found")
	}

	pubDate, err := time.ParseInLocation("2006-01-02", metadata["pubdate"],
		time.Local)
	if err != nil {
		pubDate, err = time.ParseInLocation("2006-01-02 15:04:05",
			metadata["pubdate"], time.Local)
		if err != nil {
			return Post{}, err
		}
	}

	uri := fmt.Sprintf("%s/%s.html", URI, name)

	return Post{
		Title:       title,
		Description: metadata["description"],
		PubDate:     pubDate,
		URI:         uri,
	}, nil
}
