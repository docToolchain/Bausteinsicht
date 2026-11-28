package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/docToolchain/Bausteinsicht/internal/model"
	"github.com/docToolchain/Bausteinsicht/internal/overlay"
	"github.com/spf13/cobra"
)

func newOverlayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "overlay",
		Short: "Apply or remove metric heatmap overlays on architecture diagrams",
		Long:  "Load external metrics (error rate, coverage, etc.) from JSON and overlay them as a heatmap on draw.io elements. Original styles are preserved.",
	}

	cmd.AddCommand(newOverlayApplyCmd())
	cmd.AddCommand(newOverlayRemoveCmd())
	cmd.AddCommand(newOverlayListCmd())

	return cmd
}

func newOverlayApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply <metrics-file>",
		Short: "Apply metric heatmap to draw.io diagram",
		Long: "Load metrics from JSON file and apply heatmap colors to elements. Original colors are saved in metadata.\n\n" +
			"Metrics file: " + overlay.FormatHint + ".\n" +
			"Without --metric the first metric by name is used (see 'overlay list'). Entries for elements that are not in the model are reported as warnings.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			metricsPath := args[0]
			modelPath, _ := cmd.Flags().GetString("model")
			metricKey, _ := cmd.Flags().GetString("metric")
			outputPath, _ := cmd.Flags().GetString("output")

			modelPath, err := resolveModelPath(modelPath)
			if err != nil {
				return err
			}
			m, err := model.Load(modelPath)
			if err != nil {
				return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
			}

			drawioPath := outputPath
			if drawioPath == "" {
				drawioPath = filepath.Join(filepath.Dir(modelPath), "architecture.drawio")
			}

			mf, err := overlay.LoadMetricsFile(metricsPath)
			if err != nil {
				return exitWithCode(fmt.Errorf("loading metrics: %w", err), 2)
			}

			if metricKey == "" {
				if metricKey = overlay.DefaultMetric(mf); metricKey == "" {
					return exitWithCode(fmt.Errorf("no metrics found in file; %s", overlay.FormatHint), 2)
				}
			}

			known := make(map[string]bool)
			flat, _ := model.FlattenElements(m)
			for id := range flat {
				known[id] = true
			}
			for _, id := range overlay.UnknownElementIDs(mf, known) {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "WARNING: metrics for element %q ignored: it is not in the model\n", id)
			}

			if err := overlay.Apply(drawioPath, mf, metricKey, overlay.DefaultColorScheme); err != nil {
				return exitWithCode(fmt.Errorf("applying overlay: %w", err), 2)
			}

			format, _ := cmd.Flags().GetString("format")
			if format == "json" {
				out, _ := json.Marshal(map[string]interface{}{
					"status": "applied",
					"metric": metricKey,
					"file":   drawioPath,
					"model":  m.Specification,
				})
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✅ Overlay applied: %s (metric: %s)\n", drawioPath, metricKey)
			}
			return nil
		},
	}
	cmd.Flags().String("metric", "", "Metric key to visualize (default: first available)")
	cmd.Flags().String("output", "", "Output draw.io file (default: architecture.drawio)")
	return cmd
}

func newOverlayRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove metric overlay from diagram (restore original colors)",
		RunE: func(cmd *cobra.Command, args []string) error {
			modelPath, _ := cmd.Flags().GetString("model")
			outputPath, _ := cmd.Flags().GetString("output")

			modelPath, err := resolveModelPath(modelPath)
			if err != nil {
				return err
			}
			_, err = model.Load(modelPath)
			if err != nil {
				return exitWithCode(fmt.Errorf("loading model: %w", err), 2)
			}

			drawioPath := outputPath
			if drawioPath == "" {
				drawioPath = filepath.Join(filepath.Dir(modelPath), "architecture.drawio")
			}

			if err := overlay.Remove(drawioPath); err != nil {
				return exitWithCode(fmt.Errorf("removing overlay: %w", err), 2)
			}

			format, _ := cmd.Flags().GetString("format")
			if format == "json" {
				out, _ := json.Marshal(map[string]interface{}{
					"status": "removed",
					"file":   drawioPath,
				})
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✅ Overlay removed: %s (original colors restored)\n", drawioPath)
			}
			return nil
		},
	}
	cmd.Flags().String("output", "", "Output draw.io file (default: architecture.drawio)")
	return cmd
}

func newOverlayListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <metrics-file>",
		Short: "List available metrics in file",
		Long:  "List the metrics of a metrics file with their range and the number of elements. Metrics file: " + overlay.FormatHint + ".",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			metricsPath := args[0]

			mf, err := overlay.LoadMetricsFile(metricsPath)
			if err != nil {
				return exitWithCode(fmt.Errorf("loading metrics: %w", err), 2)
			}

			summaries := overlay.Summaries(mf)
			format, _ := cmd.Flags().GetString("format")
			if format == "json" {
				type metricJSON struct {
					Name        string  `json:"name"`
					Description string  `json:"description,omitempty"`
					Min         float64 `json:"min"`
					Max         float64 `json:"max"`
					Elements    int     `json:"elements"`
				}
				metrics := make([]metricJSON, 0, len(summaries))
				for _, s := range summaries {
					metrics = append(metrics, metricJSON{s.Name, s.Description, s.Min, s.Max, s.Elements})
				}
				out, _ := json.Marshal(map[string]interface{}{
					"source":              mf.Meta.Source,
					"generated":           mf.Meta.Generated,
					"metric_descriptions": mf.Meta.MetricDescriptions,
					"element_count":       len(mf.Metrics),
					"metrics":             metrics,
				})
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			} else {
				header := "📊 Metrics"
				if mf.Meta.Source != "" {
					header += " from: " + mf.Meta.Source
				}
				if mf.Meta.Generated != "" {
					header += " (" + mf.Meta.Generated + ")"
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n\nAvailable metrics (%d elements):\n", header, len(mf.Metrics))
				for _, s := range summaries {
					line := fmt.Sprintf("  • %s: min %g, max %g (%d elements)", s.Name, s.Min, s.Max, s.Elements)
					if s.Description != "" {
						line += " — " + s.Description
					}
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
				}
			}
			return nil
		},
	}
}
