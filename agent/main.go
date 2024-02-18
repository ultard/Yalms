package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type Task struct {
	ID      int `json:"id"`
	tokens  []string
	waitFor int
	Result  int `json:"result"`
}

type Result struct {
	ID          int `json:"id"`
	WorkerID    int `json:"workerID"`
	Result      int `json:"result"`
	CompletedAt time.Time
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Initialize number of workers
	numWorkers, err := strconv.Atoi(os.Getenv("NUM_WORKERS"))
	if err != nil {
		numWorkers = 5 // Default to 5 workers
	}

	tasks := make(chan Task)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(tasks, &wg, i)
	}

	// Consume tasks from the server
	for {
		time.Sleep(2 * time.Second)

		task, err := getTask()
		if err != nil {
			continue
		}
		tasks <- task
	}

	// Wait for all workers to finish
	wg.Wait()
}

func worker(tasks <-chan Task, wg *sync.WaitGroup, id int) {
	defer wg.Done()
	for task := range tasks {
		// Perform computation
		result, ok := computeExpression(task.tokens)
		if !ok {
			continue
		}

		sendTask(Result{ID: task.ID, Result: result, WorkerID: id})
	}
}

func getTask() (Task, error) {
	resp, err := http.Get("http://localhost:8080/task")
	if err != nil {
		return Task{}, err
	}
	defer resp.Body.Close()

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return Task{}, err
	}

	return task, nil
}

func sendTask(result Result) Task {
	resp, err := http.Get("http://localhost:8080/result")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		log.Fatal(err)
	}

	return task
}

func computeExpression(tokens []string) (int, bool) {
	var stack []int

	for _, token := range tokens {
		switch token {
		case "+", "-", "*", "/":
			operand2 := stack[len(stack)-1]
			operand1 := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			switch token {
			case "+":
				stack = append(stack, operand1+operand2)
			case "-":
				stack = append(stack, operand1-operand2)
			case "*":
				stack = append(stack, operand1*operand2)
			case "/":
				if operand2 == 0 {
					return 0, false
				}
				stack = append(stack, operand1/operand2)
			}
		default:
			num, _ := strconv.Atoi(token)
			stack = append(stack, num)
		}
	}

	return stack[0], true
}
