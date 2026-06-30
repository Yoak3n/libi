package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"mirror/internal/analysis"
	mirrorconfig "mirror/internal/config"
	"mirror/internal/pipeline"
	"mirror/internal/report"
	"mirror/internal/store"

	"github.com/urfave/cli/v3"
)

func init() {
	mirrorconfig.Init()
}

func main() {
	cmd := &cli.Command{
		Name:    "mirror",
		Version: "0.1.0",
		Usage:   "Bilibili 二游社区跨群体互动分析平台",
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			store.Init() // 连接 shared DB（读取 troll 数据用）
			return ctx, nil
		},
		Commands: []*cli.Command{
			analyzeCommand(),
			reportCommand(),
			trendCommand(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return cli.ShowAppHelp(cmd)
		},
	}
	sort.Sort(cli.FlagsByName(cmd.Flags))
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func analyzeCommand() *cli.Command {
	return &cli.Command{
		Name:    "analyze",
		Aliases: []string{"a"},
		Usage:   "分析二游社区跨群体互动",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "communities",
				Aliases:  []string{"c"},
				Usage:    "目标社区列表，逗号分隔",
				Required: false,
			},
			&cli.StringFlag{
				Name:    "keywords",
				Aliases: []string{"k"},
				Usage:   "搜索关键词",
			},
			&cli.StringFlag{
				Name:    "video-list",
				Aliases: []string{"f"},
				Usage:   "指定视频列表文件路径",
			},
			&cli.StringFlag{
				Name:    "window",
				Aliases: []string{"w"},
				Usage:   "时段分组：day | week | month | quarter",
			},
			&cli.StringFlag{
				Name:  "from",
				Usage: "时段起始日期 (YYYY-MM-DD)",
			},
			&cli.StringFlag{
				Name:  "to",
				Usage: "时段结束日期 (YYYY-MM-DD)",
			},
			&cli.IntFlag{
				Name:  "max-videos",
				Usage: "最大分析视频数",
				Value: 10,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("🔬 Starting analysis...")
			if !mirrorconfig.HasValidLLM() {
				fmt.Println("  ⚠  LLM not configured. Set MIRROR_LLM_API_KEY.")
				return nil
			}
			fmt.Printf("  ✓  LLM configured: %s (%s)\n",
				mirrorconfig.Conf.LLM.Provider, mirrorconfig.Conf.LLM.Model)

			// 优先使用 video-list
			if vl := cmd.String("video-list"); vl != "" {
				if err := pipeline.RunAnalysisWithVideoList(ctx, "video-list", vl); err != nil {
					return fmt.Errorf("video-list analysis failed: %w", err)
				}
				fmt.Println("\n✅ Analysis complete.")
				return nil
			}

			communities := cmd.String("communities")
			if communities == "" {
				communities = "原神,星穹铁道,绝区零,鸣潮,明日方舟"
			}
			if err := pipeline.RunAnalysis(ctx, "batch-analysis",
				strings.Split(communities, ",")); err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}
			fmt.Println("\n✅ Analysis complete.")
			return nil
		},
	}
}

func reportCommand() *cli.Command {
	return &cli.Command{
		Name:    "report",
		Aliases: []string{"r"},
		Usage:   "聚合分析结果，生成社区关系矩阵和群体行为画像",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "dir",
				Aliases:  []string{"d"},
				Usage:    "分析结果目录",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "输出格式: json | html",
				Value:   "json",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "HTML 输出路径",
				Value:   "report.html",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			taskDir := cmd.String("dir")
			format := cmd.String("format")

			switch format {
			case "html":
				out := cmd.String("output")
				fmt.Printf("📊 Generating HTML report from %s...\n", taskDir)
				if err := report.GenerateHTML(taskDir, out); err != nil {
					return fmt.Errorf("html report: %w", err)
				}
				fmt.Printf("✅ Report → %s\n", out)

			default:
				fmt.Printf("📊 Aggregating analysis results from %s...\n", taskDir)
				_, err := analysis.RunAggregation(taskDir)
				if err != nil {
					return fmt.Errorf("aggregation: %w", err)
				}
				var summary analysis.Summary
				data, _ := os.ReadFile(taskDir + "/summary.json")
				json.Unmarshal(data, &summary)
				b, _ := json.MarshalIndent(summary, "", "  ")
				fmt.Println(string(b))
				fmt.Printf("\n✅ %s/matrix.json + profiles/ + summary.json\n", taskDir)
			}
			return nil
		},
	}
}

func trendCommand() *cli.Command {
	return &cli.Command{
		Name:    "trend",
		Aliases: []string{"t"},
		Usage:   "跨时段对比多个分析结果，生成趋势图表",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:     "dir",
				Aliases:  []string{"d"},
				Usage:    "分析结果目录，可指定多次（按时间顺序）",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "输出 HTML 路径",
				Value:   "trend.html",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dirs := cmd.StringSlice("dir")
			out := cmd.String("output")
			fmt.Printf("📈 Generating trend report from %d directories...\n", len(dirs))
			for i, d := range dirs {
				fmt.Printf("  %d. %s\n", i+1, d)
			}
			if err := report.GenerateTrendHTML(dirs, out); err != nil {
				return fmt.Errorf("trend: %w", err)
			}
			fmt.Printf("✅ Trend report → %s\n", out)
			return nil
		},
	}
}
