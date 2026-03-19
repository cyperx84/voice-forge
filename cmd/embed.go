package cmd

import (
	"fmt"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/embedding"
	"github.com/spf13/cobra"
)

var (
	embedModel     string
	embedReference string
)

var embedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Generate voice embeddings for recordings",
	Long: `Generate speaker embeddings using resemblyzer or speechbrain.
Optionally compare each recording to a reference file.

Examples:
  forge embed
  forge embed --model resemblyzer --reference ~/ref.wav`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		model := embedModel
		if model == "" {
			model = cfg.Embedding.Model
		}

		inputDir := config.ExpandPath("~/.forge/processed")
		outputDir := config.ExpandPath("~/.forge/embeddings")

		fmt.Printf("Generating embeddings (model: %s) from %s\n", model, inputDir)

		result, err := embedding.Generate(inputDir, outputDir, model)
		if err != nil {
			return fmt.Errorf("embed: %w", err)
		}

		fmt.Printf("Generated %d embeddings. Mean self-similarity: %.2f\n",
			result.Count, result.MeanSimilarity)

		if embedReference != "" {
			refPath := config.ExpandPath(embedReference)
			store, err := embedding.LoadStore(fmt.Sprintf("%s/embeddings.json", outputDir))
			if err != nil {
				return fmt.Errorf("load embeddings: %w", err)
			}

			tool := model
			sims, err := embedding.CompareToReference(store, refPath, outputDir, tool)
			if err != nil {
				return fmt.Errorf("compare to reference: %w", err)
			}

			var total float64
			for _, sim := range sims {
				total += sim
			}
			mean := total / float64(len(sims))
			fmt.Printf("Mean similarity to reference: %.2f\n", mean)
		}

		return nil
	},
}

func init() {
	embedCmd.Flags().StringVar(&embedModel, "model", "", "embedding model: resemblyzer or speechbrain (default: from config)")
	embedCmd.Flags().StringVar(&embedReference, "reference", "", "reference audio file for similarity comparison")
	rootCmd.AddCommand(embedCmd)
}
