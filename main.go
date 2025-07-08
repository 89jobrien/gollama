package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings" 
)

const ollamaChatAPI = "http://localhost:11434/api/chat"

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"` 
}

type OllamaChatResponse struct {
	Model     string      `json:"model"`
	CreatedAt string      `json:"created_at"`
	Message   ChatMessage `json:"message"`
	Done      bool        `json:"done"`
}

func main() {
	modelName := flag.String("model", "llama3.2:3b", "The name of the Ollama model to use")
	flag.Parse()
	fmt.Println("------------------------------------------------------------------")
	fmt.Printf("Starting chatbot with model: %s. Type 'exit' or 'quit' to end.\n", *modelName)
	fmt.Println("------------------------------------------------------------------")
	
	var conversationHistory []ChatMessage
	reader := bufio.NewReader(os.Stdin)
	
	for {
		fmt.Print("You: ")
		userInput, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading user input: %v", err)
			continue
		}
		
		userInput = trimNewline(userInput)
		if userInput == "exit" || userInput == "quit" {
			fmt.Println("\nBot: Goodbye!")
			break
		}
		
		conversationHistory = append(conversationHistory, ChatMessage{Role: "user", Content: userInput})
		
		fmt.Print("Bot: ")
		fullResponse, err := streamChatToOllama(*modelName, conversationHistory)
		if err != nil {
			log.Printf("Error getting response from Ollama: %v", err)
			fmt.Println("\nSorry, I encountered an error. Please check the console.")
			
			if len(conversationHistory) > 0 {
				conversationHistory = conversationHistory[:len(conversationHistory)-1]
			}
			continue
		}
		
		conversationHistory = append(conversationHistory, ChatMessage{Role: "assistant", Content: fullResponse})
		fmt.Println() 
	}
}

func streamChatToOllama(modelName string, messages []ChatMessage) (string, error) {
	
	requestData := OllamaChatRequest{
		Model:    modelName,
		Messages: messages,
		Stream:   true,
	}
	
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("error marshalling JSON: %w", err)
	}
	
	req, err := http.NewRequest("POST", ollamaChatAPI, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("received non-OK HTTP status: %s, body: %s", resp.Status, string(bodyBytes))
	}
	
	var fullResponse strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var ollamaResp OllamaChatResponse
		if err := json.Unmarshal(line, &ollamaResp); err != nil {
			
			log.Printf("Could not unmarshal line: %s, error: %v", string(line), err)
			continue
		}
		
		fmt.Print(ollamaResp.Message.Content)
		
		fullResponse.WriteString(ollamaResp.Message.Content)
		
		if ollamaResp.Done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading response stream: %w", err)
	}
	
	return fullResponse.String(), nil
}

func trimNewline(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '\r' {
		s = s[:len(s)-1]
	}
	return s
}