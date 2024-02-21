package main

import (
	"github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Operation struct {
	ID            uint   `gorm:"primaryKey"`
	Name          string `gorm:"not null"`
	ExecutionTime int    `gorm:"not null"`
}

type Expression struct {
	ID         uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4()"`
	Expression string         `gorm:"not null"`
	Status     string         `gorm:"default:'Pending'"`
	Tokens     pq.StringArray `gorm:"type:text[]"`
	Result     *int

	CreatedAt   time.Time
	ProcessAt   *time.Time
	CompletedAt *time.Time
}

type Task struct {
	ID      uuid.UUID `json:"id"`
	Tokens  []string  `json:"tokens"`
	WaitFor int       `json:"waitfor"`
}

var db *gorm.DB

func main() {
	var err error
	databaseURL := os.Getenv("POSTGRES_URL")
	db, err = gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
	err = db.AutoMigrate(&Expression{}, &Operation{})
	if err != nil {
		log.Fatalf("Error migrating the database: %v", err)
	}

	go checkExpressions()
	go checkOperations()

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
			if expression.ProcessAt == nil || time.Since(*expression.ProcessAt) < 20*time.Minute {
				continue
			}

			expression.Status = "Pending"
			expression.ProcessAt = nil
			db.Save(&expression)
		}

		time.Sleep(1 * time.Minute)
	}
}

func checkOperations() {
	for _, operationName := range []string{"+", "-", "*", "/"} {
		var operation Operation
		if err := db.Where("name = ?", operationName).First(&operation).Error; err == nil {
			continue
		}

		operation = Operation{ID: 0, Name: operationName, ExecutionTime: 1000}
		if err := db.Create(&operation).Error; err != nil {
			log.Fatalf("Не удалось создать операции")
		}
	}
}

func addExpression(c *gin.Context) {
	var expression Expression
	if err := c.BindJSON(&expression); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var err error
	expression.Tokens, err = splitExpression(expression.Expression)
	if err != nil || expression.Expression == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Выражение невалидно"})
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

	// Convert the ID parameter to UUID type
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID"})
		return
	}

	var expression Expression
	// Use the UUID variable instead of the id parameter
	if err := db.First(&expression, "id = ?", uid).Error; err != nil {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Operator not found"})
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

	var tokens []string
	tokens, _ = tokenizer(expression.Tokens, nil)

	if len(tokens) < 3 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Operator not found"})
		return
	}

	var operation Operation
	if err := db.Where("name = ?", tokens[2]).First(&operation).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Operator not found"})
		return
	}

	startedAt := time.Now()
	expression.Status = "In progress"
	expression.ProcessAt = &startedAt
	db.Save(&expression)

	c.JSON(http.StatusOK, Task{ID: expression.ID, Tokens: tokens, WaitFor: operation.ExecutionTime})
}

func receiveResult(c *gin.Context) {
	var data struct {
		ID          string `json:"id"`
		WorkerID    int    `json:"workerID"`
		Result      int    `json:"result"`
		CompletedAt time.Time
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert the ID parameter to UUID type
	uid, err := uuid.Parse(data.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID"})
		return
	}

	var expression Expression
	// Use the UUID variable instead of the id parameter
	if err := db.First(&expression, "id = ?", uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expression not found"})
		return
	}

	c.JSON(http.StatusOK, expression)

	_, expression.Tokens = tokenizer(expression.Tokens, &data.Result)

	if len(expression.Tokens) < 2 {
		expression.Status = "Completed"
		expression.Result = &data.Result
		expression.Tokens = nil
	} else {
		expression.Status = "Pending"
	}

	db.Save(&expression)
	c.JSON(http.StatusOK, gin.H{"message": "Result received"})
}
