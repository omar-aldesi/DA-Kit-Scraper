package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type PriceResponse struct {
	OriginalPrice float64 `json:"original_price"`
	SalePrice     float64 `json:"sale_price"`
}

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	},
}

func fetchPrice(productID int) (float64, error) {
	payload, _ := json.Marshal(map[string]int{"product_id": productID})

	resp, err := httpClient.Post(
		"https://www.detroitaxle.com/kobe_api/woocommerce-products/web-sale-price",
		"application/json",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var priceResp PriceResponse
	json.NewDecoder(resp.Body).Decode(&priceResp)
	return priceResp.SalePrice, nil
}

func getProductID(doc *goquery.Document) int {
	val, exists := doc.Find("button.single_add_to_cart_button").Attr("value")
	if !exists {
		return 0
	}
	id, _ := strconv.Atoi(strings.TrimSpace(val))
	return id
}

func fetchDoc(url string) (*goquery.Document, error) {
	res, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return goquery.NewDocumentFromReader(res.Body)
}

func scrapeKit(kitURL string) (Kit, error) {
	kit := Kit{
		UndiscountedPrice: 0,
		ReturnAmount:      0,
	}

	doc, err := fetchDoc(kitURL)
	if err != nil {
		return kit, err
	}

	kit.ProductID = getProductID(doc)

	// Parse parts from table first, then fetch kit price + part prices concurrently
	partId := 1
	doc.Find("table.product-pg__table tr.table__row").Each(func(i int, s *goquery.Selection) {
		cells := s.Find("td.table__cell")
		if cells.Length() < 3 {
			return
		}

		qtyStr := strings.TrimSpace(cells.Eq(0).Text())
		qty, _ := strconv.Atoi(qtyStr)
		name := strings.TrimSpace(cells.Eq(1).Find("a").Text())
		url, _ := cells.Eq(1).Find("a").Attr("href")

		kit.Parts = append(kit.Parts, Part{
			Name:     name,
			Id:       partId,
			Quantity: qty,
			URL:      url,
		})
		partId++
	})

	// Fetch kit price and part prices concurrently
	var wg sync.WaitGroup
	var kitPriceErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		kit.Price, kitPriceErr = fetchPrice(kit.ProductID)
	}()

	parts, partsErr := scrapePartPrices(kit.Parts)
	wg.Wait()

	if kitPriceErr != nil {
		return kit, kitPriceErr
	}
	if partsErr != nil {
		return kit, partsErr
	}

	kit.Parts = parts
	return kit, nil
}

func scrapePartPrices(parts []Part) ([]Part, error) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // max 10 concurrent requests

	for i := range parts {
		wg.Add(1)
		go func(i int) {
			sem <- struct{}{}
			defer func() {
				<-sem
				wg.Done()
			}()

			partDoc, err := fetchDoc(parts[i].URL)
			if err != nil {
				log.Printf("error fetching part %s: %v", parts[i].Name, err)
				return
			}

			parts[i].ProductID = getProductID(partDoc)
			parts[i].Price, err = fetchPrice(parts[i].ProductID)
			if err != nil {
				log.Printf("error fetching price for %s: %v", parts[i].Name, err)
			}
		}(i)
	}

	wg.Wait()
	return parts, nil
}
func (k *Kit) FindPartByID(id int) (*Part, error) {
	for _, part := range k.Parts {
		if part.Id == id {
			return &part, nil
		}
	}
	return nil, fmt.Errorf("part with id %d not found", id)
}
func (k *Kit) returnItem(partId int, qty int) {
	part, err := k.FindPartByID(partId)
	if err != nil {
		panic("Part not found")
	}
	k.ReturnAmount += part.Price * float64(qty) * (1 - k.DiscountPercent/100)
}

func (k *Kit) removeItem(partId int, qty int) {
	part, err := k.FindPartByID(partId)
	if err != nil {
		panic("Part not found")
	}
	k.ReturnAmount -= part.Price * float64(qty) * (1 - k.DiscountPercent/100)
}
