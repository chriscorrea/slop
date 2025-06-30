package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var manCmd = &cobra.Command{
	Use:    "man",
	Short:  "Generate man pages for slop",
	Long:   `This command generates the man pages for the slop CLI.`,
	Hidden: true, // hide this from the public help output
	Run: func(cmd *cobra.Command, args []string) {
		// define the header for the man page.
		header := &doc.GenManHeader{
			Title:   "SLOP",
			Section: "1", // Section 1 is for executable programs and shell commands
			Source:  "Slop CLI",
		}

		// create the directory if it doesn't exist
		err := os.MkdirAll("./man", os.ModePerm)
		if err != nil {
			log.Fatalf("failed to create man directory: %v", err)
		}

		// generate man page for the root command (slop)
		err = doc.GenManTree(rootCmd, header, "./man")
		if err != nil {
			log.Fatalf("failed to generate man pages: %v", err)
		}

		log.Println("Man pages successfully generated in ./man directory")
	},
}

// add the man command to root command's hierarchy
func init() {
	rootCmd.AddCommand(manCmd)
}
