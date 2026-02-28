package main

import (
	"fmt"

	z3 "github.com/Z3Prover/z3/src/api/go"
)

func runRabbitsAndPheasants() error {
	ctx := z3.NewContext()
	solver := ctx.NewSolver()

	rabbits := ctx.MkIntConst("rabbits")
	pheasants := ctx.MkIntConst("pheasants")

	solver.Assert(ctx.MkEq(ctx.MkAdd(rabbits, pheasants), ctx.MkInt(9, ctx.MkIntSort())))
	solver.Assert(ctx.MkEq(ctx.MkAdd(ctx.MkMul(ctx.MkInt(4, ctx.MkIntSort()), rabbits), ctx.MkMul(ctx.MkInt(2, ctx.MkIntSort()), pheasants)), ctx.MkInt(24, ctx.MkIntSort())))
	solver.Assert(ctx.MkGe(rabbits, ctx.MkInt(0, ctx.MkIntSort())))
	solver.Assert(ctx.MkGe(pheasants, ctx.MkInt(0, ctx.MkIntSort())))

	if status := solver.Check(); status != z3.Satisfiable {
		return fmt.Errorf("%s returned %s", "rabbits-and-pheasants", status.String())
	}

	model := solver.Model()
	rVal, rOk := model.Eval(rabbits, true)
	if !rOk {
		return fmt.Errorf("failed to evaluate %s", rabbits.String())
	}
	r, err := parseIntExpr(rVal.String())
	if err != nil {
		return err
	}
	pVal, pOk := model.Eval(pheasants, true)
	if !pOk {
		return fmt.Errorf("failed to evaluate %s", pheasants.String())
	}
	p, err := parseIntExpr(pVal.String())
	if err != nil {
		return err
	}

	fmt.Printf("Rabbits: %d, Pheasants: %d\n", r, p)
	if r != 3 || p != 6 {
		return fmt.Errorf("unexpected solution: rabbits=%d pheasants=%d", r, p)
	}
	return nil
}
