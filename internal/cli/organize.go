package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dreikanter/notesctl/note"
	"github.com/spf13/cobra"
)

type organizeMove struct {
	srcRel  string
	dstRel  string
	reason  string
}

type organizeConflict struct {
	dstRel string
	srcs   []string
	reason string
}

var organizeCmd = &cobra.Command{
	Use:   "organize",
	Short: "Organize notes by year and tags from frontmatter",
	Long: `Scan all .md files in the notes directory, read their frontmatter
for tags and date fields, and plan moves to organize them into subdirectories.

By default, this command runs in dry-run mode and only prints the planned moves.
Use --apply to actually perform the moves.

Notes without frontmatter, or with missing date/tags fields, are moved to
the 'uncategorized' directory.`,
	Args: cobra.NoArgs,
	RunE: organizeRunE,
}

func organizeRunE(cmd *cobra.Command, _ []string) error {
	apply, _ := cmd.Flags().GetBool("apply")

	root, err := notesRoot()
	if err != nil {
		return err
	}

	moves, conflicts, err := planOrganize(root)
	if err != nil {
		return err
	}

	if len(conflicts) > 0 {
		printConflicts(cmd.OutOrStdout(), conflicts)
		return fmt.Errorf("found %d conflict(s), aborting", len(conflicts))
	}

	if len(moves) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No files to organize.")
		return nil
	}

	printPlan(cmd.OutOrStdout(), moves)

	if !apply {
		fmt.Fprintln(cmd.OutOrStdout(), "\nDry-run complete. Use --apply to perform moves.")
		return nil
	}

	return executeMoves(root, moves, cmd.OutOrStdout())
}

func planOrganize(root string) ([]organizeMove, []organizeConflict, error) {
	var moves []organizeMove

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".md" {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		move, err := planSingleFile(root, relPath)
		if err != nil {
			return fmt.Errorf("%s: %w", relPath, err)
		}

		if move != nil && move.srcRel != move.dstRel {
			moves = append(moves, *move)
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	sort.Slice(moves, func(i, j int) bool {
		return moves[i].srcRel < moves[j].srcRel
	})

	conflicts := detectConflicts(root, moves)

	return moves, conflicts, nil
}

func planSingleFile(root, relPath string) (*organizeMove, error) {
	absPath := filepath.Join(root, relPath)

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	fm, _, err := note.ParseNote(data)
	if err != nil {
		return nil, err
	}

	dstRel, reason := computeDestination(relPath, fm.Date, fm.Tags)

	return &organizeMove{
		srcRel: relPath,
		dstRel: dstRel,
		reason: reason,
	}, nil
}

func detectConflicts(root string, moves []organizeMove) []organizeConflict {
	var conflicts []organizeConflict

	dstToSrcs := make(map[string][]string)
	for _, m := range moves {
		dstToSrcs[m.dstRel] = append(dstToSrcs[m.dstRel], m.srcRel)
	}

	for dstRel, srcs := range dstToSrcs {
		if len(srcs) > 1 {
			conflicts = append(conflicts, organizeConflict{
				dstRel: dstRel,
				srcs:   srcs,
				reason: "multiple sources map to same destination",
			})
			continue
		}

		dstAbs := filepath.Join(root, dstRel)
		if _, err := os.Stat(dstAbs); err == nil {
			srcRel := srcs[0]
			srcAbs := filepath.Join(root, srcRel)

			srcInfo, srcErr := os.Stat(srcAbs)
			dstInfo, dstErr := os.Stat(dstAbs)
			if srcErr == nil && dstErr == nil && os.SameFile(srcInfo, dstInfo) {
				continue
			}

			conflicts = append(conflicts, organizeConflict{
				dstRel: dstRel,
				srcs:   srcs,
				reason: "destination already exists",
			})
		}
	}

	sort.Slice(conflicts, func(i, j int) bool {
		return conflicts[i].dstRel < conflicts[j].dstRel
	})

	return conflicts
}

func computeDestination(srcRel string, date time.Time, tags []string) (string, string) {
	hasDate := !date.IsZero()
	hasTags := len(tags) > 0

	if !hasDate || !hasTags {
		base := filepath.Base(srcRel)
		return filepath.Join("uncategorized", base), "uncategorized (missing date or tags)"
	}

	year := date.Format("2006")
	primaryTag := tags[0]

	for _, t := range tags {
		if t != "" {
			primaryTag = t
			break
		}
	}

	if primaryTag == "" {
		base := filepath.Base(srcRel)
		return filepath.Join("uncategorized", base), "uncategorized (no valid tags)"
	}

	base := filepath.Base(srcRel)
	return filepath.Join(year, primaryTag, base), fmt.Sprintf("year=%s, tag=%s", year, primaryTag)
}

func printPlan(out io.Writer, moves []organizeMove) {
	fmt.Fprintf(out, "Planned moves (%d):\n\n", len(moves))

	maxSrcLen := 0
	for _, m := range moves {
		if len(m.srcRel) > maxSrcLen {
			maxSrcLen = len(m.srcRel)
		}
	}

	for _, m := range moves {
		padding := strings.Repeat(" ", maxSrcLen-len(m.srcRel))
		fmt.Fprintf(out, "  %s%s -> %s (%s)\n", m.srcRel, padding, m.dstRel, m.reason)
	}
}

func printConflicts(out io.Writer, conflicts []organizeConflict) {
	fmt.Fprintf(out, "Conflicts detected (%d):\n\n", len(conflicts))

	for i, c := range conflicts {
		fmt.Fprintf(out, "  Conflict %d: %s\n", i+1, c.dstRel)
		fmt.Fprintf(out, "    Reason: %s\n", c.reason)
		fmt.Fprintf(out, "    Sources:\n")
		for _, src := range c.srcs {
			fmt.Fprintf(out, "      - %s\n", src)
		}
		fmt.Fprintln(out)
	}
}

func executeMoves(root string, moves []organizeMove, out io.Writer) error {
	fmt.Fprintln(out, "\nExecuting moves...")

	for _, m := range moves {
		srcAbs := filepath.Join(root, m.srcRel)
		dstAbs := filepath.Join(root, m.dstRel)

		dstDir := filepath.Dir(dstAbs)
		if err := os.MkdirAll(dstDir, note.StoreDirMode(root)); err != nil {
			return fmt.Errorf("cannot create directory %s: %w", dstDir, err)
		}

		if _, err := os.Stat(dstAbs); err == nil {
			return fmt.Errorf("destination already exists: %s", m.dstRel)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("check destination %s: %w", m.dstRel, err)
		}

		if err := os.Rename(srcAbs, dstAbs); err != nil {
			return fmt.Errorf("cannot move %s to %s: %w", m.srcRel, m.dstRel, err)
		}

		fmt.Fprintf(out, "  Moved: %s -> %s\n", m.srcRel, m.dstRel)

		srcDir := filepath.Dir(srcAbs)
		if srcDir != root {
			isEmpty, err := isDirEmpty(srcDir)
			if err == nil && isEmpty {
				if err := os.Remove(srcDir); err == nil {
					fmt.Fprintf(out, "  Removed empty directory: %s\n", filepath.Dir(m.srcRel))
				}
			}
		}
	}

	fmt.Fprintln(out, "\nDone.")
	return nil
}

func isDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	names, err := f.Readdirnames(1)
	if err != nil && err.Error() != "EOF" {
		return false, err
	}

	return len(names) == 0, nil
}

func registerOrganizeFlags() {
	organizeCmd.Flags().Bool("apply", false, "actually perform the moves (default: dry-run only)")
}

func init() {
	registerOrganizeFlags()
	rootCmd.AddCommand(organizeCmd)
}
