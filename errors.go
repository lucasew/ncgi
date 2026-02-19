package main

import (
	"log"
)

// ReportError logs the error with metadata.
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

// ReportFatal logs the error and exits.
func ReportFatal(err error, metadata map[string]interface{}) {
	if err != nil {
		ReportError(err, metadata)
		log.Fatal(err)
	}
}
