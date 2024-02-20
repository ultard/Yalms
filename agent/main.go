package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
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
		time.Sleep(4 * time.Second)

		task, err := getTask()
		if !err {
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

func getTask() (Task, bool) {
	fmt.Println("Trying to get tasks from api")
	resp, err := http.Get(os.Getenv("API_URL") + "/task")
	if err != nil {
		return Task{}, false
	}

	defer resp.Body.Close()
	if resp.Status != "200" {
		return Task{}, false
	}

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		fmt.Println("Failed to decode task")
		return Task{}, false
	}

	fmt.Println(resp.Body)
	return task, true
}

func sendTask(result Result) {
	jsonData, err := json.Marshal(result)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Post(os.Getenv("API_URL")+"/result", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
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
