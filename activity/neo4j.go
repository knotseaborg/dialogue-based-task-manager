package activity

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func CreateConnection() (*DBConnection, error) {
	driver, err := neo4j.NewDriverWithContext(os.Getenv("DBTM_DB_URI"),
		neo4j.BasicAuth(os.Getenv("DBTM_DB_USER"), os.Getenv("DBTM_DB_PASSWORD"), ""))
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	return &DBConnection{Driver: driver, Context: ctx}, nil
}

func InsertActivity(dbC *DBConnection, activity *Activity) error {
	/*This functions creates a new activity in the Database*/
	result, err := neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
		`MERGE (ac:Activity{Description: $description, StartTime: datetime($startTime), 
			EndTime:datetime($endTime), Priority:$priority, Status:$status })
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
	/*This function retrieves an activity from the database*/
	activity := Activity{ID: ID}
	result, err := neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
		"MATCH (ac:Activity) WHERE id(ac) = toInteger($id) RETURN ac",
		map[string]any{
			"id": ID,
		}, neo4j.EagerResultTransformer)
	if err != nil {
		return nil, err
	}
	if len(result.Records) == 0 {
		return nil, NoActivityError{}
	}
	activityNode, _, err := neo4j.GetRecordValue[neo4j.Node](result.Records[0], "ac")
	if err != nil {
		return nil, NoActivityError{}
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
	/*This function fetches activity by filters*/
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

	if len(result.Records) == 0 {
		return nil, NoActivityError{}
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

func DeleteActivityByID(dbC *DBConnection, activityID int, cascade bool) error {
	/*This function deletes the activity and all associated activities from database if cascading is requested*/
	if cascade {
		result, err := neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
			`MATCH (ac:Activity)<-[:FOLLOWUP]-(o) WHERE ID(ac) = toInteger($id)
			RETURN ID(o) as otherActivityID`,
			map[string]any{
				"id": activityID,
			}, neo4j.EagerResultTransformer)
		if err != nil {
			return err
		}
		// Delete all follow up activities
		for _, record := range result.Records {
			otherActivityID, _, err := neo4j.GetRecordValue[int64](record, "otherActivityID")
			if err != nil {
				return err
			}
			err = DeleteActivityByID(dbC, int(otherActivityID), true)
			if err != nil {
				return err
			}
		}
		// Delete main activity
		err = DeleteActivityByID(dbC, activityID, false)
		if err != nil {
			return err
		}
		return nil
	}
	// If cascade isn't requested, only delete the activity
	_, err := neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
		`MATCH (ac:Activity) WHERE ID(ac) = toInteger($id) WITH ac
		DETACH DELETE ac`,
		map[string]any{
			"id": activityID,
		}, neo4j.EagerResultTransformer)
	if err != nil {
		return err
	}
	log.Printf("deleted activity with id: %d from database", activityID)
	return nil
}

func ModifyActivity(dbC *DBConnection, modifiedActivity *Activity) error {
	/*This functions creates a new activity in the Database*/
	_, err := neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
		`MATCH(ac) WHERE ID(ac) = toInteger($id) with ac
		SET ac.Description = $description, 
		ac.StartTime = datetime($startTime),
		ac.EndTime = datetime($endTime),
		ac.Priority = toInteger($priority),
		ac.Status = toBooleanOrNull($status)`,
		map[string]any{
			"id":          modifiedActivity.ID,
			"description": modifiedActivity.Description,
			"startTime":   modifiedActivity.StartTime,
			"endTime":     modifiedActivity.EndTime,
			"priority":    modifiedActivity.Priority,
			"status":      modifiedActivity.Status,
		}, neo4j.EagerResultTransformer)
	if err != nil {
		return err
	}
	log.Println("modified activity into: ", modifiedActivity)
	return nil
}

func InsertActivityRelation(dbC *DBConnection, mainActivityID int, otherActivityID int) error {
	/*This functions inserts a relation between activities*/
	_, err := neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
		`MATCH(m:Activity) WHERE ID(m) = toInteger($mainId) with m
		MATCH(o:Activity) WHERE ID(o) = toInteger($otherId) with m, o
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
