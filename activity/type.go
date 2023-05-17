package activity

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const TIMEFORMAT = "2016-11-22T18:59:00.000+0900"

type Activity struct {
	ID           int      `json:"id"`
	Description  string   `json:"description"`
	StartTime    string   `json:"start_time"`
	EndTime      string   `json:"end_time"`
	Participants []Person `json:"participants"`
	Keywords     []string `json:"keywords"`
	Priority     int64    `json:"priority"`
	Status       bool     `json:"status"`
}

type Person struct {
	Name   string `json:"name"`
	Handle string `json:"handle"`
}

type Filter struct {
	StartTimeBounds TimeBounds `json:"start_time_bounds"`
	EndTimeBounds   TimeBounds `json:"end_time_bounds"`
	Keywords        []string   `json:"keywords"`
}

func (f TimeBounds) isEmpty() bool {
	if f.LowerBound == "" && f.UpperBound == "" {
		return true
	}
	return false
}

type TimeBounds struct {
	LowerBound string `json:"lower_bound"`
	UpperBound string `json:"upper_bound"`
}

type DBConnection struct {
	Driver  neo4j.DriverWithContext
	Context context.Context
}

func (dbC DBConnection) CloseConnection() {
	dbC.Driver.Close(dbC.Context)
}
