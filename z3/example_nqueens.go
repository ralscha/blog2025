package main

import (
	"fmt"
	"strings"

	z3 "github.com/Z3Prover/z3/src/api/go"
)

func runNQueens() error {
	ctx := z3.NewContext()
	solver := ctx.NewSolver()

	const n = 8
	queens := make([]*z3.Expr, n)
	for i := range n {
		queens[i] = ctx.MkIntConst(fmt.Sprintf("queen_%d", i))
		solver.Assert(ctx.MkGe(queens[i], ctx.MkInt(0, ctx.MkIntSort())))
		solver.Assert(ctx.MkLt(queens[i], ctx.MkInt(n, ctx.MkIntSort())))
	}

	solver.Assert(ctx.MkDistinct(queens...))

	for i := range n {
		for j := i + 1; j < n; j++ {
			diff := j - i
			colDiff := ctx.MkSub(queens[j], queens[i])
			solver.Assert(ctx.MkNot(ctx.MkEq(colDiff, ctx.MkInt(diff, ctx.MkIntSort()))))
			solver.Assert(ctx.MkNot(ctx.MkEq(colDiff, ctx.MkInt(-diff, ctx.MkIntSort()))))
		}
	}

	if status := solver.Check(); status != z3.Satisfiable {
		return fmt.Errorf("%s returned %s", "nqueens", status.String())
	}

	model := solver.Model()
	fmt.Printf("%d-Queens solution:\n", n)
	for i := range n {
		colVal, colOk := model.Eval(queens[i], true)
		if !colOk {
			return fmt.Errorf("failed to evaluate %s", queens[i].String())
		}
		col, err := parseIntExpr(colVal.String())
		if err != nil {
			return err
		}
		row := make([]string, n)
		for j := range n {
			if int64(j) == col {
				row[j] = "Q"
			} else {
				row[j] = "."
			}
		}
		fmt.Println(strings.Join(row, " "))
	}
	return nil
}
