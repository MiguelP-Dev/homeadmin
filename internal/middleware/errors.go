package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// AppError represents a structured application error with HTTP status and message.
type AppError struct {
	Status  int
	Message string
	Wrap    error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return e.Message
}

// Unwrap returns the wrapped error for errors.Is/errors.As compatibility.
func (e *AppError) Unwrap() error {
	return e.Wrap
}

// NotFound creates a 404 AppError.
func NotFound(msg string) *AppError {
	return &AppError{Status: fiber.StatusNotFound, Message: msg}
}

// BadRequest creates a 400 AppError.
func BadRequest(msg string) *AppError {
	return &AppError{Status: fiber.StatusBadRequest, Message: msg}
}

// Forbidden creates a 403 AppError.
func Forbidden(msg string) *AppError {
	return &AppError{Status: fiber.StatusForbidden, Message: msg}
}

// Internal creates a 500 AppError.
func Internal(msg string) *AppError {
	return &AppError{Status: fiber.StatusInternalServerError, Message: msg}
}

// ErrorHandler is the centralized Fiber error handler.
// It content-negotiates: JSON for API clients, HTML for browsers.
// For HTMX requests, it sets the HX-Trigger header so client-side JS can show toasts.
func ErrorHandler(c *fiber.Ctx, err error) error {
	// Extract status and message from AppError or fallback to generic error
	status := fiber.StatusInternalServerError
	message := err.Error()

	if e, ok := err.(*AppError); ok {
		status = e.Status
		message = e.Message
	} else if fe, ok := err.(*fiber.Error); ok {
		// Preserve Fiber's own error codes (e.g. fiber.ErrNotFound → 404)
		// instead of collapsing every non-AppError into a generic 500.
		status = fe.Code
		message = fe.Message
	}

	// Log 5xx server errors; 4xx are client errors (no log needed)
	if status >= 500 {
		fmt.Printf("[ERROR] %d %s\n", status, message)
	}

	// For HTMX requests, set HX-Trigger header with toast data
	if isHtmxRequest(c) {
		c.Set("HX-Trigger", fmt.Sprintf(`{"message":"%s","level":"error"}`, message))
	}

	// Content negotiation via Accept header
	accept := c.Get("Accept")
	if containsHTML(accept) {
		c.Type("html")
		return c.Status(status).SendString(errorPageHTML(status, message))
	}

	return c.Status(status).JSON(fiber.Map{"error": message})
}

// isHtmxRequest checks if the request originated from HTMX.
func isHtmxRequest(c *fiber.Ctx) bool {
	return c.Get("HX-Request") == "true"
}

// containsHTML checks if the Accept header indicates HTML preference.
func containsHTML(accept string) bool {
	if accept == "" {
		return true // default to HTML (browser default)
	}
	for i := 0; i < len(accept); i++ {
		if accept[i] == '/' {
			// crude check: look for "html" substring
			for j := i; j < len(accept); j++ {
				if j+4 <= len(accept) && accept[j:j+4] == "html" {
					return true
				}
				if accept[j] == ',' || accept[j] == ';' || accept[j] == ' ' {
					break
				}
			}
		}
	}
	return false
}

// errorPageHTML renders a minimal HTML error page.
func errorPageHTML(status int, message string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Error %d</title>
    <style>
        body { font-family: system-ui, sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: #f5f5f5; }
        .error-container { text-align: center; padding: 2rem; background: white; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        h1 { font-size: 4rem; margin: 0; color: #e74c3c; }
        p { font-size: 1.2rem; color: #666; margin: 1rem 0; }
        a { color: #2d6a4f; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="error-container">
        <h1>%d</h1>
        <p>%s</p>
        <p><a href="/">Home</a></p>
    </div>
</body>
</html>`, status, status, message)
}
