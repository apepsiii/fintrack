package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	appLogger *log.Logger
	logMu     sync.Mutex
	logFile   *os.File
)

func initLogger() {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Warning: gagal buat folder logs: %v", err)
		appLogger = log.New(os.Stdout, "", log.LstdFlags)
		return
	}

	today := time.Now().Format("2006-01-02")
	logPath := filepath.Join(logDir, fmt.Sprintf("fintrack_%s.log", today))

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Warning: gagal buka log file: %v", err)
		appLogger = log.New(os.Stdout, "", log.LstdFlags)
		return
	}

	logFile = f
	multi := io.MultiWriter(os.Stdout, f)
	appLogger = log.New(multi, "", log.LstdFlags)
	appLogger.Printf("=== FinTrack started at %s ===", time.Now().Format("2006-01-02 15:04:05"))
}

func closeLogger() {
	if logFile != nil {
		logFile.Close()
	}
}

func logInfo(format string, args ...interface{}) {
	logMu.Lock()
	defer logMu.Unlock()
	if appLogger != nil {
		appLogger.Printf("[INFO] "+format, args...)
	}
}

func logWarn(format string, args ...interface{}) {
	logMu.Lock()
	defer logMu.Unlock()
	if appLogger != nil {
		appLogger.Printf("[WARN] "+format, args...)
	}
}

func logError(format string, args ...interface{}) {
	logMu.Lock()
	defer logMu.Unlock()
	if appLogger != nil {
		appLogger.Printf("[ERROR] "+format, args...)
	}
}

func logTx(userID int, txType, category, amount, note string) {
	logMu.Lock()
	defer logMu.Unlock()
	if appLogger != nil {
		appLogger.Printf("[TXN] user=%d type=%s category=%s amount=%s note=%q", userID, txType, category, amount, note)
	}
}
