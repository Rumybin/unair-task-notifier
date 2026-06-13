package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
	"github.com/Rumybin/unair-task-notifier/internal/diff"
)

const (
	scrapeTimeout = 30 * time.Second
	maxBodySize   = 2 * 1024 * 1024
)

// FetchTasks mengambil daftar tugas dari halaman dashboard (/my/).
// Karena block Timeline di Moodle 4.x di-render oleh JavaScript,
// kita ambil data dari block Calendar yang sudah statis di HTML
// (tabel #month-detailed dengan event-item di dalam td.day).
// Course name diambil dari Recently accessed items jika tersedia.
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
	return parseDashboardPage(limited)
}

// parseDashboardPage mengekstrak tugas dari block Calendar dan Recently accessed items.
func parseDashboardPage(r io.Reader) ([]diff.Task, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("scraper: parse HTML: %w", err)
	}

	// Ambil mapping URL -> courseName dari Recently accessed items
	courseNameMap := extractCourseNamesFromRecentItems(doc)

	// Parse event dari Calendar table
	events := parseCalendarTable(doc)

	// Gabungkan
	var tasks []diff.Task
	for _, e := range events {
		task := diff.Task{
			ID:      e.eventID,
			Title:   e.title,
			TaskURL: e.url,
			DueDate: time.Unix(e.timestamp, 0).UTC(),
		}
		if cn, ok := findCourseNameForURL(e.url, courseNameMap); ok {
			task.CourseName = cn
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// calendarEvent mewakili satu event dari tabel calendar.
type calendarEvent struct {
	eventID   string
	title     string
	url       string
	timestamp int64
}

// parseCalendarTable mengekstrak event dari <table id="month-detailed-...">
// dengan mencari td.day yang memiliki data-day-timestamp, lalu
// mengambil li[data-region="event-item"] di dalamnya.
func parseCalendarTable(doc *html.Node) []calendarEvent {
	var events []calendarEvent

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "td" {
			var isDay bool
			var dayTimestamp int64
			for _, attr := range n.Attr {
				if attr.Key == "class" {
					for _, cls := range strings.Fields(attr.Val) {
						if cls == "day" {
							isDay = true
						}
					}
				}
				if attr.Key == "data-day-timestamp" {
					ts, err := strconv.ParseInt(attr.Val, 10, 64)
					if err == nil {
						dayTimestamp = ts
					}
				}
			}
			if isDay && dayTimestamp > 0 {
				events = append(events, extractEventsFromDayCell(n, dayTimestamp)...)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return events
}

// extractEventsFromDayCell mengambil li[data-region="event-item"] dari td.day.
func extractEventsFromDayCell(td *html.Node, dayTimestamp int64) []calendarEvent {
	var events []calendarEvent

	var findEvents func(*html.Node)
	findEvents = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "li" {
			var isEvent bool
			for _, attr := range n.Attr {
				if attr.Key == "data-region" && attr.Val == "event-item" {
					isEvent = true
				}
			}
			if isEvent {
				event := calendarEvent{timestamp: dayTimestamp}
				var findAnchor func(*html.Node)
				findAnchor = func(anchorNode *html.Node) {
					if event.eventID != "" {
						return
					}
					if anchorNode.Type == html.ElementNode && anchorNode.Data == "a" {
						var hasAction bool
						for _, attr := range anchorNode.Attr {
							if attr.Key == "data-action" && attr.Val == "view-event" {
								hasAction = true
							}
							if attr.Key == "data-event-id" {
								event.eventID = attr.Val
							}
							if attr.Key == "href" {
								event.url = attr.Val
							}
							if attr.Key == "title" {
								event.title = strings.TrimSuffix(attr.Val, " is due")
							}
						}
						if hasAction {
							return
						}
					}
					for c := anchorNode.FirstChild; c != nil; c = c.NextSibling {
						findAnchor(c)
					}
				}
				findAnchor(n)
				if event.eventID != "" {
					events = append(events, event)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findEvents(c)
		}
	}
	findEvents(td)

	return events
}

// extractCourseNamesFromRecentItems mengambil mapping course name
// dari block Recently accessed items yang statis di HTML.
func extractCourseNamesFromRecentItems(doc *html.Node) map[string]string {
	names := make(map[string]string)

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			var hasRegion bool
			for _, attr := range n.Attr {
				if attr.Key == "data-region" && attr.Val == "recentlyaccesseditems-view-content" {
					hasRegion = true
				}
			}
			if hasRegion {
				var parseItems func(*html.Node)
				parseItems = func(node *html.Node) {
					if node.Type == html.ElementNode && node.Data == "div" {
						var isCard bool
						for _, attr := range node.Attr {
							if attr.Key == "class" {
								for _, cls := range strings.Fields(attr.Val) {
									if cls == "card" {
										isCard = true
									}
								}
							}
						}
						if isCard {
							extractRecentItem(node, names)
						}
					}
					for c := node.FirstChild; c != nil; c = c.NextSibling {
						parseItems(c)
					}
				}
				parseItems(n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return names
}

// extractRecentItem mengekstrak URL dan course name dari satu item
// di Recently accessed items.
func extractRecentItem(cardNode *html.Node, names map[string]string) {
	var itemURL string
	var courseName string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" && itemURL == "" {
					itemURL = attr.Val
				}
			}
		}
		if n.Type == html.TextNode && strings.TrimSpace(n.Data) != "" {
			text := strings.TrimSpace(n.Data)
			if courseName == "" && strings.Contains(text, " - ") {
				courseName = text
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(cardNode)

	if itemURL != "" && courseName != "" {
		names[itemURL] = courseName
	}
}

// findCourseNameForURL mencoba mencari course_name untuk URL tugas.
func findCourseNameForURL(taskURL string, courseNameMap map[string]string) (string, bool) {
	if cn, ok := courseNameMap[taskURL]; ok {
		return cn, true
	}
	return "", false
}
