package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Terminus-Lab/themis/internal/batch"
	"github.com/Terminus-Lab/themis/internal/metrics"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/setup"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	// Global flags (available to all subcommands)
	verbose bool

	// Evaluate command flags
	input   string
	output  string
	format  string
	summary string

	// Validate command flags
	corrThreshold float64
	saveToDb      bool
)

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "themis-cli",
	Short: "Themis CLI for batch evaluation and validation",
	Long: `Themis CLI for batch evaluation and validation of AI agent responses.

Supports JSONL input/output, concurrent evaluation, validation mode,
and Kendall's tau correlation against human annotations.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load .env file for all commands
		if err := godotenv.Load(); err != nil {
			log.Warn().Msg("No .env file found, using environment variables")
		}

		// Setup verbose logging if requested
		if verbose {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
	},
}

// evaluateCmd represents the evaluate command (main functionality)
var evaluateCmd = &cobra.Command{
	Use:   "evaluate",
	Short: "Evaluate responses from input file",
	Long: `Evaluate AI agent responses from a JSONL input file.

Processes records through the Themis evaluation pipeline and writes
results to output file. Supports concurrent evaluation with worker pool.

Examples:
  # Basic evaluation
  themis-cli evaluate -i input.jsonl -o results.jsonl

  # With summary stats file
  themis-cli evaluate -i input.jsonl -o results.jsonl -s summary.json

  # Output as summary only
  themis-cli evaluate -i input.jsonl -o summary.json -f summary`,
	SilenceUsage: true,
	RunE:         runEvaluate,
}

// validateCmd represents the validation command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate judge accuracy against human annotations",
	Long: `Validate LLM judge accuracy using comprehensive metrics.

Computes 3 core metrics:
  1. Kendall's tau (PRIMARY) - Pass/fail decision based on rank correlation
  2. Cohen's Kappa (REPORT) - Categorical agreement accounting for chance
  3. Confusion Matrix (DEBUG) - Per-class precision/recall/F1 scores

Requires input file with 'human_annotation' field for each record.
Pass/fail decision based on Kendall's tau threshold (default: 0.3).

Outputs JSON with all metrics for comprehensive judge evaluation.

Examples:
  # Validate with default threshold (0.3)
  themis-cli validate -i annotated.jsonl -s true

  # Custom threshold (stricter validation)
  themis-cli validate -i annotated.jsonl --correlation-threshold 0.5

  # Save validation report to file
  themis-cli validate -i annotated.jsonl > validation-report.json`,
	SilenceUsage: true,
	RunE:         runValidate,
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Themis CLI v1.1.0")
	},
}

func init() {
	// Global persistent flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose logging")

	// Evaluate command flags
	evaluateCmd.Flags().StringVarP(&input, "input", "i", "", "Input JSONL file path")
	evaluateCmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")
	evaluateCmd.Flags().StringVarP(&format, "format", "f", "jsonl", "Output format: jsonl, summary")
	evaluateCmd.Flags().StringVarP(&summary, "summary", "s", "", "Optional separate summary file")
	evaluateCmd.Flags().BoolVarP(&saveToDb, "save-to-db", "d", false, "Save results to database")

	_ = evaluateCmd.MarkFlagRequired("input")

	// Validate command flags
	validateCmd.Flags().StringVarP(&input, "input", "i", "", "Input file path with human annotations")
	validateCmd.Flags().Float64VarP(&corrThreshold, "correlation-threshold", "c", 0.3, "Kendall's tau threshold")
	validateCmd.Flags().BoolVarP(&saveToDb, "save-to-db", "d", false, "Save results to database")

	_ = validateCmd.MarkFlagRequired("input")

	// Add commands to root
	rootCmd.AddCommand(evaluateCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)
}

func runEvaluate(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	// Validate format
	validFormats := map[string]bool{"jsonl": true, "summary": true}
	if !validFormats[format] {
		return fmt.Errorf("invalid format %q. Supported: jsonl, summary", format)
	}

	ctx, cancel := setupGracefulShutdown()
	defer cancel()

	cfg := setup.LoadConfig()
	if !saveToDb {
		cfg.InMemoryDB = true
	} else {
		if cfg.InMemoryDB {
			log.Fatal().Msg("--save-to-db requires IN_MEMORY_DB=false in your .env")
		}
		if cfg.DBConnectionString == "" {
			log.Fatal().Msg("--save-to-db requires THEMIS_DB_URL to be set in your .env")
		}
	}

	deps, err := setup.Wire(ctx, cfg, &log.Logger)
	if err != nil {
		return fmt.Errorf("failed to wire dependencies: %w", err)
	}

	// Open input file
	f, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("failed to open input file %q: %w", input, err)
	}
	defer closeFile(f)
	log.Info().Str("file", input).Msg("Reading input file")

	// Read records
	reader := batch.NewReader(f, deps.Logger)
	recordsCh := reader.ReadAll(ctx)

	var records []batch.InputRecord
	for record := range recordsCh {
		records = append(records, record)
	}

	log.Info().Int("total", len(records)).Msg("Input file parsed")

	// Require output flag
	if output == "" {
		return fmt.Errorf("required flag \"output\" not set")
	}

	// Open output file
	outFile, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create output file %q: %w", output, err)
	}
	defer closeFile(outFile)
	log.Info().Str("file", output).Msg("Writing to output file")

	// Create writer
	writer, err := batch.NewWriter(outFile, format, deps.Logger)
	if err != nil {
		return fmt.Errorf("failed to create writer: %w", err)
	}
	defer closeFile(writer)

	// Get worker count from env var or use default
	workers := getWorkersFromEnv()
	log.Info().Int("workers", workers).Msg("Starting worker pool")

	// Process with worker pool
	processor := batch.NewProcessor(deps.Executor, workers, deps.Logger)
	results := processor.Process(ctx, records)

	// Write results (always continue on error)
	successCount := 0
	errorCount := 0
	var allResults []models.EvaluationResult

	for result := range results {
		allResults = append(allResults, result)

		if err := writer.Write(result); err != nil {
			log.Error().Err(err).Str("id", result.ID).Msg("Failed to write result")
			errorCount++
		} else {
			successCount++
		}
	}

	log.Info().
		Int("success", successCount).
		Int("errors", errorCount).
		Dur("duration", time.Since(startTime)).
		Msg("Processing complete")

	if summary != "" {
		if err := writeSummary(summary, allResults); err != nil {
			return fmt.Errorf("failed to write summary: %w", err)
		}
	}

	log.Info().Msg("Batch processing complete")
	return nil
}

func runValidate(cmd *cobra.Command, args []string) error {
	log.Info().
		Float64("threshold", corrThreshold).
		Msg("Validation mode enabled")

	ctx, cancel := setupGracefulShutdown()
	defer cancel()

	cfg := setup.LoadConfig()
	if !saveToDb {
		cfg.InMemoryDB = true
	} else {
		if cfg.InMemoryDB {
			log.Fatal().Msg("--save-to-db requires IN_MEMORY_DB=false in your .env")
		}
		if cfg.DBConnectionString == "" {
			log.Fatal().Msg("--save-to-db requires THEMIS_DB_URL to be set in your .env")
		}
	}

	deps, err := setup.Wire(ctx, cfg, &log.Logger)
	if err != nil {
		return fmt.Errorf("failed to wire dependencies: %w", err)
	}

	// Open input file
	f, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("failed to open input file %q: %w", input, err)
	}
	defer closeFile(f)

	// Read records
	reader := batch.NewReader(f, deps.Logger)
	recordsCh := reader.ReadAll(ctx)

	var records []batch.InputRecord
	for record := range recordsCh {
		records = append(records, record)
	}

	// Build annotation map and validate
	annotationMap := make(map[string]string)
	missingAnnotations := 0

	for _, record := range records {
		if record.Request.HumanAnnotation == nil || *record.Request.HumanAnnotation == "" {
			log.Error().
				Int("line", record.LineNumber).
				Str("event_id", record.Request.EventID).
				Msg("Record missing human_annotation")
			missingAnnotations++
		} else {
			annotationMap[record.Request.EventID] = *record.Request.HumanAnnotation
		}
	}

	if missingAnnotations > 0 {
		return fmt.Errorf("validation requires all records to have 'human_annotation' field (missing: %d)", missingAnnotations)
	}

	log.Info().Int("total", len(records)).Msg("Evaluating records with human annotations...")

	// Evaluate all records
	workers := getWorkersFromEnv()
	log.Info().Int("workers", workers).Msg("Starting worker pool")
	processor := batch.NewProcessor(deps.Executor, workers, deps.Logger)
	results := processor.Process(ctx, records)

	// Collect annotation pairs
	var pairs []batch.AnnotationPair
	for result := range results {
		humanAnnotation, ok := annotationMap[result.ID]
		if !ok {
			log.Warn().Str("event_id", result.ID).Msg("No human annotation found for result")
			continue
		}

		pairs = append(pairs, batch.AnnotationPair{
			EventID:         result.ID,
			HumanAnnotation: humanAnnotation,
			LLMVerdict:      result.Verdict,
			Confidence:      result.Confidence,
		})
	}

	log.Info().Msg("Computing validation metrics (Kendall's τ, Cohen's Kappa, Confusion Matrix)...")

	// Validate and output
	return validateAndOutput(pairs, corrThreshold)
}

// Helper functions
func setupGracefulShutdown() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Warn().Msg("Received interrupt signal, finishing current work...")
		cancel()
	}()

	return ctx, cancel
}

func closeFile(f io.Closer) {
	if err := f.Close(); err != nil {
		log.Error().Err(err).Msg("Failed to close file")
	}
}

func getWorkersFromEnv() int {
	workers := 5 // default
	if envWorkers := os.Getenv("THEMIS_BATCH_WORKERS"); envWorkers != "" {
		if w, err := strconv.Atoi(envWorkers); err == nil && w > 0 {
			workers = w
		} else {
			log.Warn().Str("value", envWorkers).Msg("Invalid THEMIS_BATCH_WORKERS, using default")
		}
	}
	return workers
}

func writeSummary(path string, results []models.EvaluationResult) error {
	summaryFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer closeFile(summaryFile)

	summaryWriter := batch.NewSummaryWriter(summaryFile, &log.Logger)
	for _, result := range results {
		if err := summaryWriter.Write(result); err != nil {
			log.Error().Err(err).Str("id", result.ID).Msg("Failed to add result to summary")
		}
	}

	if err := summaryWriter.Close(); err != nil {
		return err
	}

	log.Info().Str("file", path).Msg("Summary written")
	return nil
}

func validateAndOutput(pairs []batch.AnnotationPair, threshold float64) error {
	validationResult, err := batch.ValidateAnnotations(pairs, threshold)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Output JSON to stdout
	validationJSON, err := json.MarshalIndent(validationResult, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal validation result: %w", err)
	}
	fmt.Println(string(validationJSON))

	// Log summary to stderr
	printValidationSummary(validationResult)

	// Write to file
	summaryFile := "validation-summary.json"
	if err := os.WriteFile(summaryFile, validationJSON, 0644); err == nil {
		log.Info().Str("file", summaryFile).Msg("Validation summary written")
	}

	// Exit based on result
	if !validationResult.Passed {
		log.Error().
			Float64("kendall_tau", validationResult.CorrelationMetrics.KendallsTau).
			Float64("threshold", threshold).
			Msg("Validation failed: Kendall's tau below threshold")
		log.Error().Msg("Review configs/judges.yaml prompts and re-run validation")
		os.Exit(1)
	}

	log.Info().Msg("LLM judge validated against human annotations")
	log.Info().Msg("Safe to evaluate full dataset with these judge prompts")
	return nil
}

func printValidationSummary(result *metrics.ValidationResult) {
	status := "PASSED"
	if !result.Passed {
		status = "FAILED"
	}

	log.Info().
		Int("records", result.TotalRecords).
		Float64("kendall_tau", result.CorrelationMetrics.KendallsTau).
		Str("tau_interpretation", result.CorrelationMetrics.Interpretation).
		Float64("cohens_kappa", result.AgreementMetrics.CohensKappa).
		Str("kappa_interpretation", result.AgreementMetrics.Interpretation).
		Float64("threshold", result.Threshold).
		Str("status", status).
		Msg("Validation complete")
}
