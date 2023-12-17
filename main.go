package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var reader = bufio.NewReader(os.Stdin)

var levelsMap = map[int]sql.IsolationLevel{
	1: sql.LevelReadUncommitted,
	2: sql.LevelReadCommitted,
	3: sql.LevelRepeatableRead,
	4: sql.LevelSerializable,
}
 
func main() {
	db, err := sql.Open("mysql", "amy:1234@tcp(localhost:3306)/retail")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	store := NewStore(db)
	reader := bufio.NewReader(os.Stdin)

	for {
		var choice string
		fmt.Println("\n1: Add New Product")
		fmt.Println("2: Update Product")
		fmt.Println("3: Get All Products (Limited to 50)")
		fmt.Println("4: Get Product by ID")
		fmt.Println("5: Delete Product")
		fmt.Println("6: Create Invoice")
		fmt.Println("7: Clear Sales Anomalies and Run Procedure")
		fmt.Println("8: Simulate Stock Update Conflict")
		fmt.Println("9: Simulate Phantom Read")
		fmt.Println("X: Exit")
		fmt.Print("Enter choice: ")
		choice, _ = reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			var p ProductCreate

			p.CategoryID = readInt("Enter Category ID: ")
			p.Name = readLine("Enter Name: ")
			p.Stock = readInt("Enter Stock: ")
			p.Barcode = readLine("Enter Barcode: ")

			newProduct, err := store.CreateProduct(&p)
			if err != nil {
				fmt.Printf("Error adding product: %v", err)
			}
			fmt.Printf("Added Product: %+v\n", newProduct)

		case "2":
			var p ProductUpdate

			productId := readInt("Enter Product ID to Update: ")
			p.CategoryID = readInt("Enter New Category ID: ")
			p.Name = readLine("Enter New Name: ")
			p.Stock = readInt("Enter New Stock: ")

			updatedProduct, err := store.UpdateProduct(productId, &p)
			if err != nil {
				fmt.Printf("Error updating product: %v", err)
			}
			fmt.Printf("Updated Product: %+v\n", updatedProduct)

		case "3":
			products, err := store.GetAllProducts()
			if err != nil {
				fmt.Printf("Error retrieving products: %v", err)
			}
			for _, p := range *products {
				fmt.Printf("Product ID: %d, Category: [%d] %s, Name: %s, Stock: %d, Barcode: %s\n",
					p.ProductID, p.Category.CategoryID, p.Category.Name, p.Name, p.Stock, p.Barcode)
			}

		case "4":
			id := readInt("Enter Product ID to retrieve: ")
			product, err := store.GetProduct(id)
			if err != nil {
				fmt.Printf("Error retrieving product: %v", err)
			}
			fmt.Printf("Product ID: %d, Category: [%d] %s, Name: %s, Stock: %d, Barcode: %s\n",
				product.ProductID, product.Category.CategoryID, product.Category.Name, product.Name, product.Stock, product.Barcode)

		case "5":
			id := readInt("Enter Product ID to delete: ")
			err := store.DeleteProduct(id)
			if err != nil {
				log.Printf("Error deleting product: %v", err)
			} else {
				fmt.Println("Product deleted successfully.")
			}
		case "6":
			var invoice Invoice
			invoice.ShopID = readInt("Enter Shop ID: ")

			dateString := readLine("Enter Invoice Date (YYYY-MM-DD): ")
			invoice.Date, _ = time.Parse("2006-01-02", dateString)

			invoice.Note = readLine("Enter Note: ")
			invoice.Printed = readBool("Is the invoice printed? (y/n): ")

			var products []InvoiceProduct
			for {
					fmt.Println("Enter product details (enter -1 for Product ID to finish):")
					prodID := readInt("Enter Product ID: ")
					if prodID == -1 {
						break
					}

					products = append(products, InvoiceProduct{
						ProductID: prodID,
						Quantity: readInt("Enter Quantity: "),
						RetailPrice: readInt("Enter Retail Price: "),
						WholeReceiptProductPrice: readInt("Enter Whole Receipt Product Price: "),
						WrittenOff: readBool("Is the product written off? (y/n): "),
					})
			}

			// Call the function to create invoice and products
			err := store.CreateInvoiceWithProducts(invoice, products)
			if err != nil {
				fmt.Println("Error creating invoice:", err)
			}
			fmt.Println("Invoice created successfully.")
		case "7":
			count, anomalies, err := store.ClearSalesAnomaliesAndRunProcedure()
			if err != nil {
				fmt.Printf("An error occurred: %v\n", err)
				return
			}
	
			fmt.Printf("Found %d anomalies.\n", count)
			for _, anomaly := range anomalies {
				fmt.Printf("Anomaly: %+v\n", anomaly)
			}
		case "8":
			productID := readInt("Enter Product ID to simulate conflict: ")
			amounts := readLine("Enter comma-separated amounts to update stock (e.g. 10, -10): ")
			amountsArr := strings.Split(amounts, ",")
			var updateAmounts []int
			for _, amount := range amountsArr {
				amount = strings.TrimSpace(amount)
				if amount == "" {
					continue
				}
				updateAmount, err := strconv.Atoi(amount)
				if err != nil {
					fmt.Printf("Invalid amount: %s\n", amount)
					continue
				}
				updateAmounts = append(updateAmounts, updateAmount)
			}

			SimulateStockUpdateConflict(store, productID, updateAmounts)
		case "9":
			levelChoice := readLine("Enter isolation level (1: Read Uncommitted, 2: Read Committed, 3: Repeatable Read, 4: Serializable): ")
			level, err := strconv.Atoi(levelChoice)
			if err != nil {
				fmt.Printf("Invalid level: %s\n", levelChoice)
				continue
			}
			err = store.SimulatePhantomRead(2, 10, levelsMap[level])
			if err != nil {
				fmt.Printf("An error occurred: %v\n", err)
			}
		case "X", "x":
			fmt.Println("Exiting...")
			return

		default:
			fmt.Println("Invalid choice. Please enter a valid option.")
		}
	}
}

// SimulatePhantomRead simulates a phantom read scenario.
func (pdb *Store) SimulatePhantomRead(newProductID, newVariance int, level sql.IsolationLevel) error {
	ctx := context.Background()
	var wg sync.WaitGroup

	tx1, err := pdb.db.BeginTx(ctx, &sql.TxOptions{Isolation: level})
	if err != nil {
		return err
	}
	defer tx1.Rollback()

	initialCount, err := pdb.countRows(ctx, tx1, "sales_anomalies")
	if err != nil {
		return err
	}
	fmt.Printf("Initial count of sales anomalies: %d\n", initialCount)

	wg.Add(1)
	go func() {
		defer wg.Done()
		tx2, err := pdb.db.BeginTx(ctx, &sql.TxOptions{})
		if err != nil {
			fmt.Println("Transaction 2: BeginTx Error:", err)
			return
		}
		defer tx2.Rollback()

		if err := pdb.insertAnomaly(ctx, tx2, newProductID, newVariance); err != nil {
			fmt.Println("Transaction 2: Insert Error:", err)
			return
		}

		if err := tx2.Commit(); err != nil {
			fmt.Println("Transaction 2: Commit Error:", err)
			return
		}
		fmt.Println("Transaction 2: New sales anomaly inserted.")
	}()

	wg.Wait()

	finalCount, err := pdb.countRows(ctx, tx1, "sales_anomalies")
	if err != nil {
		return err
	}
	fmt.Printf("Final count of sales anomalies: %d\n", finalCount)

	if err := tx1.Commit(); err != nil {
		return err
	}

	// Check for phantom read
	if initialCount != finalCount {
		fmt.Println("A phantom read occurred: the count of sales anomalies changed.")
	} else {
		fmt.Println("No phantom read occurred: the count of sales anomalies did not change.")
	}

	return nil
}

func SimulateStockUpdateConflict(store *Store, productId int, updateAmounts []int) {
	var wg sync.WaitGroup
	fmt.Println()

	for _, amount := range updateAmounts {
		wg.Add(1)
		go func(amount int) {
			defer wg.Done()
			err := store.UpdateProductStock(productId, amount)
			if err != nil {
				fmt.Printf("Transaction Error (Amount: %d): %v\n", amount, err)
			} else {
				fmt.Printf("Transaction successfully updated stock (Amount: %d).\n", amount)
			}
		}(amount)
	}

	wg.Wait() // Wait for all transactions to complete
}

// executeSQL is a utility function to execute an SQL command without returning any rows.
func (pdb *Store) executeSQL(query string, args ...interface{}) error {
	_, err := pdb.db.Exec(query, args...)
	return err
}

// countRows is a utility function to count rows in a table.
func (pdb *Store) countRows(ctx context.Context, tx *sql.Tx, tableName string) (int, error) {
	var count int
	err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+tableName).Scan(&count)
	return count, err
}

func readLine(caption string) string {
	fmt.Print(caption)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func readInt(caption string) int {
	fmt.Print(caption)
	var i int
	fmt.Scanln(&i)
	return i
}

func readBool(prompt string) bool {
	var b string
	fmt.Print(prompt)
	fmt.Scanln(&b)
	return strings.ToLower(b) == "y" || strings.ToLower(b) == "yes"
}
