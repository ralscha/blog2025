package main

import (
	"fmt"

	z3 "github.com/Z3Prover/z3/src/api/go"
)

func runRabbitsAndPheasantsWithOr() error {
	ctx := z3.NewContext()
	solver := ctx.NewSolver()

	rabbits := ctx.MkIntConst("rabbits")
	pheasants := ctx.MkIntConst("pheasants")
	totalLegs := ctx.MkAdd(ctx.MkMul(ctx.MkInt(4, ctx.MkIntSort()), rabbits), ctx.MkMul(ctx.MkInt(2, ctx.MkIntSort()), pheasants))

	solver.Assert(ctx.MkEq(ctx.MkAdd(rabbits, pheasants), ctx.MkInt(9, ctx.MkIntSort())))
	solver.Assert(ctx.MkOr(ctx.MkEq(totalLegs, ctx.MkInt(24, ctx.MkIntSort())), ctx.MkEq(totalLegs, ctx.MkInt(27, ctx.MkIntSort()))))
	solver.Assert(ctx.MkGe(rabbits, ctx.MkInt(0, ctx.MkIntSort())))
	solver.Assert(ctx.MkGe(pheasants, ctx.MkInt(0, ctx.MkIntSort())))

	if status := solver.Check(); status != z3.Satisfiable {
		return fmt.Errorf("%s returned %s", "rabbits-and-pheasants-with-or", status.String())
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

	legs := 4*r + 2*p
	fmt.Printf("Rabbits: %d, Pheasants: %d, Legs: %d\n", r, p, legs)
	if r+p != 9 || (legs != 24 && legs != 27) {
		return fmt.Errorf("invalid model: rabbits=%d pheasants=%d legs=%d", r, p, legs)
	}
	return nil
}
