package main

import (
	"log"
	"sort"
)

func main() {
	data, err := LoadTDLibJSON("https://raw.githubusercontent.com/FallenProjects/tdlib-build/refs/heads/master/tdlib.json")
	if err != nil {
		log.Fatalf("Failed to load JSON: %v", err)
	}

	log.Printf("Loaded TDLib-%s with %d types, %d functions, and %d classes", data.Version, len(data.Types), len(data.Functions), len(data.Classes))

	types, functions, classes, options := ConvertJSONToGen(data)

	sortTLTypesByName(types)
	sortTLTypesByName(functions)
	for _, cls := range classes {
		sort.Strings(cls.Implementations)
	}

	log.Println("Generating Go code...")
	generateInterfaces(classes)
	generateTypes(types, classes)
	generateFunctions(functions, classes)
	generateMethods(functions, classes)
	generateUpdates(types)
	generateOptions(options)
	helperFiles := generateHelpers(types, functions, classes)
	filesToFmt := append([]string{"gen_interfaces.go", "gen_types.go", "gen_functions.go", "gen_methods.go", "gen_handlers.go", "client_opts.go"}, helperFiles...)
	gofmt(filesToFmt...)
	log.Println("Done.")
}
