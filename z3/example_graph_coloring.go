package main

import (
	"fmt"

	z3 "github.com/Z3Prover/z3/src/api/go"
)

func runGraphColoring() error {
	ctx := z3.NewContext()
	solver := ctx.NewSolver()

	numVertices := 4
	edges := [][2]int{{0, 1}, {1, 2}, {2, 3}, {3, 0}, {0, 2}}
	numColors := 3
	colorNames := []string{"Red", "Green", "Blue"}

	colors := make([]*z3.Expr, numVertices)
	for i := range numVertices {
		colors[i] = ctx.MkIntConst(fmt.Sprintf("color_%d", i))
		solver.Assert(ctx.MkGe(colors[i], ctx.MkInt(0, ctx.MkIntSort())))
		solver.Assert(ctx.MkLe(colors[i], ctx.MkInt(numColors-1, ctx.MkIntSort())))
	}

	for _, edge := range edges {
		solver.Assert(ctx.MkNot(ctx.MkEq(colors[edge[0]], colors[edge[1]])))
	}

	if status := solver.Check(); status != z3.Satisfiable {
		return fmt.Errorf("%s returned %s", "graph-coloring", status.String())
	}

	model := solver.Model()
	fmt.Println("Graph coloring solution:")
	for i := range numVertices {
		cVal, cOk := model.Eval(colors[i], true)
		if !cOk {
			return fmt.Errorf("failed to evaluate %s", colors[i].String())
		}
		c, err := parseIntExpr(cVal.String())
		if err != nil {
			return err
		}
		fmt.Printf("  Vertex %d: %s\n", i, colorNames[c])
	}
	return nil
}
