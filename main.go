package main

import (
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
)

var templates = template.Must(template.ParseGlob("templates/*.html"))

// Session state (simple in-memory, single user)
var currentKit *Kit

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/scrape", handleScrape)
	http.HandleFunc("/part", handlePart)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "index.html", nil)
}

func handleScrape(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	url := r.FormValue("url")
	if url == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	kit, err := scrapeKit(url)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates.ExecuteTemplate(w, "error.html", err.Error())
		return
	}

	for _, part := range kit.Parts {
		kit.UndiscountedPrice += part.Price * float64(part.Quantity)
	}
	kit.DiscountPercent = math.Round(((kit.UndiscountedPrice-kit.Price)/kit.UndiscountedPrice*100)*10) / 10

	currentKit = &kit
	templates.ExecuteTemplate(w, "kit.html", currentKit)
}

func handlePart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if currentKit == nil {
		http.Error(w, "No kit loaded", http.StatusBadRequest)
		return
	}

	partIdStr := r.FormValue("part_id")
	action := r.FormValue("action")
	qtyStr := r.FormValue("qty")

	qty, err := strconv.Atoi(qtyStr)
	if err != nil || qty < 0 {
		http.Error(w, "Invalid quantity", http.StatusBadRequest)
		return
	}

	partId, err := strconv.Atoi(partIdStr)
	if err != nil {
		http.Error(w, "Invalid part_id", http.StatusBadRequest)
		return
	}

	if currentKit.ReturnQty == nil {
		currentKit.ReturnQty = make(map[int]int)
	}

	if action == "add" {
		currentKit.ReturnQty[partId] += qty
	} else {
		currentKit.ReturnQty[partId] -= qty
		if currentKit.ReturnQty[partId] < 0 {
			currentKit.ReturnQty[partId] = 0
		}
	}

	// Recompute return amount from scratch
	currentKit.ReturnAmount = 0
	for id, q := range currentKit.ReturnQty {
		part, err := currentKit.FindPartByID(id)
		if err == nil {
			currentKit.ReturnAmount += part.Price * float64(q) * (1 - currentKit.DiscountPercent/100)
		}
	}

	templates.ExecuteTemplate(w, "return.html", currentKit)
}
