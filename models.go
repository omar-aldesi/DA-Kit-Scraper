package main

type Part struct {
	Id        int
	Name      string
	Price     float64
	Quantity  int
	URL       string
	ProductID int
}

type Kit struct {
	Price             float64
	ProductID         int
	Parts             []Part
	UndiscountedPrice float64
	DiscountPercent   float64
	ReturnAmount      float64
	ReturnQty         map[int]int
}
