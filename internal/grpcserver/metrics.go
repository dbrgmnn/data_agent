package grpcserver

import (
	"context"
	"data_agent/internal/models"
	"data_agent/proto"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type MetricService struct {
	proto.UnimplementedMetricServiceServer
	DB *sql.DB
}

// retrieves all metrics from the database
func (s *MetricService) ListMetrics(ctx context.Context, req *proto.MetricRequest) (*proto.MetricList, error) {
	// query all metrics for a specific hostname with a limit
	query := `
		SELECT m.id, m.host_id, m.uptime, m.cpu, m.ram, m.disk, m.network, m.time
		FROM metrics m
		JOIN hosts h ON m.host_id = h.id
		WHERE h.hostname = $1
		ORDER BY m.time DESC
		LIMIT $2
	`
	rows, err := s.DB.QueryContext(ctx, query, req.Hostname, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var metrics []*proto.Metric
	for rows.Next() {
		m, err := s.scanMetric(rows)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return &proto.MetricList{Metrics: metrics}, nil
}

// retrieves the latest metrics
func (s *MetricService) GetLatestMetrics(ctx context.Context, _ *proto.Empty) (*proto.MetricList, error) {
	// query latest metrics for all hosts
	query := `
		SELECT DISTINCT ON (m.host_id) m.id, m.host_id, m.uptime, m.cpu, m.ram, m.disk, m.network, m.time
		FROM metrics m
		ORDER BY m.host_id, m.time DESC
	`
	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest metrics: %w", err)
	}
	defer rows.Close()

	var metrics []*proto.Metric
	for rows.Next() {
		m, err := s.scanMetric(rows)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return &proto.MetricList{Metrics: metrics}, nil
}

// scanMetric helper to scan a row and map to proto
func (s *MetricService) scanMetric(rows *sql.Rows) (*proto.Metric, error) {
	var (
		id, hostID   int64
		uptime       uint64
		cpu, ram     float64
		diskJSON     []byte
		networkJSON  []byte
		recordedTime time.Time
	)

	if err := rows.Scan(&id, &hostID, &uptime, &cpu, &ram, &diskJSON, &networkJSON, &recordedTime); err != nil {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	// Parse JSON from database
	var diskMetrics []models.DiskMetric
	if err := json.Unmarshal(diskJSON, &diskMetrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal disk metrics: %w", err)
	}

	var netMetrics []models.NetMetric
	if err := json.Unmarshal(networkJSON, &netMetrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal network metrics: %w", err)
	}

	// Map to proto
	pMetric := &proto.Metric{
		Id:     id,
		HostId: hostID,
		Uptime: uptime,
		Cpu:    cpu,
		Ram:    ram,
		Time:   timestamppb.New(recordedTime),
	}

	for _, d := range diskMetrics {
		pMetric.Disk = append(pMetric.Disk, &proto.DiskMetric{
			Path:        d.Path,
			Total:       d.Total,
			Used:        d.Used,
			Free:        d.Free,
			UsedPercent: d.UsedPercent,
		})
	}

	for _, n := range netMetrics {
		pMetric.Network = append(pMetric.Network, &proto.NetMetric{
			Name:        n.Name,
			BytesSent:   n.BytesSent,
			BytesRecv:   n.BytesRecv,
			PacketsSent: n.PacketsSent,
			PacketsRecv: n.PacketsRecv,
			ErrIn:       n.ErrIn,
			ErrOut:      n.ErrOut,
			DropIn:      n.DropIn,
			DropOut:     n.DropOut,
		})
	}

	return pMetric, nil
}
