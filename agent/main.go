package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type Task struct {
	ID      string   `json:"id"`
	Tokens  []string `json:"tokens"`
	WaitFor int      `json:"waitfor"`
}

type Result struct {
	ID          string `json:"id"`
	WorkerID    int    `json:"workerID"`
	Result      int    `json:"result"`
	CompletedAt time.Time
}

func main() {
	// Initialize number of workers
	numWorkers, err := strconv.Atoi(os.Getenv("NUM_WORKERS"))
	if err != nil {
		numWorkers = 5 // Default to 5 workers
	}

	tasks := make(chan Task)
	answers := make(chan Result)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(tasks, answers, &wg, i)
	}

	go tasker(tasks)
	go sender(answers)

	wg.Wait()
}

func tasker(tasks chan<- Task) {
	for {
		time.Sleep(4 * time.Second)
		//fmt.Println("Trying to get tasks from api")

		resp, err := http.Get(os.Getenv("API_URL") + "/task")
		if err != nil {
			fmt.Println(err)
			continue
		}

		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			continue
		}

		var task Task
		if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
			fmt.Println("Failed to decode task")
			continue
		}

		fmt.Println(task)
		tasks <- task
	}
}

func sender(answers chan Result) {
	for answer := range answers {
		time.Sleep(4 * time.Second)
		fmt.Println("Trying to send task to api")

		jsonData, err := json.Marshal(answer)
		if err != nil {
			answers <- answer
			continue
		}

		resp, err := http.Post(os.Getenv("API_URL")+"/result", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			answers <- answer
			continue
		}

		defer resp.Body.Close()
	}
}

func worker(tasks <-chan Task, answers chan Result, wg *sync.WaitGroup, id int) {
	defer wg.Done()
	for task := range tasks {
		fmt.Println(task)
		time.Sleep(time.Duration(task.WaitFor) * time.Millisecond)
		result, ok := computeExpression(task.Tokens)
		if !ok {
			continue
		}

		res := Result{ID: task.ID, Result: result, WorkerID: id, CompletedAt: time.Now()}
		answers <- res
	}
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
