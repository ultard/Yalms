package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Operation struct {
	gorm.Model
	ID            uint   `gorm:"primaryKey"`
	Name          string `gorm:"not null"`
	ExecutionTime int    `gorm:"not null"`
}

type Expression struct {
	ID         uint     `gorm:"primaryKey"`
	Expression string   `gorm:"not null"`
	Status     string   `gorm:"default:'Pending'"`
	tokens     []string `gorm:"default:[]"`
	Result     *int

	CreatedAt   time.Time
	ProcessAt   *time.Time
	CompletedAt *time.Time
}

type Task struct {
	ID          uint `gorm:"primaryKey"`
	tokens      int
	Result      *int
	CreatedAt   time.Time
	CompletedAt *time.Time
}

var db *gorm.DB

func init() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Cannot load .env file: %v", err)
	}
}

func checkDatabase() {
	databaseURL := fmt.Sprintf("postgres://%s:%s@%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_HOST"))
	database, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	_ = database.Exec(fmt.Sprintf("CREATE DATABASE %s;", os.Getenv("POSTGRES_DB")))
}

func main() {
	checkDatabase()

	var err error
	databaseURL := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_DB"))
	db, err = gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	err = db.AutoMigrate(&Expression{}, &Operation{}, &Task{})
	if err != nil {
		log.Fatalf("Error migrating the database: %v", err)
	}

	go checkExpressions()

	// Initialize Gin router
	router := gin.Default()

	// Define routes
	router.POST("/expressions", addExpression)
	router.GET("/expressions", listExpressions)
	router.GET("/expressions/:id", getExpressionByID)
	router.GET("/operations", listOperations)
	router.POST("/operations", setOperations)
	router.GET("/task", getTask)
	router.POST("/result", receiveResult)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	err = router.Run(":" + port)
	if err != nil {
		return
	}

}

func checkExpressions() {
	for {
		var expressions []Expression
		if err := db.Where("status = ?", "In progress").Find(&expressions).Error; err != nil {
			time.Sleep(10 * time.Second)
			continue
		}

		for _, expression := range expressions {
			if expression.ProcessAt != nil && time.Since(*expression.ProcessAt) < 20*time.Minute {
				continue
			}

			expression.Status = "Pending"
			expression.ProcessAt = nil
		}

		time.Sleep(1 * time.Minute)
	}
}

func addExpression(c *gin.Context) {
	var expression Expression
	if err := c.BindJSON(&expression); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	expression.tokens = splitExpression(expression.Expression)
	if err := db.Create(&expression).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, expression)
}

func listExpressions(c *gin.Context) {
	var expressions []Expression
	db.Find(&expressions)

	c.JSON(http.StatusOK, expressions)
}

func getExpressionByID(c *gin.Context) {
	id := c.Param("id")
	var expression Expression
	if err := db.First(&expression, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expression not found"})
		return
	}

	c.JSON(http.StatusOK, expression)
}

func listOperations(c *gin.Context) {
	var operations []Operation
	db.Find(&operations)

	c.JSON(http.StatusOK, operations)
}

func setOperations(c *gin.Context) {
	var data Operation
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var operation Operation
	if err := db.Where("name = ?", data.Name).First(&operation).Error; err != nil {
		if err := db.Create(&data).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, operation)
		return
	}

	operation.ExecutionTime = data.ExecutionTime
	db.Save(&operation)

	c.JSON(http.StatusCreated, operation)
}

func getTask(c *gin.Context) {
	var expression Expression
	if err := db.Where("status = ?", "Pending").First(&expression).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No pending tasks found"})
		return
	}

	startedAt := time.Now()
	expression.Status = "In progress"
	expression.ProcessAt = &startedAt
	db.Save(&expression)

	c.JSON(http.StatusOK, expression)
}

func receiveResult(c *gin.Context) {
	var data struct {
		ID     uint `json:"id"`
		Result int  `json:"result"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var expression Expression
	if err := db.First(&expression, data.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expression not found"})
		return
	}

	expression.Result = &data.Result
	expression.Status = "Completed"
	db.Save(&expression)
	c.JSON(http.StatusOK, gin.H{"message": "Result received"})
}
