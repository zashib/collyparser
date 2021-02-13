package main

import (
	"os"
	"strings"
	"unicode"

	"github.com/codingsince1985/geo-golang"
	"github.com/codingsince1985/geo-golang/google"
	"github.com/gocolly/colly"
)

const (
	branchSelector = "div.row.location-list-row"
	aboutSelector  = `//li[@class="camp-menu-item"]//a[contains(text(), "About")]`
	staffSelector  = `div.field-prgf-2c-left.block-description--text.col-12.col-lg`
)

type Person struct {
	Name     string
	Position string
	Email    string
	Phone    string
}

type Branch struct {
	Name      string
	Borough   string
	Address   string
	Phone     string
	Longitude float64
	Latitude  float64
	Staff     []Person
}

func main() {
	branches := make(map[string]Branch)
	var branchesTotal []Branch

	// Instantiate default collector
	c := colly.NewCollector(
		// Visit only certain domain
		colly.AllowedDomains("ymcanyc.org"),
		colly.Async(true),
	)

	// Get branches
	c.OnHTML(branchSelector, func(e *colly.HTMLElement) {
		e.ForEach("div.location-list-item", func(_ int, el *colly.HTMLElement) {

			lon, lat := getLocation(el.ChildText(
				"div.field-location-direction"),
				google.Geocoder(os.Getenv("GOOGLE_API_KEY")),
			)

			branch := Branch{
				Name:      el.ChildText("h2.location-item--title.card-type--branch"),
				Borough:   el.ChildText("div.field-borough"),
				Address:   el.ChildText("div.field-location-direction"),
				Phone:     el.ChildText("div.field-location-phone.field-item > a"),
				Longitude: lon,
				Latitude:  lat,
			}

			// Collect branches to map
			branchURL := e.Request.AbsoluteURL(el.ChildAttr("a.btn-primary", "href"))
			branches[branchURL] = branch

			// Visit branch page
			c.Visit(branchURL)
		})
	})

	// Visit "About" page
	c.OnXML(aboutSelector, func(e *colly.XMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	// Get staff
	c.OnHTML(staffSelector, func(e *colly.HTMLElement) {
		var persons []Person
		e.ForEach("p", func(_ int, el *colly.HTMLElement) {
			if strings.HasSuffix(e.Request.URL.String(), "/about") {
				person := Person{}

				lineFeedSlice := strings.Split(el.Text, "\n")
				person.Name = strings.TrimSpace(lineFeedSlice[0])
				if len(lineFeedSlice) > 1 {
					if lineFeedSlice[len(lineFeedSlice)-1] != "" {
						if unicode.IsDigit([]rune(lineFeedSlice[len(lineFeedSlice)-1])[0]) {
							person.Phone = strings.TrimSpace(lineFeedSlice[len(lineFeedSlice)-1])
						}
					}
				}
				if el.ChildText("a") != "" {
					if len(strings.Split(el.Text, "\n")) > 1 {
						person.Position = strings.ReplaceAll(
							strings.Split(el.Text, "\n")[1],
							el.ChildText("a"),
							"",
						)
					}
				}
				person.Email = el.ChildText("a")
				if person != (Person{}) {
					persons = append(persons, person)
				}
			}
		})

		if currentBranch, found := branches[e.Request.URL.String()]; found {
			currentBranch.Staff = persons
			branches[e.Request.URL.String()] = currentBranch
		}

	})

	c.Visit("https://ymcanyc.org/locations?type&amenities")
	// Wait until threads are finished
	c.Wait()

	for _, v := range branches {
		branchesTotal = append(branchesTotal, v)
	}

}

func getLocation(addr string, geocoder geo.Geocoder) (float64, float64) {
	location, _ := geocoder.Geocode(addr)
	if location != nil {
		return location.Lat, location.Lng
	} else {
		return 0, 0
	}
}
