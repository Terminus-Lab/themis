package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/Terminus-Lab/themis/internal/batch"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/setup"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	startTime := time.Now()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	input := flag.String("input", "", "Input file relative path")
	output := flag.String("output", "", "Output file relative path")
	format := flag.String("format", "jsonl", "Output file format. Supported formats: 'jsonl', 'summary'")
	summary := flag.String("summary", "", "Optional separate summary file")
	workers := flag.Int("workers", 5, "Concurrent evaluators workers")
	continueOnError := flag.Bool("continue-on-error", true, "Continue on evaluation failures")
	dryRun := flag.Bool("dry-run", false, "Validate input without evaluating")
	validate := flag.Bool("validate", false, "Validation mode: compute correlation with human annotations")
	corrThreshold := flag.Float64("correlation-threshold", 0.3, "Kendall's tau threshold for validation")

	flag.Parse()

	if *input == "" {
		log.Fatal().Msg("required flag -input not provided")
	}
	formatValidator(format)

	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("No .env file found, using environment variables")
	}

	ctx, cancel := setupGracefulShutdown()
	defer cancel()

	cfg := setup.LoadConfig()

	deps, err := setup.Wire(ctx, cfg, &log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to wire dependencies")
	}

	// Open input file
	var inputFile io.Reader
	if *input == "-" {
		inputFile = os.Stdin
		log.Info().Msg("Reading from stdin")
	} else {
		f, err := os.Open(*input)
		if err != nil {
			log.Fatal().Err(err).Str("file", *input).Msg("Failed to open input file")
		}
		defer f.Close()
		inputFile = f
		log.Info().Str("file", *input).Msg("Reading input file")
	}

	// Read records
	reader := batch.NewReader(inputFile, deps.Logger)
	recordsCh := reader.ReadAll(ctx)

	var records []batch.InputRecord
	for record := range recordsCh {
		records = append(records, record)
	}

	log.Info().Int("total", len(records)).Msg("Input file parsed")

	// Dry run validation
	if *dryRun {
		dryRunAndExit(records)
	}

	// Validation mode
	if *validate {
		runValidationMode(ctx, records, deps, *corrThreshold)
		return
	}

	// Open output file
	var outputFile io.Writer
	if *output == "" {
		outputFile = os.Stdout
		log.Info().Msg("Writing to stdout")
	} else {
		f, err := os.Create(*output)
		if err != nil {
			log.Fatal().Err(err).Str("file", *output).Msg("Failed to create output file")
		}
		defer f.Close()
		outputFile = f
		log.Info().Str("file", *output).Msg("Writing to output file")
	}

	// Create writer
	writer, err := batch.NewWriter(outputFile, *format, deps.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create writer")
	}
	defer writer.Close()

	// Process with worker pool
	processor := batch.NewProcessor(deps.Executor, *workers, deps.Logger)
	results := processor.Process(ctx, records)

	// Write results
	successCount := 0
	errorCount := 0
	var allResults []models.EvaluationResult

	for result := range results {
		allResults = append(allResults, result)

		if err := writer.Write(result); err != nil {
			log.Error().Err(err).Str("id", result.ID).Msg("Failed to write result")
			errorCount++

			if !*continueOnError {
				log.Fatal().Msg("Stopping due to write error")
			}
		} else {
			successCount++
		}
	}

	log.Info().
		Int("success", successCount).
		Int("errors", errorCount).
		Dur("duration", time.Since(startTime)).
		Msg("Processing complete")

	if *summary != "" {
		writeSummary(summary, allResults)
	}

	log.Info().Msg("Batch processing complete")
}

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

func formatValidator(format *string) {
	validFormats := map[string]bool{"jsonl": true, "summary": true}
	if !validFormats[*format] {
		log.Fatal().
			Str("format", *format).
			Msg("Invalid format. Supported: jsonl, summary")
	}
}

func writeSummary(summary *string, results []models.EvaluationResult) {
	summaryFile, err := os.Create(*summary)
	if err != nil {
		log.Fatal().Err(err).Str("file", *summary).Msg("Failed to create summary file")
	}
	defer summaryFile.Close()

	// Create a summary writer and populate it
	summaryWriter := batch.NewSummaryWriter(summaryFile, &log.Logger)
	for _, result := range results {
		if err := summaryWriter.Write(result); err != nil {
			log.Error().Err(err).Str("id", result.ID).Msg("Failed to add result to summary")
		}
	}

	// Close writes the computed stats to the file
	if err := summaryWriter.Close(); err != nil {
		log.Fatal().Err(err).Str("file", *summary).Msg("Failed to write summary file")
	}

	log.Info().Str("file", *summary).Msg("Summary written")
}

func dryRunAndExit(records []batch.InputRecord) {
	errorCount := 0
	for _, record := range records {
		if record.Error != nil {
			log.Error().
				Int("line", record.LineNumber).
				Err(record.Error).
				Msg("Validation error")
			errorCount++
		}
	}

	if errorCount > 0 {
		log.Fatal().Int("errors", errorCount).Msg("Validation failed")
	}

	log.Info().Msg("Validation successful")
	os.Exit(0)
}

func runValidationMode(ctx context.Context, records []batch.InputRecord, deps *setup.Dependencies, threshold float64) {
	log.Info().Msg("Validation mode enabled")

	// Build map of event_id -> human_annotation for O(1) lookup
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
		log.Fatal().
			Int("missing", missingAnnotations).
			Msg("Validation mode requires all records to have 'human_annotation' field")
	}

	log.Info().Int("total", len(records)).Msg("Evaluating records with human annotations...")

	// Evaluate all records
	processor := batch.NewProcessor(deps.Executor, 5, deps.Logger)
	results := processor.Process(ctx, records)

	// Collect annotation pairs using map lookup
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

	// Validate
	validationResult, err := batch.ValidateAnnotations(pairs, threshold)
	if err != nil {
		log.Fatal().Err(err).Msg("Validation failed")
	}

	// Output validation result as JSON to stdout
	validationJSON, err := json.MarshalIndent(validationResult, "", "  ")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to marshal validation result")
	}
	fmt.Println(string(validationJSON))

	// Print summary to stderr (for logging)
	printValidationSummary(validationResult)

	// Write to file as well
	summaryFile := "validation-summary.json"
	if err := os.WriteFile(summaryFile, validationJSON, 0644); err == nil {
		log.Info().Str("file", summaryFile).Msg("Validation summary written")
	}

	// Exit based on result
	if !validationResult.Passed {
		log.Error().
			Float64("tau", validationResult.KendallTau).
			Float64("threshold", threshold).
			Msg("Validation failed: Kendall's tau below threshold")
		log.Error().Msg("Review configs/judges.yaml prompts and re-run validation")
		os.Exit(1)
	}

	log.Info().Msg("LLM judge validated against human annotations")
	log.Info().Msg("Safe to evaluate full dataset with these judge prompts")
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
