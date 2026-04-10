package db

import (
	"context"
	"data_agent/internal/config"
	"data_agent/internal/models"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// cache for host IDs to avoid redundant database lookups
var hostCache sync.Map

// initialize and return database connection
func InitDB() (*sql.DB, error) {
	// load config from .env file
	cfg := config.LoadConfig()
	if cfg.DBHost == "" || cfg.DBPort == "" || cfg.DBUser == "" || cfg.DBPass == "" || cfg.DBName == "" {
		return nil, fmt.Errorf("database configuration variables are not set properly")
	}

	// make connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName)

	// open connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// create a context that is canceled on exit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// test connection
	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to the database: %w", err)
	}

	log.Println("Successfully connected to the database")
	return db, nil
}

// insert host and metric into database
func SaveMetric(ctx context.Context, db *sql.DB, metric *models.MetricMessage) error {
	var hostID int64

	// check cache first
	if id, ok := hostCache.Load(metric.Host.Hostname); ok {
		hostID = id.(int64)
	}

	// transactions for secure queries
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// rollback will be executed if something goes wrong
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("transaction rollback error: %v", err)
		}
	}()

	if hostID == 0 {
		// check if host exists in database
		err = tx.QueryRowContext(ctx, "SELECT id FROM hosts WHERE hostname=$1", metric.Host.Hostname).Scan(&hostID)
		if errors.Is(err, sql.ErrNoRows) {
			// insert host into database when not exists
			err = tx.QueryRowContext(
				ctx,
				`INSERT INTO hosts (hostname, os, platform, platform_ver, kernel_ver) 
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id`,
				metric.Host.Hostname,
				metric.Host.OS,
				metric.Host.Platform,
				metric.Host.PlatformVer,
				metric.Host.KernelVer,
			).Scan(&hostID)
			if err != nil {
				return fmt.Errorf("insert host info: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("select host_id: %w", err)
		}
		// update cache
		hostCache.Store(metric.Host.Hostname, hostID)
	}

	// set host_id in metric
	metric.Metric.HostID = hostID

	// marshaling disk and network slices to JSON
	diskJSON, err := json.Marshal(metric.Metric.Disk)
	if err != nil {
		return fmt.Errorf("marshal disk metrics: %w", err)
	}
	networkJSON, err := json.Marshal(metric.Metric.Network)
	if err != nil {
		return fmt.Errorf("marshal network metrics: %w", err)
	}

	// insert metric into database
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO metrics (host_id, uptime, cpu, ram, disk, network, time)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		metric.Metric.HostID,
		metric.Metric.Uptime,
		metric.Metric.CPU,
		metric.Metric.RAM,
		diskJSON,
		networkJSON,
		metric.Metric.Time,
	)
	if err != nil {
		return fmt.Errorf("insert metric: %w", err)
	}

	// commit transaction when all saved
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
