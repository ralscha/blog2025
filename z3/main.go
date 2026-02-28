package main

import (
	"fmt"
	"os"
)

func main() {
	examples := []example{
		{name: "RabbitsAndPheasants", run: runRabbitsAndPheasants},
		{name: "RabbitsAndPheasantsWithOr", run: runRabbitsAndPheasantsWithOr},
		{name: "XKCD287", run: runXKCD287},
		{name: "EinsteinRiddle", run: runEinsteinRiddle},
		{name: "SkisAssignment", run: runSkisAssignment},
		{name: "OrganizeYourDay", run: runOrganizeYourDay},
		{name: "Sudoku", run: runSudoku},
		{name: "NQueens", run: runNQueens},
		{name: "MagicSquare", run: runMagicSquare},
		{name: "GraphColoring", run: runGraphColoring},
		{name: "Knapsack", run: runKnapsack},
		{name: "SendMoreMoney", run: runSendMoreMoney},
		{name: "BuildingAHouse", run: runBuildingAHouse},
		{name: "RBAC", run: runRBAC},
		{name: "PackageResolver", run: runPackageResolver},
	}

	for _, ex := range examples {
		fmt.Printf("\n--- %s ---\n", ex.name)
		if err := ex.run(); err != nil {
			fmt.Printf("FAILED: %v\n", err)
			os.Exit(1)
		}
	}
}
