package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Email structure
type Email struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	Time    string   `json:"time"`
}

var emailLogs []Email
var emailCount = 0
var startTime = time.Now()

const maxEmailsPerHour = 5 // Change limit if needed

// Rate Limiter per IP
type RateLimiter struct {
	visitors map[string]*rate.Limiter
	mu       sync.Mutex
}

// Initialize rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*rate.Limiter),
	}
}

// Get limiter for an IP address
func (r *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	// If IP exists, return existing limiter
	if limiter, exists := r.visitors[ip]; exists {
		return limiter
	}

	// Otherwise, create a new limiter (e.g., 5 requests per minute)
	limiter := rate.NewLimiter(5, 5)
	r.visitors[ip] = limiter
	return limiter
}

// Middleware for rate limiting
func rateLimitMiddleware(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := rl.GetLimiter(ip)

		// If user exceeds rate limit, block the request
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests. Please try again later."})
			c.Abort()
			return
		}
		c.Next()
	}
}

// Save email log
func saveEmailLog(email Email) error {
	logs, err := loadEmailLogs()
	if err != nil {
		return err
	}

	logs = append(logs, email)

	file, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("email_logs.json", file, 0644)
}

// Load email logs
func loadEmailLogs() ([]Email, error) {
	file, err := os.ReadFile("email_logs.json")
	if err != nil {
		return []Email{}, nil
	}

	var logs []Email
	err = json.Unmarshal(file, &logs)
	if err != nil {
		return []Email{}, err
	}

	return logs, nil
}

// Validate email format
func isValidEmail(email string) bool {
	regex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(regex)
	return re.MatchString(email)
}

// Send email using AWS SES
func sendEmailWithSES(email Email) error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"), // Change to your AWS region
	})
	if err != nil {
		return err
	}

	svc := ses.New(sess)

	// Convert recipient list to AWS SES format
	toAddresses := make([]*string, len(email.To))
	for i, recipient := range email.To {
		toAddresses[i] = aws.String(recipient)
	}

	// Email input
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses: toAddresses,
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Text: &ses.Content{
					Data: aws.String(email.Body),
				},
			},
			Subject: &ses.Content{
				Data: aws.String(email.Subject),
			},
		},
		Source: aws.String(email.From),
	}

	// Send the email
	_, err = svc.SendEmail(input)
	return err
}

func main() {
	// Initialize rate limiter
	rateLimiter := NewRateLimiter()

	// Initialize Gin router
	r := gin.Default()
	r.Use(rateLimitMiddleware(rateLimiter)) // Apply rate limiting

	// Email sending route
	r.POST("/send-email", func(c *gin.Context) {
		// Reset email count if 1 hour has passed
		if time.Since(startTime).Hours() >= 1 {
			startTime = time.Now()
			emailCount = 0
		}

		// Parse JSON request
		var email Email
		if err := c.ShouldBindJSON(&email); err != nil {
			c.JSON(400, gin.H{"error": "Invalid email format. Please provide a valid JSON."})
			return
		}

		// Validate email addresses
		if !isValidEmail(email.From) {
			c.JSON(400, gin.H{"error": "Invalid sender email format."})
			return
		}
		for _, recipient := range email.To {
			if !isValidEmail(recipient) {
				c.JSON(400, gin.H{"error": "Invalid recipient email format."})
				return
			}
		}

		// Check email sending limit
		if emailCount >= maxEmailsPerHour {
			c.JSON(429, gin.H{"error": "Email limit exceeded. Try again later."})
			return
		}

		// Send email using AWS SES
		err := sendEmailWithSES(email)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to send email via AWS SES."})
			return
		}

		// Log the email
		email.Time = time.Now().Format(time.RFC3339)
		err = saveEmailLog(email)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to save email log."})
			return
		}

		// Increment email count
		emailCount++

		c.JSON(200, gin.H{"message": "Email sent successfully via AWS SES", "emails_sent": emailCount})
	})

	// API usage stats
	r.GET("/stats", func(c *gin.Context) {
		elapsedTime := time.Since(startTime).Hours()
		remainingEmails := maxEmailsPerHour - emailCount
		if remainingEmails < 0 {
			remainingEmails = 0
		}

		c.JSON(200, gin.H{
			"total_emails_sent":             emailCount,
			"emails_sent_last_hour":         emailCount,
			"time_elapsed_since_reset":      fmt.Sprintf("%.2f hours", elapsedTime),
			"remaining_emails_before_limit": remainingEmails,
			"limit_reached":                 emailCount >= maxEmailsPerHour,
		})
	})

	// Test endpoint
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Request successful"})
	})

	// Start the server
	r.Run(":8080")
}
