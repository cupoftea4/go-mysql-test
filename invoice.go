package main

func (pdb *Store) CreateInvoiceWithProducts(invoice Invoice, products []InvoiceProduct) error {
	tx, err := pdb.db.Begin()
	if err != nil {
		return err
	}

	// Insert the invoice
	result, err := tx.Exec(`INSERT INTO invoices (shop_id, date, note, printed) VALUES (?, ?, ?, ?)`,
		invoice.ShopID, invoice.Date, invoice.Note, invoice.Printed)
	if err != nil {
		tx.Rollback()
		return err
	}

	invoiceID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	// Insert each product associated with the invoice
	for _, product := range products {
		_, err = tx.Exec(`INSERT INTO invoice_products (invoice_id, product_id, quantity, retail_price, wholereceipt_product_price, written_off) VALUES (?, ?, ?, ?, ?, ?)`,
			invoiceID, product.ProductID, product.Quantity, product.RetailPrice, product.WholeReceiptProductPrice, product.WrittenOff)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
