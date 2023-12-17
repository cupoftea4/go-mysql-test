package main

import "time"

type Product struct {
	ProductID   int      `json:"product_id"`
	Category    Category `json:"category"`
	Name        string   `json:"name"`
	Stock       int      `json:"stock"`
	Barcode     string   `json:"barcode"`
}

type ProductCreate struct {
	CategoryID int    `json:"category_id"`
	Name       string `json:"name"`
	Stock      int    `json:"stock"`
	Barcode    string `json:"barcode"`
}

type ProductUpdate struct {
	CategoryID int    `json:"category_id"`
	Name       string `json:"name"`
	Stock      int    `json:"stock"`
}

type Category struct {
	CategoryID int    `json:"category_id"`
	Name       string `json:"name"`
}

type Invoice struct {
	InvoiceID int       `json:"invoice_id"`
	ShopID    int       `json:"shop_id"`
	Date      time.Time `json:"date"`
	Note      string    `json:"note"`
	Printed   bool      `json:"printed"`
}

type InvoiceProduct struct {
	InvoiceProductID        int  `json:"invoice_product_id"`
	InvoiceID               int  `json:"invoice_id"`
	ProductID               int  `json:"product_id"`
	Quantity                int  `json:"quantity"`
	RetailPrice             int  `json:"retail_price"`
	WholeReceiptProductPrice int  `json:"wholereceipt_product_price"`
	WrittenOff              bool `json:"written_off"`
}

type SalesAnomaly struct {
	ProductID int
	Variance  int
}
