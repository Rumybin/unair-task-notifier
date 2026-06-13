package scraper

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
	"github.com/Rumybin/unair-task-notifier/internal/diff"
)

const (
	scrapeTimeout = 30 * time.Second
	maxBodySize  = 2 * 1024 * 1024
)

func FetchTasks(ctx context.Context, client *http.Client, baseURL string) ([]diff.Task, error) {
	dashURL := baseURL + "/my/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dashURL, nil)
	if err != nil {
		return nil, fmt.Errorf("scraper: create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scraper: GET %s: %w", dashURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scraper: %s returned status %d", dashURL, resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, maxBodySize)
	return parseTasksPage(limited)
}

func parseTasksPage(r io.Reader) ([]diff.Task, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("scraper: parse HTML: %w", err)
	}
	var tasks []diff.Task
	var currentTimestamp int64
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "div" {
				var region string
				for _, attr := range n.Attr {
					if attr.Key == "data-region" {
						region = attr.Val
					}
					if attr.Key == "data-timestamp" && region == "event-list-content-date" {
						ts, err := strconv.ParseInt(attr.Val, 10, 64)
						if err == nil {
							currentTimestamp = ts
						}
					}
				}
			}
			if n.Data == "div" {
				var isEventItem bool
				for _, attr := range n.Attr {
					if attr.Key == "data-region" && attr.Val == "event-list-item" {
						isEventItem = true
					}
				}
				if isEventItem {
					task, ok := parseTaskItem(n, currentTimestamp)
					if ok {
						tasks = append(tasks, task)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return tasks, nil
}

func parseTaskItem(n *html.Node, ts int64) (diff.Task, bool) {
	var task diff.Task
	if ts > 0 {
		task.DueDate = time.Unix(ts, 0).UTC()
	}
	var eventNode *html.Node
	var searchAnchor func(*html.Node)
	searchAnchor = func(node *html.Node) {
		if eventNode != nil {
			return
		}
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, attr := range node.Attr {
				if attr.Key == "data-action" && attr.Val == "view-event" {
					eventNode = node
					return
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			searchAnchor(c)
		}
	}
	searchAnchor(n)
	if eventNode == nil {
		log.Printf("scraper: warning: skip item, no a[data-action=view-event]")
		return task, false
	}
	for _, attr := range eventNode.Attr {
		switch attr.Key {
		case "data-event-id":
			task.ID = attr.Val
		case "href":
			task.TaskURL = attr.Val
		case "title":
			task.Title = strings.TrimSuffix(attr.Val, " is due")
		}
	}
	if task.ID == "" {
		log.Printf("scraper: warning: skip item, no data-event-id")
		return task, false
	}
	// Cari small.text-end.text-nowrap untuk jam
	var searchTime func(*html.Node)
	searchTime = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "small" {
			var classes []string
			for _, attr := range node.Attr {
				if attr.Key == "class" {
					classes = strings.Fields(attr.Val)
				}
			}
			hasTextEnd := false
			hasTextNowrap := false
			for _, cls := range classes {
				if cls == "text-end" {
					hasTextEnd = true
				}
				if cls == "text-nowrap" {
					hasTextNowrap = true
				}
			}
			if hasTextEnd && hasTextNowrap && node.FirstChild != nil {
				timeStr := strings.TrimSpace(node.FirstChild.Data)
				parts := strings.Split(timeStr, ":")
				if len(parts) == 2 {
					h, err1 := strconv.Atoi(parts[0])
					m, err2 := strconv.Atoi(parts[1])
					if err1 == nil && err2 == nil && h >= 0 && h < 24 && m >= 0 && m < 60 {
						task.DueDate = time.Date(
							task.DueDate.Year(), task.DueDate.Month(), task.DueDate.Day(),
							h, m, 0, 0, time.UTC,
						)
					}
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			searchTime(c)
		}
	}
	searchTime(n)
	// Cari div.event-name-container > small.mb-0
	var searchCourse func(*html.Node)
	searchCourse = func(node *html.Node) {
		if task.CourseName != "" {
			return
		}
		if node.Type == html.ElementNode && node.Data == "div" {
			var classes []string
			for _, attr := range node.Attr {
				if attr.Key == "class" {
					classes = strings.Fields(attr.Val)
				}
			}
			for _, cls := range classes {
				if cls == "event-name-container" {
					for c := node.FirstChild; c != nil; c = c.NextSibling {
						if c.Type == html.ElementNode && c.Data == "small" {
							var smallClasses []string
							for _, a := range c.Attr {
								if a.Key == "class" {
									smallClasses = strings.Fields(a.Val)
								}
							}
							for _, sc := range smallClasses {
								if sc == "mb-0" && c.FirstChild != nil {
									task.CourseName = strings.TrimSpace(c.FirstChild.Data)
									return
								}
							}
						}
					}
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			searchCourse(c)
		}
	}
	searchCourse(n)
	return task, true
}

