// Package mocktest demonstrates httpx mock transport for testing:
// - MockTransport with OnGet / OnPost / OnPut / OnDelete handlers
// - NewJSONResponse / NewResponse helpers
// - CallCount tracking
// - Simulating errors and edge cases
// - Writing table-driven tests with mock
package mocktest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/n0l3r/httpx"
	"github.com/n0l3r/httpx/mock"
)

// Run executes all mock examples.
func Run() {
	fmt.Println("\n═══════════════════════════════════════════")
	fmt.Println("  MOCK TRANSPORT EXAMPLES")
	fmt.Println("═══════════════════════════════════════════")

	exampleBasicMock()
	exampleMockError()
	exampleMockMultiMethod()
	exampleMockTableDriven()
	exampleMockCallCount()
	exampleMockDefault()
}

// [1] Basic MockTransport usage.
func exampleBasicMock() {
	fmt.Println("\n[1] Basic MockTransport — intercept GET /users")

	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	mt := mock.NewMockTransport().
		OnGet("/users", func(req *http.Request) (*mock.Response, error) {
			return mock.NewJSONResponse(200, []User{
				{ID: 1, Name: "Alice"},
				{ID: 2, Name: "Bob"},
			}), nil
		})

	c, _ := httpx.New(httpx.WithTransport(mt))

	resp, err := c.Get(context.Background(), "http://api.example.com/users")
	if err != nil {
		fmt.Printf("  ✗ %v\n", err)
		return
	}

	var users []User
	resp.JSON(&users)
	fmt.Printf("  ✓ status=%d, got %d users\n", resp.StatusCode(), len(users))
	for _, u := range users {
		fmt.Printf("    - %s (id=%d)\n", u.Name, u.ID)
	}
}

// [2] Simulating errors.
func exampleMockError() {
	fmt.Println("\n[2] Mock network error and 4xx/5xx responses")

	mt := mock.NewMockTransport().
		OnGet("/broken", func(req *http.Request) (*mock.Response, error) {
			return nil, fmt.Errorf("connection refused")
		}).
		OnGet("/unauthorized", func(req *http.Request) (*mock.Response, error) {
			return mock.NewJSONResponse(401, map[string]string{"error": "unauthorized"}), nil
		}).
		OnGet("/server-error", func(req *http.Request) (*mock.Response, error) {
			return mock.NewJSONResponse(500, map[string]string{"error": "internal server error"}), nil
		})

	c, _ := httpx.New(httpx.WithTransport(mt))

	// Network error
	_, err := c.Get(context.Background(), "http://api.example.com/broken")
	fmt.Printf("  network error: %v\n", err)

	// 401
	resp, _ := c.Get(context.Background(), "http://api.example.com/unauthorized")
	fmt.Printf("  unauthorized: status=%d IsClientError=%v\n", resp.StatusCode(), resp.IsClientError())

	// 500
	resp, _ = c.Get(context.Background(), "http://api.example.com/server-error")
	fmt.Printf("  server error: status=%d IsServerError=%v\n", resp.StatusCode(), resp.IsServerError())
}

// [3] Mock multiple methods.
func exampleMockMultiMethod() {
	fmt.Println("\n[3] Mock multiple HTTP methods")

	type Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	mt := mock.NewMockTransport().
		OnGet("/items/1", func(req *http.Request) (*mock.Response, error) {
			return mock.NewJSONResponse(200, Item{ID: 1, Name: "Widget"}), nil
		}).
		OnPost("/items", func(req *http.Request) (*mock.Response, error) {
			return mock.NewJSONResponse(201, Item{ID: 2, Name: "Gadget"}), nil
		}).
		OnPut("/items/1", func(req *http.Request) (*mock.Response, error) {
			return mock.NewJSONResponse(200, Item{ID: 1, Name: "Widget v2"}), nil
		}).
		OnDelete("/items/1", func(req *http.Request) (*mock.Response, error) {
			return &mock.Response{StatusCode: 204}, nil
		})

	c, _ := httpx.New(httpx.WithTransport(mt))
	base := "http://api.example.com"

	// GET
	resp, _ := c.Get(context.Background(), base+"/items/1")
	var item Item
	resp.JSON(&item)
	fmt.Printf("  GET  → %d %s\n", resp.StatusCode(), item.Name)

	// POST
	resp, _ = c.Post(context.Background(), base+"/items", httpx.WithJSONBody(Item{Name: "Gadget"}))
	resp.JSON(&item)
	fmt.Printf("  POST → %d %s\n", resp.StatusCode(), item.Name)

	// PUT
	resp, _ = c.Put(context.Background(), base+"/items/1", httpx.WithJSONBody(Item{Name: "Widget v2"}))
	resp.JSON(&item)
	fmt.Printf("  PUT  → %d %s\n", resp.StatusCode(), item.Name)

	// DELETE
	resp, _ = c.Delete(context.Background(), base+"/items/1")
	fmt.Printf("  DEL  → %d\n", resp.StatusCode())
}

// [4] Table-driven test scenario.
func exampleMockTableDriven() {
	fmt.Println("\n[4] Table-driven scenarios")

	type testCase struct {
		name     string
		path     string
		wantCode int
		wantErr  bool
	}

	cases := []testCase{
		{"success", "/ok", 200, false},
		{"not found", "/missing", 404, false},
		{"server error", "/error", 500, false},
		{"network fail", "/fail", 0, true},
	}

	mt := mock.NewMockTransport().
		OnGet("/ok", func(_ *http.Request) (*mock.Response, error) {
			return mock.NewResponse(200, []byte(`"ok"`)), nil
		}).
		OnGet("/missing", func(_ *http.Request) (*mock.Response, error) {
			return mock.NewResponse(404, []byte(`"not found"`)), nil
		}).
		OnGet("/error", func(_ *http.Request) (*mock.Response, error) {
			return mock.NewResponse(500, []byte(`"error"`)), nil
		}).
		OnGet("/fail", func(_ *http.Request) (*mock.Response, error) {
			return nil, fmt.Errorf("dial tcp: connection refused")
		})

	c, _ := httpx.New(httpx.WithTransport(mt))

	for _, tc := range cases {
		resp, err := c.Get(context.Background(), "http://api.example.com"+tc.path)
		gotErr := err != nil
		gotCode := 0
		if resp != nil {
			gotCode = resp.StatusCode()
		}

		ok := gotErr == tc.wantErr && (tc.wantErr || gotCode == tc.wantCode)
		mark := "✓"
		if !ok {
			mark = "✗"
		}
		fmt.Printf("  %s %s: code=%d err=%v\n", mark, tc.name, gotCode, gotErr)
	}
}

// [5] Call count tracking.
func exampleMockCallCount() {
	fmt.Println("\n[5] CallCount — verify how many times routes were hit")

	mt := mock.NewMockTransport().
		OnGet("/health", func(_ *http.Request) (*mock.Response, error) {
			return mock.NewResponse(200, nil), nil
		})

	c, _ := httpx.New(httpx.WithTransport(mt))

	for range 5 {
		c.Get(context.Background(), "http://api.example.com/health")
	}

	fmt.Printf("  ✓ /health called %d time(s)\n", mt.CallCount())
	fmt.Printf("    Total recorded requests: %d\n", len(mt.Requests))
}

// [6] Default handler — catch-all.
func exampleMockDefault() {
	fmt.Println("\n[6] Default handler — catch-all for unknown routes")

	mt := mock.NewMockTransport().
		OnGet("/known", func(_ *http.Request) (*mock.Response, error) {
			return mock.NewResponse(200, []byte(`"known"`)), nil
		})

	// Default catches everything else
	mt.Default = func(req *http.Request) (*mock.Response, error) {
		return mock.NewJSONResponse(404, map[string]string{
			"error": "route not found: " + req.URL.Path,
		}), nil
	}

	c, _ := httpx.New(httpx.WithTransport(mt))

	resp, _ := c.Get(context.Background(), "http://api.example.com/known")
	fmt.Printf("  /known     → %d\n", resp.StatusCode())

	resp, _ = c.Get(context.Background(), "http://api.example.com/anything/else")
	fmt.Printf("  /other     → %d %s\n", resp.StatusCode(), resp.String())
}
