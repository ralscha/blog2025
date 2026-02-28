package main

import (
	"fmt"
	"strings"

	z3 "github.com/Z3Prover/z3/src/api/go"
)

func runMagicSquare() error {
	ctx := z3.NewContext()
	solver := ctx.NewSolver()

	n := 3
	magicSum := 15

	cells := make([][]*z3.Expr, n)
	all := make([]*z3.Expr, 0, n*n)
	for i := range n {
		cells[i] = make([]*z3.Expr, n)
		for j := range n {
			cells[i][j] = ctx.MkIntConst(fmt.Sprintf("m_%d_%d", i, j))
			solver.Assert(ctx.MkGe(cells[i][j], ctx.MkInt(1, ctx.MkIntSort())))
			solver.Assert(ctx.MkLe(cells[i][j], ctx.MkInt(n*n, ctx.MkIntSort())))
			all = append(all, cells[i][j])
		}
	}
	solver.Assert(ctx.MkDistinct(all...))

	for i := range n {
		solver.Assert(ctx.MkEq(ctx.MkAdd(cells[i]...), ctx.MkInt(magicSum, ctx.MkIntSort())))
	}
	for j := range n {
		col := make([]*z3.Expr, n)
		for i := range n {
			col[i] = cells[i][j]
		}
		solver.Assert(ctx.MkEq(ctx.MkAdd(col...), ctx.MkInt(magicSum, ctx.MkIntSort())))
	}

	d1 := make([]*z3.Expr, n)
	d2 := make([]*z3.Expr, n)
	for i := range n {
		d1[i] = cells[i][i]
		d2[i] = cells[i][n-1-i]
	}
	solver.Assert(ctx.MkEq(ctx.MkAdd(d1...), ctx.MkInt(magicSum, ctx.MkIntSort())))
	solver.Assert(ctx.MkEq(ctx.MkAdd(d2...), ctx.MkInt(magicSum, ctx.MkIntSort())))

	if status := solver.Check(); status != z3.Satisfiable {
		return fmt.Errorf("%s returned %s", "magic-square", status.String())
	}

	model := solver.Model()
	fmt.Println("Magic square solution:")
	for i := range n {
		line := make([]string, n)
		for j := range n {
			vVal, vOk := model.Eval(cells[i][j], true)
			if !vOk {
				return fmt.Errorf("failed to evaluate %s", cells[i][j].String())
			}
			v, err := parseIntExpr(vVal.String())
			if err != nil {
				return err
			}
			line[j] = fmt.Sprintf("%d", v)
		}
		fmt.Println(strings.Join(line, " "))
	}
	return nil
}
