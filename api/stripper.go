package main

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

func splitExpression(expression string) []string {
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

	postfix := infixToPostfix(tokens)
	return postfix
}
