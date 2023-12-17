package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (pdb *Store) CreateProduct(p *ProductCreate) (*Product, error) {
  var id int64
  var err error

  for attempts := 0; attempts < 3; attempts++ {
      var result sql.Result
      result, err = pdb.db.Exec(`INSERT INTO products (category_id, name, stock, barcode) VALUES (?, ?, ?, ?)`, 
                                p.CategoryID, p.Name, p.Stock, p.Barcode)

      if err == nil {
        id, err = result.LastInsertId()
        if err == nil {
          break
        }
      }

      if !isTransientError(err) {
        return nil, err
      }

      // Log the error and wait before retrying
      fmt.Printf("Attempt %d failed: %v. Retrying...\n", attempts+1, err)
      time.Sleep(time.Second * 2) // Wait for 2 seconds before retrying
  }

  if err != nil {
      return nil, fmt.Errorf("after retries, failed to create product: %v", err)
  }

  return pdb.GetProduct(int(id))
}

func (pdb *Store) GetAllProducts() (*[]Product, error) {
	var products []Product
	query := `SELECT p.product_id, p.name, p.stock, p.barcode, c.category_id, c.name 
            FROM products p 
            JOIN product_categories c ON p.category_id = c.category_id 
            ORDER BY p.product_id ASC
            LIMIT 50`
	rows, err := pdb.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p Product
		var categoryName string
		if err := rows.Scan(&p.ProductID, &p.Name, &p.Stock, &p.Barcode, &p.Category.CategoryID, &categoryName); err != nil {
			return nil, err
		}
		p.Category.Name = categoryName
		products = append(products, p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &products, nil
}

func (pdb *Store) GetProduct(productID int) (*Product, error) {
	p := &Product{}
	query := `SELECT p.product_id, p.name, p.stock, p.barcode, c.category_id, c.name 
            FROM products p 
            JOIN product_categories c ON p.category_id = c.category_id 
            WHERE p.product_id = ?`
	err := pdb.db.QueryRow(query, productID).Scan(&p.ProductID, &p.Name, &p.Stock, &p.Barcode, &p.Category.CategoryID, &p.Category.Name)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (pdb *Store) UpdateProduct(productId int, p *ProductUpdate) (*Product, error) {
	query := `UPDATE products SET category_id = ?, name = ?, stock = ? WHERE product_id = ?`
	_, err := pdb.db.Exec(query, p.CategoryID, p.Name, p.Stock, productId)
	if err != nil {
		return nil, err
	}

	return pdb.GetProduct(productId)
}

func (pdb *Store) DeleteProduct(productID int) error {
	query := `DELETE FROM products WHERE product_id = ?`
	res, err := pdb.db.Exec(query, productID)

	if err != nil {
		return err
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affectedRows == 0 {
		return fmt.Errorf("Product with ID %d not found", productID)
	}

	fmt.Printf("Affected Rows: %d\n", affectedRows)
	return err
}

func (pdb *Store) UpdateProductStock(productID, changeInStock int) error {
  for attempts := 0; attempts < 3; attempts++ {
      // Start transaction
      tx, err := pdb.db.Begin()
      if err != nil {
          return err
      }

      // Get current stock
      var currentStock int
      err = tx.QueryRow("SELECT stock FROM products WHERE product_id = ?", productID).Scan(&currentStock)
      if err != nil {
          tx.Rollback()
          return err
      }

      // Calculate new stock
      newStock := currentStock + changeInStock

      // Update stock with optimistic locking
      result, err := tx.Exec("UPDATE products SET stock = ? WHERE product_id = ? AND stock = ?", newStock, productID, currentStock)
      if err != nil {
        fmt.Printf("Attempt %d failed: %v. Retrying...\n", attempts+1, err)
        tx.Rollback()
        return err
      }

      affected, err := result.RowsAffected()
      if err != nil {
        fmt.Printf("Attempt %d failed: %v. Retrying...\n", attempts+1, err)
        tx.Rollback()
        return err
      }

      if affected == 0 {
          // No rows affected, meaning the stock was changed by another transaction.
          // Rollback and possibly retry.
          fmt.Printf("Attempt %d failed: %s. Retrying...\n", attempts+1, "no rows affected")
          tx.Rollback()
          continue // Retry the loop
      }

      // Commit transaction
      if err := tx.Commit(); err != nil {
        fmt.Printf("Attempt %d failed: %v. Retrying...\n", attempts+1, err)
        return err
      }
      return nil // Success
  }
  return fmt.Errorf("update failed after retries")
}


// isTransientError checks if the error is a transient MySQL error.
func isTransientError(err error) bool {
  // Check if error is a MySQL error
  if driverErr, ok := err.(*mysql.MySQLError); ok {
      switch driverErr.Number {
      case 1040: // ER_CON_COUNT_ERROR: Too many connections
      case 1205: // ER_LOCK_WAIT_TIMEOUT: Lock wait timeout exceeded
      case 1213: // ER_LOCK_DEADLOCK: Deadlock found
      case 2003: // CR_CONN_HOST_ERROR: Can't connect to MySQL server on 'host'
      case 2006: // CR_SERVER_GONE_ERROR: MySQL server has gone away
      case 2013: // CR_SERVER_LOST: Lost connection to MySQL server during query
        return true
      }
  }
  return false
}
