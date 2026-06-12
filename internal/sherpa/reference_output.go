package sherpa

import "fmt"

func PrintReferences(name string, refs []Reference) {
	if len(refs) == 0 {
		fmt.Println("no references found:", name)
		return
	}

	fmt.Println("🔍 REFERENCES")
	fmt.Println()
	fmt.Println(name)
	fmt.Println()

	for _, ref := range refs {
		fmt.Printf(
			"  %s:%d\n",
			ref.Position.File,
			ref.Position.Line,
		)
	}

	fmt.Println()
	fmt.Printf("Found %d references\n", len(refs))
}