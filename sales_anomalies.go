package main

import (
	"context"
	"database/sql"
	"fmt"
)

func (pdb *Store) ClearSalesAnomaliesAndRunProcedure() (int, []SalesAnomaly, error) {
	var anomalies []SalesAnomaly
	var count int

	tx, err := pdb.db.Begin()
	if err != nil {
		return count, anomalies, err
	}

	// Delete all records from sales_anomalies
	_, err = tx.Exec("DELETE FROM sales_anomalies")
	if err != nil {
		tx.Rollback() // Rollback in case of error
		return count, anomalies, fmt.Errorf("error deleting sales anomalies: %v", err)
	}

	// Call the stored procedure
	_, err = tx.Exec("CALL IdentifyIrregularSalesPatterns()")
	if err != nil {
		tx.Rollback() // Rollback in case of error
		return count, anomalies, fmt.Errorf("error executing stored procedure: %v", err)
	}

	// Fetch the count of anomalies
	err = tx.QueryRow("SELECT COUNT(*) FROM sales_anomalies").Scan(&count)
	if err != nil {
		tx.Rollback()
		return count, anomalies, fmt.Errorf("error fetching anomalies count: %v", err)
	}

	// Fetch up to 5 anomalies
	rows, err := tx.Query("SELECT product_id, observed_variance FROM sales_anomalies LIMIT 5")
	if err != nil {
		tx.Rollback()
		return count, anomalies, fmt.Errorf("error fetching anomalies: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var anomaly SalesAnomaly
		// Assume SalesAnomaly has fields that match your anomaly table
		if err := rows.Scan(&anomaly.ProductID, &anomaly.Variance); err != nil {
			tx.Rollback()
			return count, anomalies, err
		}
		anomalies = append(anomalies, anomaly)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return count, anomalies, fmt.Errorf("error committing transaction: %v", err)
	}

	return count, anomalies, nil
}

// insertAnomaly is a utility function to insert a new anomaly.
func (pdb *Store) insertAnomaly(ctx context.Context, tx *sql.Tx, productID, observedVariance int) error {
	return pdb.executeSQL("INSERT INTO sales_anomalies (product_id, observed_variance) VALUES (?, ?)", productID, observedVariance)
}
