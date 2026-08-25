package middleware

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/homeadmin/internal/i18n"
)

// AppError represents a structured application error with HTTP status and message.
type AppError struct {
	Status  int
	Message string
	Key     string
	Args    []any
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

// Unprocessable creates a 422 AppError (valid request, invalid form payload).
func Unprocessable(msg string) *AppError {
	return &AppError{Status: fiber.StatusUnprocessableEntity, Message: msg}
}

// Forbidden creates a 403 AppError.
func Forbidden(msg string) *AppError {
	return &AppError{Status: fiber.StatusForbidden, Message: msg}
}

// Internal creates a 500 AppError.
func Internal(msg string) *AppError {
	return &AppError{Status: fiber.StatusInternalServerError, Message: msg}
}

// Keyed creates an AppError identified by an i18n translation key.
// The Message field is left empty; ErrorHandler translates via lang at response time.
func Keyed(status int, key string, args ...any) *AppError {
	return &AppError{Status: status, Key: key, Args: args}
}

// translate resolves the display message for an AppError.
// For keyed errors it uses i18n.Tf; for raw errors it returns the Message as-is.
func translate(lang string, e *AppError) string {
	if e.Key != "" {
		return i18n.Tf(lang, e.Key, e.Args...)
	}
	return e.Message
}

// ErrorHandler is the centralized Fiber error handler.
// It content-negotiates: JSON for API clients, HTML for browsers.
// For HTMX requests, it sets the HX-Trigger header so client-side JS can show toasts.
func ErrorHandler(c *fiber.Ctx, err error) error {
	// Extract status and message from AppError or fallback to generic error
	status := fiber.StatusInternalServerError
	message := err.Error()
	lang := "en"

	if e, ok := err.(*AppError); ok {
		status = e.Status
		// Resolve lang from Locals (set by middleware/auth.go WU-2)
		if l, ok := c.Locals("lang").(string); ok && l != "" {
			lang = l
		}
		message = translate(lang, e)
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

	// For HTMX requests, set HX-Trigger header with toast data.
	// json.Marshal keeps the payload valid even when the message contains
	// quotes or other characters that raw fmt.Sprintf interpolation would break.
	if isHtmxRequest(c) {
		trigger, jsonErr := json.Marshal(hxTrigger{Message: message, Level: "error"})
		if jsonErr == nil {
			c.Set("HX-Trigger", string(trigger))
		}
	}

	// Content negotiation via Accept header
	accept := c.Get("Accept")
	if containsHTML(accept) {
		c.Type("html")
		return c.Status(status).SendString(errorPageHTML(status, message, lang))
	}

	return c.Status(status).JSON(fiber.Map{"error": message})
}

// hxTrigger is the JSON payload sent in the HX-Trigger header so client-side
// JS can render a toast for HTMX requests.
type hxTrigger struct {
	Message string `json:"message"`
	Level   string `json:"level"`
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

// errorPageHTML renders a minimal HTML error page. Title and home-link labels
// localize through the dead keys error.title/error.home; lang is resolved by
// the caller from c.Locals("lang").
func errorPageHTML(status int, message string, lang string) string {
	title := fmt.Sprintf("%s %d", i18n.T(lang, "error.title"), status)
	home := i18n.T(lang, "error.home")
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
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
        <p><a href="/">%s</a></p>
    </div>
</body>
</html>`, lang, title, status, message, home)
}
