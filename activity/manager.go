package activity

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"net/http"

	"github.com/gin-gonic/gin"
)

var dbC *DBConnection

func Main() {
	var err error
	dbC, err = CreateConnection()
	if err != nil {
		panic(err)
	}
	defer dbC.CloseConnection()
	router := gin.Default()
	router.PUT("/create", CreateMainActivity)
	router.PUT("/create/:id", CreateFollowUpActivity)
	router.GET("/activity/:id", ActivityByID)
	router.POST("/activity", ActivitiesByFilter)
	router.DELETE("/activity", DeleteActivity)
	router.PUT("/move", EditActivity)

	router.Run("localhost:8080")
}

func CreateMainActivity(c *gin.Context) {
	var activity Activity
	if err := c.BindJSON(&activity); err != nil {
		log.Fatal(err)
	}
	if err := InsertActivity(dbC, &activity); err != nil {
		log.Fatal(err)
	}
	jsonContent, err := json.Marshal(activity)
	if err != nil {
		log.Fatal(err)
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": string(jsonContent),
	})
	fmt.Println("Created new Activity")
}

func CreateFollowUpActivity(c *gin.Context) {
	var activity Activity
	mainActivityID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Fatal(err)
	}
	if err := c.BindJSON(&activity); err != nil {
		log.Fatal(err)
	}
	if err := InsertActivity(dbC, &activity); err != nil {
		log.Fatal(err)
	}
	InsertActivityRelation(dbC, mainActivityID, activity.ID)
	jsonContent, err := json.Marshal(activity)
	if err != nil {
		log.Fatal(err)
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": string(jsonContent),
	})
	fmt.Println("Created Follow up Activity for Activity")
}

func ActivityByID(c *gin.Context) {
	ID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Fatal(err)
	}
	// Returns the activity given an ID
	activity, err := FetchActivityByID(dbC, ID)
	if err != nil {
		log.Fatal(err)
	}
	jsonContent, err := json.Marshal(activity)
	if err != nil {
		log.Fatal(err)
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": string(jsonContent),
	})
	fmt.Println("Fetched activity from ID")
}

func ActivitiesByFilter(c *gin.Context) {
	// Returns activities
	var filter Filter
	if err := c.BindJSON(&filter); err != nil {
		log.Fatal(err)
	}
	activities, err := FetchActivityByFilter(dbC, &filter)
	if err != nil {
		log.Fatal(err)
	}
	jsonContent, err := json.Marshal(activities)
	if err != nil {
		log.Fatal(err)
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": string(jsonContent),
	})
	fmt.Println("Fetched activities")
}

func EditActivity(c *gin.Context) {
	// Pass the modified activity with old id.
	// Old id will be used to delete the old activity
	var activity Activity
	if err := c.BindJSON(&activity); err != nil {
		log.Fatal(err)
	}
	//Delete activity
	if err := DeleteActivityByID(dbC, activity.ID); err != nil {
		log.Fatal(err)
	}
	//Create new activity
	if err := InsertActivity(dbC, &activity); err != nil {
		log.Fatal(err)
	}
	jsonContent, err := json.Marshal(activity)
	if err != nil {
		log.Fatal(err)
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": string(jsonContent),
	})
	fmt.Println("Edited Activity")
}

func DeleteActivity(c *gin.Context) {
	// Pass the whole object, just to be sure of the deletion
	var activity Activity
	if err := c.BindJSON(activity); err != nil {
		log.Fatal(err)
	}
	//Delete activity
	if err := DeleteActivityByID(dbC, activity.ID); err != nil {
		log.Fatal(err)
	}
	jsonContent, err := json.Marshal(activity)
	if err != nil {
		log.Fatal(err)
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": string(jsonContent),
	})
}
