package main

import (
	"fmt"
	"strings"

	z3 "github.com/Z3Prover/z3/src/api/go"
)

func runSudoku() error {
	ctx := z3.NewContext()
	solver := ctx.NewSolver()

	cells := make([][]*z3.Expr, 9)
	for i := range 9 {
		cells[i] = make([]*z3.Expr, 9)
		for j := range 9 {
			cells[i][j] = ctx.MkIntConst(fmt.Sprintf("cell_%d_%d", i, j))
			solver.Assert(ctx.MkGe(cells[i][j], ctx.MkInt(1, ctx.MkIntSort())))
			solver.Assert(ctx.MkLe(cells[i][j], ctx.MkInt(9, ctx.MkIntSort())))
		}
	}

	for i := range 9 {
		solver.Assert(ctx.MkDistinct(cells[i]...))
	}

	for j := range 9 {
		col := make([]*z3.Expr, 9)
		for i := range 9 {
			col[i] = cells[i][j]
		}
		solver.Assert(ctx.MkDistinct(col...))
	}

	for boxRow := range 3 {
		for boxCol := range 3 {
			block := make([]*z3.Expr, 0, 9)
			for i := range 3 {
				for j := range 3 {
					block = append(block, cells[boxRow*3+i][boxCol*3+j])
				}
			}
			solver.Assert(ctx.MkDistinct(block...))
		}
	}

	puzzle := [][]int{
		{5, 3, 0, 0, 7, 0, 0, 0, 0},
		{6, 0, 0, 1, 9, 5, 0, 0, 0},
		{0, 9, 8, 0, 0, 0, 0, 6, 0},
		{8, 0, 0, 0, 6, 0, 0, 0, 3},
		{4, 0, 0, 8, 0, 3, 0, 0, 1},
		{7, 0, 0, 0, 2, 0, 0, 0, 6},
		{0, 6, 0, 0, 0, 0, 2, 8, 0},
		{0, 0, 0, 4, 1, 9, 0, 0, 5},
		{0, 0, 0, 0, 8, 0, 0, 7, 9},
	}
	for i := range 9 {
		for j := range 9 {
			if puzzle[i][j] != 0 {
				solver.Assert(ctx.MkEq(cells[i][j], ctx.MkInt(puzzle[i][j], ctx.MkIntSort())))
			}
		}
	}

	if status := solver.Check(); status != z3.Satisfiable {
		return fmt.Errorf("%s returned %s", "sudoku", status.String())
	}

	model := solver.Model()
	fmt.Println("Sudoku solution:")
	for i := range 9 {
		parts := make([]string, 0, 11)
		for j := range 9 {
			vVal, vOk := model.Eval(cells[i][j], true)
			if !vOk {
				return fmt.Errorf("failed to evaluate %s", cells[i][j].String())
			}
			v, err := parseIntExpr(vVal.String())
			if err != nil {
				return err
			}
			parts = append(parts, fmt.Sprintf("%d", v))
			if j == 2 || j == 5 {
				parts = append(parts, "|")
			}
		}
		fmt.Println(strings.Join(parts, " "))
		if i == 2 || i == 5 {
			fmt.Println("------+-------+------")
		}
	}
	return nil
}
