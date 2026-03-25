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
	setuplogger "github.com/Terminus-Lab/themis/internal/setup/logger"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	verbose bool

	// evaluate flags
	input   string
	output  string
	format  string
	summary string
	saveToDb bool

	// validate flags
	corrThreshold float64
)

func main() {
	log.Logger = setuplogger.New("info")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "themis-cli",
	Short: "Themis CLI for batch conversation evaluation",
	Long: `Themis CLI evaluates multi-turn AI agent conversations in batch.

Supports JSONL input/output and concurrent evaluation with a worker pool.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if err := godotenv.Load(); err != nil {
			log.Warn().Msg("No .env file found, using environment variables")
		}
		if verbose {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
	},
}

var evaluateCmd = &cobra.Command{
	Use:   "evaluate",
	Short: "Evaluate conversations from a JSONL input file",
	Long: `Evaluate multi-turn AI agent conversations from a JSONL input file.

Each line must be a conversation object with 'conversation_id', 'agent', and 'turns' fields.

Examples:
  # Basic evaluation
  themis-cli evaluate -i conversations.jsonl -o results.jsonl

  # Summary output only
  themis-cli evaluate -i conversations.jsonl -f summary

  # JSONL results + separate summary file
  themis-cli evaluate -i conversations.jsonl -o results.jsonl -s summary.json`,
	SilenceUsage: true,
	RunE:         runEvaluate,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate judge accuracy against human annotations",
	Long: `Validate LLM judge accuracy on conversations using human annotations.

Input file must contain conversations with 'human_annotation' field per conversation.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("conversation validation not yet implemented — coming in next release")
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Themis CLI v2.0.0")
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose logging")

	evaluateCmd.Flags().StringVarP(&input, "input", "i", "", "Input JSONL file path (conversation records)")
	evaluateCmd.Flags().StringVarP(&output, "output", "o", "", "Output JSONL file path")
	evaluateCmd.Flags().StringVarP(&format, "format", "f", "jsonl", "Output format: jsonl, summary")
	evaluateCmd.Flags().StringVarP(&summary, "summary", "s", "", "Optional separate summary file")
	evaluateCmd.Flags().BoolVarP(&saveToDb, "save-to-db", "d", false, "Save results to database")
	_ = evaluateCmd.MarkFlagRequired("input")

	validateCmd.Flags().StringVarP(&input, "input", "i", "", "Input file path with human annotations")
	validateCmd.Flags().Float64VarP(&corrThreshold, "correlation-threshold", "c", 0.3, "Kendall's tau threshold")
	_ = validateCmd.MarkFlagRequired("input")

	rootCmd.AddCommand(evaluateCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)
}

func runEvaluate(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

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

	f, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("failed to open input file %q: %w", input, err)
	}
	defer closeFile(f)
	log.Info().Str("file", input).Msg("reading conversation input file")

	reader := batch.NewConversationReader(f, deps.Logger)
	recordsCh := reader.ReadAll(ctx)

	var records []batch.ConversationInputRecord
	annotations := make(map[string]batch.Annotation)
	for record := range recordsCh {
		records = append(records, record)
		if record.Error == nil && (record.HumanLabel != "" || record.HumanScore != nil) {
			annotations[record.Request.ConversationID] = batch.Annotation{
				HumanLabel: record.HumanLabel,
				HumanScore: record.HumanScore,
			}
		}
	}

	log.Info().Int("total", len(records)).Msg("conversation input file parsed")

	if output == "" && format != "summary" {
		return fmt.Errorf("required flag \"output\" not set")
	}

	var outFile io.WriteCloser
	if output == "" {
		outFile = os.Stdout
	} else {
		f2, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("failed to create output file %q: %w", output, err)
		}
		defer closeFile(f2)
		outFile = f2
		log.Info().Str("file", output).Msg("writing to output file")
	}

	workers := getWorkersFromEnv()
	log.Info().Int("workers", workers).Msg("starting conversation worker pool")

	processor := batch.NewConversationProcessor(deps.ConversationEvaluator, workers, deps.Logger)
	results := processor.Process(ctx, records)

	successCount := 0
	errorCount := 0
	var allResults []models.ConversationEvaluationResult

	if format == "summary" {
		summaryWriter := batch.NewConversationSummaryWriter(outFile, deps.Logger)
		for result := range results {
			allResults = append(allResults, result)
			if err := summaryWriter.Write(result); err != nil {
				log.Error().Err(err).Str("conversation_id", result.ConversationID).Msg("failed to add result to summary")
				errorCount++
			} else {
				successCount++
			}
		}
		if len(annotations) > 0 {
			report := batch.ComputeCorrelationReport(allResults, annotations)
			logCorrelationReport(&report)
			summaryWriter.SetCorrelationReport(&report)
		}
		if err := summaryWriter.Close(); err != nil {
			return fmt.Errorf("failed to write summary: %w", err)
		}
	} else {
		encoder := json.NewEncoder(outFile)
		for result := range results {
			allResults = append(allResults, result)
			if err := encoder.Encode(result); err != nil {
				log.Error().Err(err).Str("conversation_id", result.ConversationID).Msg("failed to write result")
				errorCount++
			} else {
				successCount++
			}
		}
		if len(annotations) > 0 {
			report := batch.ComputeCorrelationReport(allResults, annotations)
			logCorrelationReport(&report)
			// Write correlation report as a final JSON line with a marker field
			type correlationReportLine struct {
				Type string `json:"_type"`
				batch.CorrelationReport
			}
			line := correlationReportLine{Type: "correlation_report", CorrelationReport: report}
			if err := encoder.Encode(line); err != nil {
				log.Error().Err(err).Msg("failed to write correlation report line")
			}
		}
	}

	if summary != "" {
		if err := writeSummary(summary, allResults, annotations); err != nil {
			return fmt.Errorf("failed to write summary: %w", err)
		}
	}

	log.Info().
		Int("success", successCount).
		Int("errors", errorCount).
		Dur("duration", time.Since(startTime)).
		Msg("evaluation complete")

	return nil
}

func writeSummary(path string, results []models.ConversationEvaluationResult, annotations map[string]batch.Annotation) error {
	summaryFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer closeFile(summaryFile)

	summaryWriter := batch.NewConversationSummaryWriter(summaryFile, &log.Logger)
	for _, result := range results {
		if err := summaryWriter.Write(result); err != nil {
			log.Error().Err(err).Str("conversation_id", result.ConversationID).Msg("failed to add result to summary")
		}
	}
	if len(annotations) > 0 {
		report := batch.ComputeCorrelationReport(results, annotations)
		summaryWriter.SetCorrelationReport(&report)
	}

	return summaryWriter.Close()
}

func logCorrelationReport(report *batch.CorrelationReport) {
	evt := log.Info().Int("annotated_count", report.AnnotatedCount)
	if report.KendallTau != nil {
		evt = evt.Float64("kendall_tau", *report.KendallTau)
	}
	if report.CohensKappa != nil {
		evt = evt.Float64("cohens_kappa", *report.CohensKappa)
	}
	if report.WeightedKappa != nil {
		evt = evt.Float64("weighted_kappa", *report.WeightedKappa)
	}
	evt.Msg("correlation report")
}

func setupGracefulShutdown() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Warn().Msg("received interrupt signal, finishing current work...")
		cancel()
	}()
	return ctx, cancel
}

func closeFile(f io.Closer) {
	if err := f.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close file")
	}
}

func getWorkersFromEnv() int {
	workers := 5
	if envWorkers := os.Getenv("THEMIS_BATCH_WORKERS"); envWorkers != "" {
		if w, err := strconv.Atoi(envWorkers); err == nil && w > 0 {
			workers = w
		} else {
			log.Warn().Str("value", envWorkers).Msg("invalid THEMIS_BATCH_WORKERS, using default")
		}
	}
	return workers
}
