package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Film struct {
	Title           string   `bson:"title"`
	ReleaseYear     int      `bson:"releaseYear"`
	Directors       []string `bson:"directors"`
	BoxOffice       float64  `bson:"boxOffice"`
	WorldwideGross  float64  `bson:"worldwideGross"`
	CountryOfOrigin string   `bson:"countryOfOrigin"`
}

var log = logrus.New()

func init() {
	// Configure logrus
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	log.SetLevel(logrus.DebugLevel)
}

func main() {
	log.Info("Starting film data scraping application")

	// Initialize MongoDB
	log.Info("Connecting to MongoDB")
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://replace-admin:replace-password@mongodb:27017"))
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to MongoDB")
	}
	defer client.Disconnect(context.Background())

	collection := client.Database("films").Collection("films")

	// Scrape main page
	log.Info("Starting to scrape main Wikipedia page")
	films := scrapeMainPage()
	log.WithField("count", len(films)).Info("Completed scraping main page")

	// Process each film
	successCount := 0
	failureCount := 0

	for _, film := range films {
		logger := log.WithFields(logrus.Fields{
			"title": film.Title,
			"year":  film.ReleaseYear,
		})

		logger.Info("Inserting film into database")

		// Insert into MongoDB
		_, err = collection.InsertOne(context.Background(), film)
		if err != nil {
			logger.WithError(err).Error("Failed to insert film")
			failureCount++
		} else {
			logger.Info("Successfully inserted film")
			successCount++
		}
	}

	log.WithFields(logrus.Fields{
		"total_processed": len(films),
		"successful":      successCount,
		"failed":          failureCount,
	}).Info("Completed processing all films")
}

func scrapeMainPage() []Film {
	var films []Film

	log.Info("Fetching main Wikipedia page")
	resp, err := http.Get("https://en.wikipedia.org/wiki/List_of_highest-grossing_films")
	if err != nil {
		log.WithError(err).Fatal("Failed to fetch main page")
	}
	defer resp.Body.Close()

	doc, err := htmlquery.Parse(resp.Body)
	if err != nil {
		log.WithError(err).Fatal("Failed to parse main page HTML")
	}

	log.Info("Starting to parse film entries from main page")
	filmCount := 0

	// Start from row 2
	rowNum := 2
	for {
		// Use XPath to find each row
		xpath := fmt.Sprintf("//*[@id=\"mw-content-text\"]/div[1]/table[1]/tbody/tr[%d]", rowNum)
		row := htmlquery.FindOne(doc, xpath)

		if row == nil {
			log.WithField("row", rowNum).Debug("No more rows found")
			break
		}
		log.WithField("row", rowNum).WithField("content", htmlquery.InnerText(row)).Debug("raw row content")

		// Find the film link in the first column
		linkNode := htmlquery.FindOne(row, ".//th[1]//a")
		if linkNode == nil {
			log.WithField("row", rowNum).Warn("No film link found in row")
			rowNum++
			continue
		}

		link := htmlquery.SelectAttr(linkNode, "href")
		if link == "" {
			log.WithField("row", rowNum).Warn("Empty link found in row")
			rowNum++
			continue
		}

		fullURL := "https://en.wikipedia.org" + link
		log.WithFields(logrus.Fields{
			"row": rowNum,
			"url": fullURL,
		}).Info("Scraping individual film page")

		// Extract other data from the row
		rank := strings.TrimSpace(htmlquery.InnerText(htmlquery.FindOne(row, "./td[1]")))
		log.WithField("row", rowNum).WithField("rank", rank).Debug()
		peakRank := strings.TrimSpace(htmlquery.InnerText(htmlquery.FindOne(row, "./td[2]")))
		log.WithField("row", rowNum).WithField("peakRank", peakRank).Debug()
		title := strings.TrimSpace(htmlquery.InnerText(linkNode))
		log.WithField("row", rowNum).WithField("title", title).Debug()
		worldwideGross := strings.TrimSpace(htmlquery.InnerText(htmlquery.FindOne(row, "./td[3]")))
		log.WithField("row", rowNum).WithField("worldwideGross", worldwideGross).Debug()
		year := strings.TrimSpace(htmlquery.InnerText(htmlquery.FindOne(row, "./td[4]")))
		log.WithField("row", rowNum).WithField("year", year).Debug()

		log.WithFields(logrus.Fields{
			"rank":           rank,
			"peakRank":       peakRank,
			"title":          title,
			"worldwideGross": worldwideGross,
			"year":           year,
		}).Debug("Extracted row data")

		film := scrapeFilmPage(fullURL)
		if film != nil {
			// Update film with data from main table
			if yearNum, err := strconv.Atoi(year); err == nil {
				film.ReleaseYear = yearNum
			}
			film.WorldwideGross = float64(extractMoney(worldwideGross))

			films = append(films, *film)
			filmCount++
			log.WithFields(logrus.Fields{
				"title": film.Title,
				"count": filmCount,
			}).Info("Successfully scraped film")
		}

		time.Sleep(100 * time.Millisecond)
		rowNum++
	}

	log.WithField("total_films", filmCount).Info("Completed scraping main page")
	return films
}

func scrapeFilmPage(url string) *Film {
	logger := log.WithField("url", url)

	resp, err := http.Get(url)
	if err != nil {
		logger.WithError(err).Error("Failed to fetch film page")
		return nil
	}
	defer resp.Body.Close()

	doc, err := htmlquery.Parse(resp.Body)
	if err != nil {
		logger.WithError(err).Error("Failed to parse film page HTML")
		return nil
	}

	film := &Film{}

	// Get title
	titleNode := htmlquery.FindOne(doc, "//h1[@id='firstHeading']")
	film.Title = strings.TrimSpace(htmlquery.InnerText(titleNode))
	logger = logger.WithField("title", film.Title)
	logger.Debug("Extracted film title")

	// Get release year from infobox
	yearNode := htmlquery.FindOne(doc, "//th[contains(text(), 'Release date')]/following-sibling::td[1]//li[1]")
	if yearNode != nil {
		yearText := htmlquery.InnerText(yearNode)
		film.ReleaseYear = extractYear(yearText)
		logger.WithField("year", film.ReleaseYear).Debug("Extracted release year")
	}

	// Get directors
	directorNodes := htmlquery.Find(doc, "//th[contains(text(), 'Directed by')]/following-sibling::td[1]//a")
	for _, node := range directorNodes {
		director := strings.TrimSpace(htmlquery.InnerText(node))
		if director != "" {
			film.Directors = append(film.Directors, director)
		}
	}
	logger.WithField("directors", film.Directors).Debug("Extracted directors")

	// Get box office
	boxOfficeNode := htmlquery.FindOne(doc, "//th[contains(text(), 'Box office')]/following-sibling::td[1]")
	if boxOfficeNode != nil {
		boxOfficeText := htmlquery.InnerText(boxOfficeNode)
		film.BoxOffice = float64(extractMoney(boxOfficeText))
		logger.WithField("box_office", film.BoxOffice).Debug("Extracted box office")
	}

	// Get country
	countryNode := htmlquery.FindOne(doc, "//th[text()='Country']/following-sibling::td[1]|//th[text()='Countries']/following-sibling::td[1]")
	if countryNode != nil {
		countryText := strings.TrimSpace(htmlquery.InnerText(countryNode))
		film.CountryOfOrigin = strings.Split(countryText, "\n")[0] // Take first country if multiple
		logger.WithField("country", film.CountryOfOrigin).Debug("Extracted country")
	}

	logger.Info("Successfully scraped film details")
	return film
}

func extractYear(text string) int {
	logger := log.WithField("text", text)

	for _, word := range strings.Fields(text) {
		if year, err := strconv.Atoi(word); err == nil && year > 1900 && year < 2100 {
			logger.WithField("year", year).Debug("Successfully extracted year")
			return year
		}
	}

	logger.Warn("Failed to extract year")
	return 0
}

func extractMoney(text string) int {
	logger := log.WithField("text", text)

	// Remove everything before $ and after [
	if idx := strings.Index(text, "$"); idx != -1 {
		text = text[idx:]
	}
	if idx := strings.Index(text, "["); idx != -1 {
		text = text[:idx]
	}

	// Remove currency symbols and spaces
	text = strings.ReplaceAll(text, "$", "")
	text = strings.ReplaceAll(text, ",", "")
	text = strings.ReplaceAll(text, " ", "")

	// Convert billion/million to numbers
	text = strings.ToLower(text)
	if strings.Contains(text, "billion") {
		text = strings.ReplaceAll(text, "billion", "")
		text = strings.Trim(text, " ")
		if amount, err := strconv.ParseInt(text, 0, 0); err == nil {
			return int(amount * 1000000000)
		}
	}
	if strings.Contains(text, "million") {
		text = strings.ReplaceAll(text, "million", "")
		text = strings.Trim(text, " ")
		if amount, err := strconv.ParseInt(text, 0, 0); err == nil {
			return int(amount * 1000000)
		}
	}

	text = strings.Trim(text, " ")
	if amount, err := strconv.ParseInt(text, 0, 0); err == nil {
		return int(amount)
	}

	logger.WithField("cleanedText", text).Warn("Failed to extract money amount")
	return 0
}

func extractRunningTime(text string) int {
	logger := log.WithField("text", text)

	for _, word := range strings.Fields(text) {
		if minutes, err := strconv.Atoi(word); err == nil {
			logger.WithField("minutes", minutes).Debug("Successfully extracted running time")
			return minutes
		}
	}

	logger.Warn("Failed to extract running time")
	return 0
}
