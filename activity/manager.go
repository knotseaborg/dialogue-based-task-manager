package activity

import (
	"fmt"
	"log"
	"strconv"
	"time"

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
	router.PUT("/activity/create", CreateMainActivity)
	router.PUT("/followup/create/:id", CreateFollowUpActivity)
	router.GET("/activity/:id", ActivityByID)
	router.POST("/activity", ActivitiesByFilter)
	router.GET("/followup/:id", FollowUpActivitiesByID)
	router.DELETE("/activity/:id", DeleteActivity)
	router.PUT("activity/edit", EditActivity)
	router.GET("/user/time", Now)

	router.Run("localhost:8080")
}

func Now(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"time":    time.Now().Format(TIMEFORMAT),
		"message": fmt.Sprint("time right now is ", time.Now()),
	})
}

func CreateMainActivity(c *gin.Context) {
	var activity Activity
	if err := c.BindJSON(&activity); err != nil {
		log.Fatal(err)
	}
	if err := InsertActivity(dbC, &activity); err != nil {
		log.Fatal(err)
	}
	c.JSON(http.StatusOK, gin.H{
		"code":     http.StatusOK,
		"activity": activity,
		"message":  "stored activity in database",
	})
	log.Println("stored activity: ", activity)
}

func CreateFollowUpActivity(c *gin.Context) {
	var activity Activity
	mainActivityID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Fatal(err)
	}
	// Check if the main activity exists
	mainActivity, err := FetchActivityByID(dbC, mainActivityID)
	if err != nil {
		switch err.(type) {
		default:
			log.Fatal(err)
		case NoActivityError:
			c.JSON(http.StatusOK, gin.H{
				"code":     http.StatusNotFound,
				"activity": "",
				"message":  "main activity not found in database",
			})
			log.Printf("main activity with id: %d not found in database", mainActivityID)
			return
		}
	}
	if err := c.BindJSON(&activity); err != nil {
		log.Fatal(err)
	}
	if err := InsertActivity(dbC, &activity); err != nil {
		log.Fatal(err)
	}
	InsertActivityRelation(dbC, mainActivityID, activity.ID)
	c.JSON(http.StatusOK, gin.H{
		"code":     http.StatusOK,
		"activity": activity,
		"message":  fmt.Sprintf("stored follow-up activity with id: %d, in database for activity with id: %d", activity.ID, mainActivityID),
	})
	log.Println("stored follow-up activity with id: ", activity, "for activity: ", mainActivity)
}

func ActivityByID(c *gin.Context) {
	ID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Fatal(err)
	}
	// Returns the activity given an ID
	activity, err := FetchActivityByID(dbC, ID)
	if err != nil {
		switch err.(type) {
		default:
			log.Fatal(err)
		case NoActivityError:
			c.JSON(http.StatusOK, gin.H{
				"code":     http.StatusNotFound,
				"activity": "",
				"message":  fmt.Sprintf("activty with id: %d, does not exist in the database", ID),
			})
			log.Printf("activty with id %d does not exist in the database", ID)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"code":     http.StatusOK,
		"activity": activity,
		"message":  fmt.Sprintf("fetched activity details from database where activity id is %d", ID),
	})
	log.Println("fetched activity from database using ID")
}

func FollowUpActivitiesByID(c *gin.Context) {
	ID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Fatal(err)
	}
	// Returns the activity given an ID
	_, err = FetchActivityByID(dbC, ID)
	if err != nil {
		switch err.(type) {
		default:
			log.Fatal(err)
		case NoActivityError:
			c.JSON(http.StatusOK, gin.H{
				"code":     http.StatusNotFound,
				"activity": "",
				"message":  fmt.Sprintf("activty with id: %d, does not exist in the database", ID),
			})
			log.Printf("activty with id %d does not exist in the database", ID)
			return
		}
	}
	activities, err := FetchFollowUpActivitiesByID(dbC, ID)
	if err != nil {
		switch err.(type) {
		default:
			log.Fatal(err)
		case NoActivityError:
			c.JSON(http.StatusOK, gin.H{
				"code":       http.StatusNotFound,
				"activities": activities,
				"message":    fmt.Sprintf("activty with id: %d, does not exist in the database", ID),
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"code":       http.StatusOK,
		"activities": activities,
		"message":    fmt.Sprintf("fetched activity details from database where activity id is %d", ID),
	})
	log.Println("fetched activity from database using ID")
}

func ActivitiesByFilter(c *gin.Context) {
	/* Returns activities which satisfy the search conditions in the filter */
	var filter Filter
	if err := c.BindJSON(&filter); err != nil {
		log.Fatal(err)
	}
	activities, err := FetchActivitiesByFilter(dbC, &filter)
	if err != nil {
		switch err.(type) {
		default:
			log.Fatal(err)
		case NoActivityError:
			c.JSON(http.StatusOK, gin.H{
				"code":       http.StatusNotFound,
				"activities": "",
				"message":    "activty with the filter does not exist in the database",
			})
			log.Printf("activty with the filter does not exist in the database")
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"code":       http.StatusOK,
		"activities": activities,
		"message":    "fetched activities from database using filter",
	})
	fmt.Println("fetched activities: ", len(activities), ", using filter", filter)
}

func EditActivity(c *gin.Context) {
	/*Edit activity details*/
	var activity Activity
	if err := c.BindJSON(&activity); err != nil {
		log.Fatal(err)
	}
	if err := ModifyActivity(dbC, &activity); err != nil {
		log.Fatal(err)
	}
	log.Println("modified activity into: ", activity)
	c.JSON(http.StatusOK, gin.H{
		"code":     http.StatusOK,
		"activity": activity,
		"message":  fmt.Sprintf("edited activity with id: %d, in database", activity.ID),
	})
}

func DeleteActivity(c *gin.Context) {
	// Pass the whole object, just to be sure of the deletion
	mainActivityID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Fatal(err)
	}
	//Delete activity
	if err := DeleteActivityByID(dbC, mainActivityID, true); err != nil {
		log.Fatal(err)
	}
	log.Printf("deleted activity with id: %d from database", mainActivityID)
	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": fmt.Sprintf("deleted activity with id: %d", mainActivityID),
	})
}
