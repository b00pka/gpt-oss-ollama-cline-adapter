package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"flag"
)

// nopCloser wraps a bytes.Reader to implement io.ReadCloser
type nopCloser struct {
	reader *bytes.Reader
}

func (nc *nopCloser) Read(p []byte) (n int, err error) {
	return nc.reader.Read(p)
}

func (nc *nopCloser) Close() error {
	return nil
}

// Environment variables
var (
	targetBaseURL = os.Getenv("TARGET_BASE_URL")
	listenHost    = os.Getenv("TOOL_CALL_ADAPTER_HOST")
	listenPort    = os.Getenv("TOOL_CALL_ADAPTER_PORT")
)

// Grammar file path (can be set via --config flag or environment variable)
var grammarFilePath string

// ChatCompletionRequest represents the request body for OpenAI-compatible chat completions
type ChatCompletionRequest struct {
	Model    string                       `json:"model"`
	Messages []ChatMessage                `json:"messages"`
	Tools    []Tool                       `json:"tools,omitempty"`
	ToolChoice interface{}                  `json:"tool_choice,omitempty"`
	Stream   bool                         `json:"stream,omitempty"`
	Options  map[string]interface{}       `json:"options,omitempty"`
}

// ChatMessage represents a message in the chat
type ChatMessage struct {
	Role    string  `json:"role"`
	Content string  `json:"content"`
	Name    *string `json:"name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool call
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Tool represents a tool definition
type Tool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Parameters  map[string]interface{} `json:"parameters"`
	} `json:"function"`
}

// ChatCompletionResponse represents the response from OpenAI-compatible completions
type ChatCompletionResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []Choice     `json:"choices"`
	Usage   *Usage       `json:"usage,omitempty"`
}

// Choice represents a choice in the response
type Choice struct {
	Index        int       `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason *string   `json:"finish_reason,omitempty"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// loadGrammar loads the Cline grammar from the file
func loadGrammar() string {
	grammarPath := grammarFilePath
	if grammarPath == "" {
		grammarPath = os.Getenv("GRAMMAR_FILE_PATH")
	}
	if grammarPath == "" {
		grammarPath = "/app/cline.gbnf"
	}
	
	data, err := ioutil.ReadFile(grammarPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not read grammar file: %v\n", err)
		fmt.Fprintf(os.Stderr, "Warning: using embedded grammar\n")
		return `root ::= analysis? start final .+
analysis ::= "<|channel|>analysis<|message|>" ( [^<] | "<" [^|] | "<|" [^e] )* "<|end|>"
start ::= "<|start|>assistant"
final ::= "<|channel|>final<|message|>"`
	}
	return string(data)
}

// handleProxyRequest handles all incoming requests and proxies them to the target
func handleProxyRequest(w http.ResponseWriter, r *http.Request) {
	// Parse the target URL
	targetURL, err := url.Parse(targetBaseURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid target URL: %v", err), http.StatusInternalServerError)
		return
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Modify the request if needed
	if r.Method == http.MethodPost {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading request body: %v", err), http.StatusBadRequest)
			return
		}
		r.Body.Close()
		r.Body = &nopCloser{reader: bytes.NewReader(body)}

		// Parse the request body
		var req ChatCompletionRequest
		if err := json.Unmarshal(body, &req); err == nil {
			// Add the grammar to the options if not already present
			if req.Options == nil {
				req.Options = make(map[string]interface{})
			}
			if _, hasGrammar := req.Options["grammar"]; !hasGrammar {
				req.Options["grammar"] = loadGrammar()
				// Re-encode the modified request body
				newBody, jsonErr := json.Marshal(req)
				if jsonErr == nil {
					r.Body = &nopCloser{reader: bytes.NewReader(newBody)}
					r.ContentLength = int64(len(newBody))
					r.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))
				}
			}
		}
	}

	// Proxy the request
	proxy.ServeHTTP(w, r)
}

func main() {
	// Define command-line flags
	flag.StringVar(&grammarFilePath, "config", "", "Path to grammar file (.gbnf)")
	flag.Parse()

	// Validate environment variables
	if targetBaseURL == "" {
		targetBaseURL = "http://ollama:11434/v1"
	}
	if listenHost == "" {
		listenHost = "0.0.0.0"
	}
	if listenPort == "" {
		listenPort = "8000"
	}

	// Print configuration
	fmt.Printf("Starting GPT-OSS Cline Adapter Proxy\n")
	fmt.Printf("  Target Base URL: %s\n", targetBaseURL)
	fmt.Printf("  Listening on: %s:%s\n", listenHost, listenPort)
	fmt.Printf("  Grammar file: %s\n", grammarFilePath)

	// Handle all routes with the proxy
	http.HandleFunc("/", handleProxyRequest)

	// Start the server
	addr := fmt.Sprintf("%s:%s", listenHost, listenPort)
	fmt.Printf("Server starting on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}