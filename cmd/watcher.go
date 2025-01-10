package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vansante/go-filewatcher/watcher"
)

func init() {
	Cmd.Flags().StringVar(&initCmd,
		"init-cmd", "", "The command to execute on initial start (pre-compilation)",
	)
	Cmd.Flags().StringVar(&prepCmd,
		"prep-cmd", "", "The command to execute on changes (compilation)",
	)
	Cmd.Flags().StringVar(&runCmd,
		"run-cmd", "", "The command to execute to run the program (run)",
	)
	extensions = Cmd.Flags().StringSlice("file-extensions", []string{
		".go",
		".mod",
	}, "The file extensions to watch (comma separated)")
}

var (
	initCmd string
	prepCmd string
	runCmd  string

	extensions *[]string
)

var Cmd = &cobra.Command{
	Use:   os.Args[0],
	Short: "File change watcher",
	Long:  "Watch files in a directory and execute command on changes",
	Example: os.Args[0] + ` --init-cmd="echo Start" --prep-cmd="echo Updated" --run-cmd="echo Running"` +
		` --extensions=".foo,.baz" path/to/dir/to/watch1 dir/to/watch2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if runCmd == "" {
			return fmt.Errorf("--run-cmd is required")
		}

		w, err := watcher.New(cmd.Context(), prepCmd, runCmd)
		if err != nil {
			return err
		}

		if extensions != nil && len(*extensions) > 0 {
			w.SetExtensions(*extensions)
		}

		for _, path := range args {
			err := w.AddPath(path)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
		}
		if len(args) == 0 {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			err = w.AddPath(wd)
			if err != nil {
				return err
			}
		}

		if initCmd != "" {
			_, err := watcher.RunCommand(cmd.Context(), initCmd, true)
			if err != nil {
				return fmt.Errorf("error running init command '%s': %w", initCmd, err)
			}
		}
		if prepCmd != "" {
			_, err := watcher.RunCommand(cmd.Context(), prepCmd, true)
			if err != nil {
				return fmt.Errorf("error running prep command '%s': %w", prepCmd, err)
			}
		}

		go w.Watch()

		<-cmd.Context().Done()
		return nil
	},
}
