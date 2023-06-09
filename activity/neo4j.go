package activity

import (
	"context"
	"fmt"
	"os"
	"strings"

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
				"text": strings.ToLower(keyword),
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
				"name":   strings.ToLower(person.Name),
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
		"MATCH (ac:Activity) WHERE id(ac) = toInteger($id) RETURN ac, toSTRING(ac.StartTime) as StartTime, toSTRING(ac.EndTime) as EndTime",
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
	startTime, _, err := neo4j.GetRecordValue[string](result.Records[0], "StartTime")
	if err != nil {
		return nil, err
	}
	activity.StartTime = startTime
	endTime, _, err := neo4j.GetRecordValue[string](result.Records[0], "EndTime")
	if err != nil {
		return nil, err
	}
	activity.EndTime = endTime
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

func FetchActivitiesByFilter(dbC *DBConnection, filter *Filter) ([]Activity, error) {
	/*This function fetches activity by filters*/
	activities := make([]Activity, 0)
	hasDefault := false
	// Default values for TimeBounds
	if filter.StartTimeBounds.isEmpty() {
		hasDefault = true
		filter.StartTimeBounds.LowerBound = "2000-11-22T18:59:00.000+0900"
		filter.StartTimeBounds.UpperBound = "2099-11-22T18:59:00.000+0900"
	}

	if filter.EndTimeBounds.isEmpty() {
		hasDefault = true
		filter.EndTimeBounds.LowerBound = "2000-11-22T18:59:00.000+0900"
		filter.EndTimeBounds.UpperBound = "2099-11-22T18:59:00.000+0900"
	}

	for i := range filter.Keywords {
		filter.Keywords[i] = strings.ToLower(filter.Keywords[i])
	}

	for i := range filter.Participants {
		filter.Participants[i] = strings.ToLower(filter.Participants[i])
	}

	var query string
	if hasDefault {
		query = `MATCH (ac:Activity) WHERE ac.StartTime >= DATETIME($startTimeLower) AND ac.StartTime <= DATETIME($startTimeUpper)
		AND ac.EndTime >= DATETIME($endTimeLower) AND ac.EndTime <= DATETIME($endTimeUpper) WITH ac
		MATCH (kw:Keyword) WHERE SIZE($keywords) = 0 OR toLower(kw.Text) IN $keywords WITH kw, ac
		MATCH (kw)-[:INDEX]-(ac) WITH ac
		MATCH (p:Person) WHERE SIZE($participants) = 0 OR toLower(p.Name) IN $participants WITH ac, p
		MATCH (p)-[:PARTICIPANT]-(ac) WHERE SIZE($status) = 0 OR ac.Status IN $status
		RETURN DISTINCT ID(ac) as acID`
	} else {
		query = `MATCH (ac:Activity) WHERE ac.StartTime >= DATETIME($startTimeLower) AND ac.StartTime <= DATETIME($startTimeUpper)
		OR ac.EndTime >= DATETIME($endTimeLower) AND ac.EndTime <= DATETIME($endTimeUpper) WITH ac
		MATCH (kw:Keyword) WHERE SIZE($keywords) = 0 OR toLower(kw.Text) IN $keywords WITH kw, ac
		MATCH (kw)-[:INDEX]-(ac) WITH ac
		MATCH (p:Person) WHERE SIZE($participants) = 0 OR toLower(p.Name) IN $participants WITH ac, p
		MATCH (p)-[:PARTICIPANT]-(ac) WHERE SIZE($status) = 0 OR ac.Status IN $status
		RETURN DISTINCT ID(ac) as acID`
	}

	result, err := neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
		query,
		map[string]any{
			"participants":   filter.Participants,
			"keywords":       filter.Keywords,
			"startTimeLower": filter.StartTimeBounds.LowerBound,
			"startTimeUpper": filter.StartTimeBounds.UpperBound,
			"endTimeLower":   filter.EndTimeBounds.LowerBound,
			"endTimeUpper":   filter.EndTimeBounds.UpperBound,
			"status":         filter.Status,
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

func FetchFollowUpActivitiesByID(dbC *DBConnection, mainActivityID int) ([]Activity, error) {
	/*This function fetches follow-up activities for a given activity by id*/
	activities := make([]Activity, 0)

	result, err := neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
		`MATCH (mac:Activity)<-[:FOLLOWUP]-(oac:Activity) WHERE ID(mac) = toInteger($mainActivityID)
		return DISTINCT ID(oac) as oacID`,
		map[string]any{
			"mainActivityID": mainActivityID,
		}, neo4j.EagerResultTransformer)
	if err != nil {
		return nil, err
	}

	if len(result.Records) == 0 {
		return nil, NoActivityError{}
	}
	for _, record := range result.Records {
		otherActivityID, _, err := neo4j.GetRecordValue[int64](record, "oacID")
		if err != nil {
			return nil, fmt.Errorf("could not find activity node")
		}
		activity, err := FetchActivityByID(dbC, int(otherActivityID))
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
	return nil
}

func ModifyActivity(dbC *DBConnection, modifiedActivity *Activity) error {
	activity, err := FetchActivityByID(dbC, modifiedActivity.ID)
	if err != nil {
		return err
	}
	modifiedMap := map[string]any{"id": modifiedActivity.ID}
	if modifiedActivity.Description == "" {
		modifiedMap["description"] = activity.Description
	} else {
		modifiedMap["description"] = modifiedActivity.Description
	}
	if modifiedActivity.StartTime == "" {
		modifiedMap["startTime"] = activity.StartTime
	} else {
		modifiedMap["startTime"] = modifiedActivity.StartTime
	}
	if modifiedActivity.EndTime == "" {
		modifiedMap["endTime"] = activity.EndTime
	} else {
		modifiedMap["endTime"] = modifiedActivity.EndTime
	}
	if modifiedActivity.Priority == 0 {
		modifiedMap["priority"] = activity.Priority
	} else {
		modifiedMap["priority"] = modifiedActivity.Priority
	}
	if !modifiedActivity.Status { //Activities can only be set to complete
		modifiedMap["status"] = activity.Status
	} else {
		modifiedMap["status"] = modifiedActivity.Status
	}
	fmt.Print(activity)
	/*This functions creates a new activity in the Database*/
	_, err = neo4j.ExecuteQuery(dbC.Context, dbC.Driver,
		`MATCH(ac:Activity) WHERE ID(ac) = toInteger($id) with ac
		SET ac.Description = $description, 
		ac.StartTime = datetime($startTime),
		ac.EndTime = datetime($endTime),
		ac.Priority = toInteger($priority),
		ac.Status = toBooleanOrNull($status)`,
		modifiedMap, neo4j.EagerResultTransformer)
	if err != nil {
		return err
	}
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
