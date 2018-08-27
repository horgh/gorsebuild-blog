// Generate an RSS feed for my blog.
//
// I get the posts and info about them by examining Markdown files.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/horgh/rss"
	"github.com/pkg/errors"
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

// ByPubDate implements sort.Interface Reverse chronologically.
type ByPubDate []Post

func (p ByPubDate) Len() int           { return len(p) }
func (p ByPubDate) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ByPubDate) Less(i, j int) bool { return p[j].PubDate.Before(p[i].PubDate) }

func main() {
	log.SetFlags(0)

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

	feed := rss.Feed{
		Title:       "The One and the Many",
		Link:        URI,
		Description: "A personal blog with articles discussing programming, GNU/Linux, technology, and other interests.",
		PubDate:     posts[0].PubDate,
	}

	for _, post := range posts {
		feed.Items = append(feed.Items, rss.Item{
			Title:       post.Title,
			Link:        post.URI,
			Description: post.Description,
			PubDate:     post.PubDate,
		})
	}

	if len(feed.Items) > 5 {
		feed.Items = feed.Items[:5]
	}

	err = rss.WriteFeedXML(feed, *outputFile)
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
		if err := fh.Close(); err != nil {
			log.Fatalf("%+v", errors.Wrap(err, "error closing file"))
		}
	}()

	return parsePost(name, fh)
}

func parsePost(
	name string,
	reader io.Reader,
) (Post, error) {
	// Read page for title and meta information.

	scanner := bufio.NewScanner(reader)

	metaRE := regexp.MustCompile("^META (\\S+) (.*)$")
	metadata := map[string]string{}
	var metaName, metaValue string

	titleRE := regexp.MustCompile("^# (.+)$")
	var title string

	for scanner.Scan() {
		// New META begins.
		matches := metaRE.FindStringSubmatch(scanner.Text())
		if matches != nil {
			if len(metaName) > 0 {
				metadata[metaName] = metaValue
			}
			metaName = matches[1]
			metaValue = matches[2]
			if _, ok := metadata[metaName]; ok {
				return Post{}, errors.Errorf("found duplicate metadata: %s", metaName)
			}
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

		if title == "" {
			matches := titleRE.FindStringSubmatch(scanner.Text())
			if matches != nil {
				title = matches[1]
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return Post{}, errors.Wrap(err, "error scanning")
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

	// Filename is x.md. URI should be x.html.
	uri := fmt.Sprintf("%s/%s.html", URI, strings.TrimSuffix(name, ".md"))

	return Post{
		Title:       title,
		Description: metadata["description"],
		PubDate:     pubDate,
		URI:         uri,
	}, nil
}
