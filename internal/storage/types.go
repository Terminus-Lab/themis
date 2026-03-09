package storage

import "time"

type QueryFilters struct {
	AgentName    string
	AgentVersion string
	Verdict      string
	FromDate     time.Time
	ToDate       time.Time
	Limit        int
	Offset       int
	OrderBy      string
}

type QueryResult struct {
}
