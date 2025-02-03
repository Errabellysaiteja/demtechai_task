Mock AWS SES API
This project is a mock API simulating the behavior of AWS SES (Simple Email Service). The goal is to test email sending without actually using the AWS SES service. Instead, it tracks API usage statistics and mimics the API's contract and behavior.

Features
Email Simulation: Simulates email sending with from, to, subject, and body parameters, without actually sending the email.
Rate Limiting: Enforces a rate limit on the number of emails that can be sent within an hour (maximum 5 emails per hour).
Email Logs: Tracks and logs all email details with timestamps.
Error Handling: Mimics AWS SES error codes like MessageRejected, Throttling, and InvalidParameterValue.
Documentation
AWS SES Documentation
For a detailed overview of AWS SES, refer to the AWS SES API Documentation.

Special Rules (Email Warming-Up)
AWS SES imposes certain restrictions in the initial phase of using the service, often referred to as "email warming-up." This mock API imitates such behavior by limiting the number of emails that can be sent per hour during the early testing phase.

Error Codes
The following AWS SES-like error codes are simulated by the mock API:

MessageRejected: The email message is rejected due to a policy violation.
Throttling: Too many requests; the API rate limit has been exceeded.
InvalidParameterValue: The provided input (e.g., email format) is invalid.
API Endpoints
POST /send-email
Accepts email details and returns a success message if the email is valid and within the rate limit.
{
  "message": "Email sent (mocked successfully)",
  "emails_sent": 3
}

GET /stats
Displays the email sending statistics, including the number of emails sent, remaining allowed emails, and whether the limit has been reached.

Example Response:
{
  "total_emails_sent": 10,
  "emails_sent_last_hour": 5,
  "time_elapsed_since_reset": "0.5 hours",
  "remaining_emails_before_limit": 0,
  "limit_reached": true
}

GET /test
A test endpoint to verify if the API server is running correctly.

Example Response
{
  "message": "Request successful"
}
Setup & Usage
Requirements
Go 1.18 or higher
Gin framework
AWS SDK for Go
Setup Instructions
1.Clone the repository:
2.install dependencies
3.run the server
4.api will be created

Testing the API
Send an Email: Send a POST request to http://localhost:8080/send-email with email details in the body.
View Stats: Send a GET request to http://localhost:8080/stats to view the API usage statistics.
Test Endpoint: Send a GET request to http://localhost:8080/test to ensure the server is working.
Conclusion
This mock AWS SES API allows you to test email-sending functionality without using AWS SES itself. It includes rate limiting, error handling, and logging to simulate real-world scenarios. It's a useful tool for testing before integrating with AWS SES.

