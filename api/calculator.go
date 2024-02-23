package main

import (
	"fmt"
	"unicode"
)

// Функция для определения приоритета операторов
func precedence(operator rune) int {
	switch operator {
	case '+', '-':
		return 1
	case '*', '/':
		return 2
	default:
		return 0 // для скобок
	}
}

// Функция для преобразования инфиксного выражения в постфиксное
func infixToPostfix(tokens []string) []string {
	var output []string
	var operatorStack []string

	for _, token := range tokens {
		switch token {
		case "+", "-", "*", "/":
			for len(operatorStack) > 0 && precedence(rune(operatorStack[len(operatorStack)-1][0])) >= precedence(rune(token[0])) {
				output = append(output, operatorStack[len(operatorStack)-1])
				operatorStack = operatorStack[:len(operatorStack)-1]
			}
			operatorStack = append(operatorStack, token)
		case "(":
			operatorStack = append(operatorStack, token)
		case ")":
			for operatorStack[len(operatorStack)-1] != "(" {
				output = append(output, operatorStack[len(operatorStack)-1])
				operatorStack = operatorStack[:len(operatorStack)-1]
			}
			operatorStack = operatorStack[:len(operatorStack)-1] // Убираем "(" из стека
		default:
			output = append(output, token)
		}
	}

	output = append(output, operatorStack...)

	return output
}

func areBracketsBalanced(expression string) bool {
	var stack []rune
	for _, char := range expression {
		switch char {
		case '(':
			stack = append(stack, char)
		case ')':
			if len(stack) == 0 {
				return false
			}
			stack = stack[:len(stack)-1]
		}
	}
	return len(stack) == 0
}

// Проверка на валидность выражения
func isValidExpression(expression string) bool {
	// Проверка сбалансированности скобок
	if !areBracketsBalanced(expression) {
		return false
	}

	// Проверка на недопустимые символы и некорректное использование операторов
	previousChar := ' '
	for i, char := range expression {
		if !unicode.IsDigit(char) && char != '+' && char != '-' && char != '*' && char != '/' && char != '(' && char != ')' && char != ' ' {
			return false // Недопустимый символ
		}

		if char == '+' || char == '*' || char == '/' {
			if i == 0 || previousChar == '+' || previousChar == '-' || previousChar == '*' || previousChar == '/' || previousChar == '(' {
				return false // Некорректное положение оператора
			}
		}

		if char != ' ' {
			previousChar = char
		}
	}

	// Проверка, что выражение не заканчивается оператором
	if previousChar == '+' || previousChar == '-' || previousChar == '*' || previousChar == '/' {
		return false
	}

	return true
}

func splitExpression(expression string) ([]string, error) {
	if !isValidExpression(expression) {
		return nil, fmt.Errorf("invalid expression")
	}

	var tokens []string
	token := ""
	for _, char := range expression {
		if char == ' ' {
			continue
		} else if char == '+' || char == '-' || char == '*' || char == '/' || char == '(' || char == ')' {
			if token != "" {
				tokens = append(tokens, token)
				token = ""
			}
			tokens = append(tokens, string(char))
		} else {
			token += string(char)
		}
	}
	if token != "" {
		tokens = append(tokens, token)
	}

	tokens = infixToPostfix(tokens)
	return tokens, nil
}

func tokenizer(tokens []string, result *float64) ([]string, []string) {
	var first, stack []string

	for i, token := range tokens {
		switch token {
		case "+", "-", "*", "/":
			stack = append(stack, token)

			if len(first) == 0 {
				first = stack[i-2:]
				stack = append([]string{}, stack[:i-2]...)

				if result != nil {
					stack = append([]string{fmt.Sprintf("%f", *result)}, stack...)
				}
			}
		default:
			stack = append(stack, token)
		}
	}

	return first, stack
}
