package main

import (
	"log"
	"slices"
	"time"
)

func checkExpressions() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var expressions []Expression
			if err := db.Where("status = ?", "In progress").Find(&expressions).Error; err != nil {
				continue
			}

			var operations []Operation
			if err := db.Find(&operations).Error; err != nil {
				continue
			}

			for _, expression := range expressions {
				tokens, _ := tokenizer(expression.Tokens, nil)
				operatorID := slices.IndexFunc(operations, func(o Operation) bool { return o.Name == tokens[2] })
				executionTime := operations[operatorID].ExecutionTime

				if expression.ProcessAt == nil ||
					time.Since(*expression.ProcessAt) < (1*time.Minute+time.Duration(executionTime)*time.Millisecond) {
					continue
				}

				expression.Status = "Pending"
				expression.ProcessAt = nil
				db.Save(&expression)
			}

		}
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
