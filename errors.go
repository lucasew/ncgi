package main

import (
	"log"
)

// ReportError handles non-fatal errors through a centralized mechanism.
// It ensures consistent logging and is the designated injection point
// for external error monitoring systems (like Sentry) in the future.
// It silently returns on nil errors to simplify call-site logic.
func ReportError(err error, metadata map[string]interface{}) {
	if err == nil {
		return
	}
	// In a real application, this would send to Sentry or similar.
	if len(metadata) > 0 {
		log.Printf("ERROR: %v | Metadata: %v", err, metadata)
	} else {
		log.Printf("ERROR: %v", err)
	}
}

// ReportFatal operates identically to ReportError but explicitly
// halts the application via log.Fatal. It should only be used when
// the application enters an unrecoverable state where continuing
// operation would cause data corruption or meaningless behavior.
func ReportFatal(err error, metadata map[string]interface{}) {
	if err != nil {
		ReportError(err, metadata)
		log.Fatal(err)
	}
}
