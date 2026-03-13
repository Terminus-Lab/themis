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
	RunE: runEvaluate,
}

// validateCmd represents the validation command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate judge accuracy against human annotations",
	Long: `Validate LLM judge accuracy using Kendall's tau correlation.

Requires input file with 'human_annotation' field for each record.
Computes correlation between human annotations and LLM verdicts.
Recommended threshold: 0.3 (moderate agreement).

Examples:
  # Validate with default threshold (0.3)
  themis-cli validate -i annotated.jsonl

  # Custom threshold
  themis-cli validate -i annotated.jsonl --correlation-threshold 0.5

  # Output to file
  themis-cli validate -i annotated.jsonl > validation-result.json`,
	RunE: runValidate,
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

	evaluateCmd.MarkFlagRequired("input")

	// Validate command flags
	validateCmd.Flags().StringVarP(&input, "input", "i", "", "Input file path with human annotations")
	validateCmd.Flags().Float64Var(&corrThreshold, "correlation-threshold", 0.3, "Kendall's tau threshold")

	validateCmd.MarkFlagRequired("input")

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
	log.Info().Msg("Validation mode enabled")

	ctx, cancel := setupGracefulShutdown()
	defer cancel()

	cfg := setup.LoadConfig()
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

	log.Info().Msg("Computing Kendall's correlation...")

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
		return fmt.Errorf("validation failed: Kendall's tau (%.3f) below threshold (%.3f)",
			validationResult.KendallTau, threshold)
	}

	log.Info().Msg("LLM judge validated against human annotations")
	log.Info().Msg("Safe to evaluate full dataset with these judge prompts")
	return nil
}

func printValidationSummary(result *batch.ValidationResult) {
	status := "PASSED"
	if !result.Passed {
		status = "FAILED"
	}

	log.Info().
		Int("records", result.TotalRecords).
		Int("agreement", result.AgreementCount).
		Float64("agreement_rate", result.AgreementRate).
		Float64("kendall_tau", result.KendallTau).
		Float64("threshold", result.Threshold).
		Str("status", status).
		Str("interpretation", result.Interpretation).
		Msg("Validation complete")
}
