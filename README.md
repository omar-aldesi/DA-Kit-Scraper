# Kit Scraper

A personal tool for analyzing **Detroit Axle** kit URLs — extract parts, view pricing, and calculate return values on the fly.

Built for work. Built to be fast.

---

## Stack

| Layer    | Tech |
|----------|------|
| Backend  | Go   |
| Frontend | HTMX |
| Styling  | CSS  |

---

## What it does

- Paste a Detroit Axle kit URL
- Instantly scrapes all parts & pricing
- Shows kit price, undiscounted price, and discount %
- Toggle individual parts for return and adjust quantities
- Calculates total return value in real-time

---

## Running it

```bash
go run main.go
```

Then open `http://localhost:8080` in your browser.

---

## Structure

```
.
├── main.go          # server and endpoints
├── models.go        # models and DS structure
├── scrape.go        # scraper logic
├── templates/
│   ├── index.html   # main page
│   ├── kit.html     # kit result partial
│   └── return.html  # return amount partial
└── static/
    └── style.css    # styles
```

---

made with ♥ by Omar