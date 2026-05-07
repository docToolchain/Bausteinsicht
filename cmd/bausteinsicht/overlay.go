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
		Long:  "Load metrics from JSON file and apply heatmap colors to elements. Original colors are saved in metadata.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			metricsPath := args[0]
			modelPath, _ := cmd.Flags().GetString("model")
			metricKey, _ := cmd.Flags().GetString("metric")
			outputPath, _ := cmd.Flags().GetString("output")

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
				if len(mf.Metrics) > 0 && len(mf.Metrics[0].Values) > 0 {
					for k := range mf.Metrics[0].Values {
						metricKey = k
						break
					}
				}
				if metricKey == "" {
					return exitWithCode(fmt.Errorf("no metrics found in file"), 2)
				}
			}

			if err := overlay.Apply(drawioPath, mf, metricKey, overlay.DefaultColorScheme); err != nil {
				return exitWithCode(fmt.Errorf("applying overlay: %w", err), 2)
			}

			format, _ := cmd.Flags().GetString("format")
			if format == "json" {
				out, _ := json.Marshal(map[string]interface{}{
					"status":  "applied",
					"metric":  metricKey,
					"file":    drawioPath,
					"model":   m.Specification,
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

			_, err := model.Load(modelPath)
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			metricsPath := args[0]

			mf, err := overlay.LoadMetricsFile(metricsPath)
			if err != nil {
				return exitWithCode(fmt.Errorf("loading metrics: %w", err), 2)
			}

			format, _ := cmd.Flags().GetString("format")
			if format == "json" {
				out, _ := json.Marshal(map[string]interface{}{
					"source":              mf.Meta.Source,
					"generated":           mf.Meta.Generated,
					"metric_descriptions": mf.Meta.MetricDescriptions,
					"element_count":       len(mf.Metrics),
				})
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "📊 Metrics from: %s (%s)\n\n", mf.Meta.Source, mf.Meta.Generated)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Available metrics (%d elements):\n", len(mf.Metrics))
				for metric, desc := range mf.Meta.MetricDescriptions {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  • %s: %s\n", metric, desc)
				}
			}
			return nil
		},
	}
}
