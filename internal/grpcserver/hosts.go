package grpcserver

import (
	"context"
	"data_agent/proto"
	"database/sql"
	"fmt"
)

type HostService struct {
	proto.UnimplementedHostServiceServer
	DB *sql.DB
}

// retrieves all hosts from the database
func (s *HostService) ListHosts(ctx context.Context, _ *proto.Empty) (*proto.HostList, error) {
	// query all hosts from database
	rows, err := s.DB.QueryContext(ctx, `SELECT id, hostname, os, platform, platform_ver, kernel_ver FROM hosts`)
	if err != nil {
		return nil, fmt.Errorf("failed to query hosts: %w", err)
	}
	defer rows.Close()

	var hosts []*proto.Host
	for rows.Next() {
		var host proto.Host
		if err := rows.Scan(&host.Id, &host.Hostname, &host.Os, &host.Platform, &host.PlatformVer, &host.KernelVer); err != nil {
			return nil, fmt.Errorf("failed to scan host: %w", err)
		}
		hosts = append(hosts, &host)
	}

	return &proto.HostList{Hosts: hosts}, nil
}

// retrieves a specific host by hostname
func (s *HostService) GetHost(ctx context.Context, req *proto.HostName) (*proto.Host, error) {
	var host proto.Host

	// query host from database
	err := s.DB.QueryRowContext(
		ctx,
		`SELECT id, hostname, os, platform, platform_ver, kernel_ver FROM hosts WHERE hostname=$1`, req.Hostname,
	).Scan(&host.Id, &host.Hostname, &host.Os, &host.Platform, &host.PlatformVer, &host.KernelVer)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("host not found: %s", req.Hostname)
		}
		return nil, fmt.Errorf("failed to get host: %w", err)
	}
	return &host, nil
}
