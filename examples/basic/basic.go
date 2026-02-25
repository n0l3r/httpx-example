// Package basic demonstrates core httpx features:
// - Creating a client with functional options
// - GET, POST, PUT, PATCH, DELETE requests
// - JSON helpers (GetJSON, PostJSON, PutJSON)
// - Fluent request builder
// - Response helpers
// - Default headers & base URL
// - Context support
// - Form upload (application/x-www-form-urlencoded)
// - Multipart file upload (multipart/form-data)
package basic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/n0l3r/httpx"
)

// --- Data models ---

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Run executes all basic examples against an embedded test server.
func Run() {
	srv := startServer()
	defer srv.Close()

	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  BASIC CLIENT EXAMPLES")
	fmt.Println("═══════════════════════════════════════════")

	exampleNewClient(srv.URL)
	exampleGetJSON(srv.URL)
	examplePostJSON(srv.URL)
	examplePutJSON(srv.URL)
	exampleDelete(srv.URL)
	exampleFluentBuilder(srv.URL)
	exampleResponseHelpers(srv.URL)
	exampleContextDeadline(srv.URL)
	exampleDefaultHeaders(srv.URL)
	exampleBodyForm(srv.URL)
	exampleBodyMultipart(srv.URL)
}

// --- Examples ---

func exampleNewClient(baseURL string) {
	fmt.Println("\n[1] Creating a client with options")

	c, err := httpx.New(
		httpx.WithBaseURL(baseURL),
		httpx.WithTimeout(10*time.Second),
		httpx.WithDefaultHeader("X-App-Name", "httpx-demo"),
		httpx.WithDefaultHeader("Accept", "application/json"),
		httpx.WithConnectionPool(50, 5, 30*time.Second),
	)
	if err != nil {
		fmt.Printf("  ✗ %v\n", err)
		return
	}

	resp, err := c.Get(context.Background(), "/users/1")
	if err != nil {
		fmt.Printf("  ✗ %v\n", err)
		return
	}
	fmt.Printf("  ✓ GET /users/1 → %d (%d bytes)\n", resp.StatusCode(), len(resp.Bytes()))
}

func exampleGetJSON(baseURL string) {
	fmt.Println("\n[2] GetJSON — decode response body into struct")

	c, _ := httpx.New(httpx.WithBaseURL(baseURL))

	var user User
	if err := c.GetJSON(context.Background(), "/users/1", &user); err != nil {
		fmt.Printf("  ✗ %v\n", err)
		return
	}
	fmt.Printf("  ✓ User: id=%d name=%q email=%q\n", user.ID, user.Name, user.Email)
}

func examplePostJSON(baseURL string) {
	fmt.Println("\n[3] PostJSON — send JSON body and decode response")

	c, _ := httpx.New(httpx.WithBaseURL(baseURL))

	req := CreateUserRequest{Name: "Alice", Email: "alice@example.com"}
	var created User
	if err := c.PostJSON(context.Background(), "/users", req, &created); err != nil {
		fmt.Printf("  ✗ %v\n", err)
		return
	}
	fmt.Printf("  ✓ Created: id=%d name=%q\n", created.ID, created.Name)
}

func examplePutJSON(baseURL string) {
	fmt.Println("\n[4] PutJSON — update resource")

	c, _ := httpx.New(httpx.WithBaseURL(baseURL))

	req := CreateUserRequest{Name: "Alice Updated", Email: "alice2@example.com"}
	var updated User
	if err := c.PutJSON(context.Background(), "/users/1", req, &updated); err != nil {
		fmt.Printf("  ✗ %v\n", err)
		return
	}
	fmt.Printf("  ✓ Updated: id=%d name=%q\n", updated.ID, updated.Name)
}

func exampleDelete(baseURL string) {
	fmt.Println("\n[5] Delete — DELETE request")

	c, _ := httpx.New(httpx.WithBaseURL(baseURL))

	resp, err := c.Delete(context.Background(), "/users/1")
	if err != nil {
		fmt.Printf("  ✗ %v\n", err)
		return
	}
	fmt.Printf("  ✓ DELETE /users/1 → %d\n", resp.StatusCode())
}

func exampleFluentBuilder(baseURL string) {
	fmt.Println("\n[6] Fluent request builder")

	c, _ := httpx.New(httpx.WithBaseURL(baseURL))

	req, err := c.NewRequest(context.Background(), "GET", "/users").
		Header("X-Custom-Header", "demo-value").
		Query("page", "1").
		Query("limit", "10").
		Accept("application/json").
		BearerToken("my-secret-token").
		Build()
	if err != nil {
		fmt.Printf("  ✗ build: %v\n", err)
		return
	}

	resp, err := c.Do(req)
	if err != nil {
		fmt.Printf("  ✗ %v\n", err)
		return
	}
	fmt.Printf("  ✓ GET /users?page=1&limit=10 → %d\n", resp.StatusCode())
	fmt.Printf("    Authorization: %s\n", req.Header.Get("Authorization"))
}

func exampleResponseHelpers(baseURL string) {
	fmt.Println("\n[7] Response helpers")

	c, _ := httpx.New(httpx.WithBaseURL(baseURL))

	resp, _ := c.Get(context.Background(), "/users/1")
	fmt.Printf("  StatusCode:    %d\n", resp.StatusCode())
	fmt.Printf("  IsSuccess:     %v\n", resp.IsSuccess())
	fmt.Printf("  IsClientError: %v\n", resp.IsClientError())
	fmt.Printf("  IsServerError: %v\n", resp.IsServerError())
	fmt.Printf("  Body (string): %s\n", strings.TrimSpace(resp.String()))
	fmt.Printf("  Header X-Request-ID: %q\n", resp.Header("X-Request-ID"))

	// EnsureSuccess on 4xx
	resp404, _ := c.Get(context.Background(), "/not-found")
	err := resp404.EnsureSuccess()
	fmt.Printf("  EnsureSuccess on 404: %v\n", err)
}

func exampleContextDeadline(baseURL string) {
	fmt.Println("\n[8] Context deadline / cancellation")

	c, _ := httpx.New()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// /slow returns after 200ms — should be cancelled.
	_, err := c.Get(ctx, baseURL+"/slow")
	if err != nil {
		fmt.Printf("  ✓ Request cancelled as expected: %v\n", err)
		if httpx.IsTimeout(err) {
			fmt.Println("    → classified as: timeout")
		}
	} else {
		fmt.Println("  ✗ expected error, got nil")
	}
}

func exampleDefaultHeaders(baseURL string) {
	fmt.Println("\n[9] Default headers + WithDefaultHeaders")

	c, _ := httpx.New(
		httpx.WithBaseURL(baseURL),
		httpx.WithDefaultHeaders(map[string]string{
			"X-Service":     "demo-svc",
			"X-Environment": "development",
		}),
	)

	resp, _ := c.Get(context.Background(), "/echo-headers")
	var headers map[string]string
	_ = resp.JSON(&headers)
	fmt.Printf("  ✓ X-Service: %q\n", headers["X-Service"])
	fmt.Printf("    X-Environment: %q\n", headers["X-Environment"])
}

func exampleBodyForm(baseURL string) {
	fmt.Println("\n[10] BodyForm — application/x-www-form-urlencoded")

	c, _ := httpx.New()
	resp, err := c.Execute(context.Background(), "POST", baseURL+"/login",
		httpx.WithFormBody(url.Values{
			"username": {"alice"},
			"password": {"secret"},
		}),
	)
	if err != nil {
		fmt.Printf("  ✗ %v\n", err)
		return
	}
	fmt.Printf("  ✓ status=%d  Content-Type sent: application/x-www-form-urlencoded\n", resp.StatusCode())
}

func exampleBodyMultipart(baseURL string) {
	fmt.Println("\n[11] BodyMultipart — multipart/form-data file upload")

	c, _ := httpx.New()
	resp, err := c.Execute(context.Background(), "POST", baseURL+"/upload",
		httpx.WithMultipartBody(
			map[string]string{"title": "my report"},
			[]httpx.FormFile{
				{
					FieldName:   "file",
					FileName:    "report.txt",
					Content:     strings.NewReader("report content here"),
					ContentType: "text/plain",
				},
			},
		),
	)
	if err != nil {
		fmt.Printf("  ✗ %v\n", err)
		return
	}
	fmt.Printf("  ✓ status=%d  file uploaded as multipart/form-data\n", resp.StatusCode())
}

// --- Embedded test server ---

func startServer() *httptest.Server {
	mux := http.NewServeMux()

	alice := User{ID: 1, Name: "Alice", Email: "alice@example.com"}

	mux.HandleFunc("/users/1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(alice)
		case http.MethodPut:
			var req CreateUserRequest
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(User{ID: 1, Name: req.Name, Email: req.Email})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode([]User{alice})
		case http.MethodPost:
			var req CreateUserRequest
			json.NewDecoder(r.Body).Decode(&req)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(User{ID: 2, Name: req.Name, Email: req.Email})
		}
	})

	mux.HandleFunc("/not-found", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	})

	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(200 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		}
	})

	mux.HandleFunc("/echo-headers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		out := make(map[string]string)
		for k, vs := range r.Header {
			if len(vs) > 0 {
				out[k] = vs[0]
			}
		}
		json.NewEncoder(w).Encode(out)
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	return httptest.NewServer(mux)
}
