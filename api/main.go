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
	ID         uint   `gorm:"primaryKey"`
	Expression string `gorm:"not null"`
	Status     string `gorm:"default:'Pending'"`
	Result     *int

	CreatedAt   time.Time
	ProcessAt   *time.Time
	CompletedAt *time.Time
}

type Task struct {
	ID          uint   `gorm:"primaryKey"`
	first       int    `gorm:"not null"`
	second      int    `gorm:"not null"`
	operator    string `gorm:"not null"`
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

func main() {
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

func addExpression(c *gin.Context) {
	var expression Expression
	if err := c.BindJSON(&expression); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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
	var operation Operation
	if err := c.BindJSON(&operation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.Update("ExecutionTime", operation.ExecutionTime)
	c.JSON(http.StatusOK, operation)
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
