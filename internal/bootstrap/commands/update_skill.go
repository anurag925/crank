package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/bootstrap"
	"github.com/anurag925/crank/internal/utils"
)

const skillTemplatePath = "templates/agents_skills_crank_project_SKILL.md.tmpl"

const skillOutputRelPath = ".agents/skills/crank-project/SKILL.md"

// NewUpdateSkillCmd returns the `update skill` cobra command.
func NewUpdateSkillCmd(reg *bootstrap.Registry) *cobra.Command {
	var projectDir string

	cmd := &cobra.Command{
		Use:   "update-skill",
		Short: "Regenerate the crank-project agent SKILL.md in the current project",
		Long: `update-skill re-renders the crank-project agent skill file
(.agents/skills/crank-project/SKILL.md) using the latest template bundled with
the crank CLI. This keeps AI coding assistants in sync with the project's
current architecture conventions.

The file is regenerated from the base feature's template, so it always reflects
the latest crank conventions regardless of the crank version the project was
originally generated with.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateSkill(reg, projectDir)
		},
	}

	cmd.Flags().StringVar(&projectDir, "project", ".", "path to the project directory")
	return cmd
}

func updateSkill(reg *bootstrap.Registry, projectDir string) error {
	info, err := bootstrap.LoadProjectInfo(projectDir)
	if err != nil {
		return fmt.Errorf("load project info: %w", err)
	}

	base, ok := reg.Get("base")
	if !ok {
		return fmt.Errorf("base feature not found in registry")
	}

	tmplData, err := fs.ReadFile(base.Templates(), skillTemplatePath)
	if err != nil {
		return fmt.Errorf("read skill template: %w", err)
	}

	tmpl, err := template.New("SKILL.md").Option("missingkey=error").Parse(string(tmplData))
	if err != nil {
		return fmt.Errorf("parse skill template: %w", err)
	}

	ctx := bootstrap.NewContext(info.ProjectName, info.ModulePath, info.Features)

	var sb strings.Builder
	if err := tmpl.Execute(&sb, ctx); err != nil {
		return fmt.Errorf("execute skill template: %w", err)
	}

	dest := filepath.Join(projectDir, skillOutputRelPath)
	if err := utils.EnsureDir(filepath.Dir(dest)); err != nil {
		return fmt.Errorf("create skill dir: %w", err)
	}
	if err := os.WriteFile(dest, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("write skill file: %w", err)
	}

	fmt.Printf("✔ Updated %s\n", skillOutputRelPath)
	return nil
}
