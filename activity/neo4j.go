package activity

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func CreateConnection() (*DBConnection, error) {
	dbURI := "neo4j://localhost:7687" // scheme://host(:port) (default port is 7687)
	driver, err := neo4j.NewDriverWithContext(dbURI, neo4j.BasicAuth("neo4j", "password", ""))
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	return &DBConnection{Driver: driver, Context: ctx}, nil
}

func InsertActivity(dbC *DBConnection, activity *Activity) error {
	/*This functions creates a new activity in the Database
	 */
	result, err := neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
		`MERGE (ac:Activity{Description: $description, StartTime: datetime($startTime), EndTime:datetime($endTime), Priority:$priority })
		return id(ac) as activityID`,
		map[string]any{
			"description": activity.Description,
			"startTime":   activity.StartTime,
			"endTime":     activity.EndTime,
			"priority":    activity.Priority,
			"status":      activity.Status,
		}, neo4j.EagerResultTransformer)
	if err != nil {
		return err
	}
	activityID, _, err := neo4j.GetRecordValue[int64](result.Records[0], "activityID")
	if err != nil {
		return err
	}

	// Create keywords
	for _, keyword := range activity.Keywords {
		// Create nodes
		_, err = neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
			`MERGE (kw:Keyword{Text: $text}) WITH kw
			MATCH (ac:Activity) WHERE id(ac) = toInteger($id) WITH kw, ac
			MERGE (kw) <-[r: INDEX]-> (ac)`,
			map[string]any{
				"text": keyword,
				"id":   activityID,
			}, neo4j.EagerResultTransformer)
		if err != nil {
			return err
		}
	}

	// Create persons
	for _, person := range activity.Participants {
		_, err = neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
			`MERGE (p:Person{Name: $name, Handle: $handle}) WITH p
			MATCH (ac:Activity) WHERE id(ac) = toInteger($id) WITH p, ac
			MERGE (p) <-[r: PARTICIPANT]-> (ac)`,
			map[string]any{
				"name":   person.Name,
				"handle": person.Handle,
				"id":     activityID,
			}, neo4j.EagerResultTransformer)
		if err != nil {
			return err
		}
	}

	//Update activity ID
	activity.ID = int(activityID)

	return nil
}

func FetchActivityByID(dbC *DBConnection, ID int) (*Activity, error) {
	/*This function retrieves an activity from the database
	 */
	activity := Activity{ID: ID}
	result, err := neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
		"MATCH (ac:Activity) WHERE id(ac) = toInteger($id) RETURN ac",
		map[string]any{
			"id": ID,
		}, neo4j.EagerResultTransformer)
	if err != nil {
		return nil, err
	}
	activityNode, _, err := neo4j.GetRecordValue[neo4j.Node](result.Records[0], "ac")
	if err != nil {
		return nil, fmt.Errorf("could not find activity node")
	}
	startTime, err := neo4j.GetProperty[time.Time](activityNode, "StartTime")
	fmt.Println(startTime)
	if err != nil {
		return nil, err
	}
	activity.StartTime = startTime.Format(TIMEFORMAT)
	endTime, err := neo4j.GetProperty[time.Time](activityNode, "EndTime")
	fmt.Println(endTime)
	if err != nil {
		return nil, err
	}
	activity.EndTime = endTime.Format(TIMEFORMAT)
	activity.Description, err = neo4j.GetProperty[string](activityNode, "Description")
	if err != nil {
		return nil, err
	}
	activity.Priority, err = neo4j.GetProperty[int64](activityNode, "Priority")
	if err != nil {
		return nil, err
	}
	activity.Status, err = neo4j.GetProperty[bool](activityNode, "Status")
	if err != nil {
		return nil, err
	}

	// Fetch participants
	result, err = neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
		`MATCH (ac:Activity) WHERE id(ac)=toInteger($id) WITH ac
		MATCH (ac)-[:PARTICIPANT]-(p:Person) RETURN p`,
		map[string]any{
			"id": ID,
		}, neo4j.EagerResultTransformer)
	if err != nil {
		return nil, err
	}

	for _, record := range result.Records {
		var person Person = Person{}
		personNode, _, err := neo4j.GetRecordValue[neo4j.Node](record, "p")
		if err != nil {
			return nil, err
		}
		person.Name, err = neo4j.GetProperty[string](personNode, "Name")
		if err != nil {
			return nil, err
		}
		person.Handle, err = neo4j.GetProperty[string](personNode, "Handle")
		if err != nil {
			return nil, err
		}
		activity.Participants = append(activity.Participants, person)
	}

	return &activity, nil
}

func FetchActivityByFilter(dbC *DBConnection, filter *Filter) ([]Activity, error) {
	/*This function fetches activity by filters
	 */
	activities := make([]Activity, 0)

	// Default values for TimeBounds
	if filter.StartTimeBounds.isEmpty() {
		filter.StartTimeBounds.LowerBound = "2000-11-22T18:59:00.000+0900"
		filter.StartTimeBounds.UpperBound = "2099-11-22T18:59:00.000+0900"
	}

	if filter.EndTimeBounds.isEmpty() {
		filter.EndTimeBounds.LowerBound = "2000-11-22T18:59:00.000+0900"
		filter.EndTimeBounds.UpperBound = "2099-11-22T18:59:00.000+0900"
	}

	result, err := neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
		`MATCH (ac:Activity) WHERE ac.StartTime >= DATETIME($startTimeLower) AND ac.StartTime <= DATETIME($startTimeUpper)
		AND ac.EndTime >= DATETIME($EndTimeLower) AND ac.EndTime <= DATETIME($EndTimeUpper) WITH ac
		MATCH (kw:Keyword) WHERE SIZE($keywords) = 0 OR kw.Text IN $keywords WITH kw
		MATCH (kw)-[:INDEX]-(ac:Activity)
		RETURN ID(ac) as acID`,
		map[string]any{
			"keywords":       filter.Keywords,
			"startTimeLower": filter.StartTimeBounds.LowerBound,
			"startTimeUpper": filter.StartTimeBounds.UpperBound,
			"EndTimeLower":   filter.EndTimeBounds.LowerBound,
			"EndTimeUpper":   filter.EndTimeBounds.UpperBound,
		}, neo4j.EagerResultTransformer)
	if err != nil {
		return nil, err
	}

	for _, record := range result.Records {
		activityID, _, err := neo4j.GetRecordValue[int64](record, "acID")
		if err != nil {
			return nil, fmt.Errorf("could not find activity node")
		}
		activity, err := FetchActivityByID(dbC, int(activityID))
		if err != nil {
			return nil, err
		}
		activities = append(activities, *activity)
	}

	return activities, nil
}

func DeleteActivityByID(dbC *DBConnection, activityID int) error {
	/*This function deletes activity from database
	 */
	_, err := neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
		`MATCH (ac:Activity) WHERE ID(ac) = toInteger($id) WITH ac
		DETACH DELETE ac`,
		map[string]any{
			"id": activityID,
		}, neo4j.EagerResultTransformer)
	if err != nil {
		return err
	}
	return nil
}

func InsertActivityRelation(dbC *DBConnection, mainActivityID int, otherActivityID int) error {
	/*This functions inserts a relation between activitiesoo
	 */
	_, err := neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
		`MATCH(m:Activity) WHERE ID(m) = toInteger($mainId) with m
		MATCH(o:Activity) WHERE ID(o) = toInteger($otherId) with o
		MERGE (m)<-[:FOLLOWUP]-(o)`,
		map[string]any{
			"mainId":  mainActivityID,
			"otherId": otherActivityID,
		}, neo4j.EagerResultTransformer)
	if err != nil {
		return err
	}
	return nil
}
