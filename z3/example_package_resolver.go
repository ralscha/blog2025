package main

import (
	"fmt"

	z3 "github.com/Z3Prover/z3/src/api/go"
)

func runPackageResolver() error {
	ctx := z3.NewContext()

	fmt.Println("1. Solvable dependency resolution (prefer newer versions)")
	if err := runPackageResolverSat(ctx); err != nil {
		return err
	}

	fmt.Println("\n2. Conflicting dependency resolution")
	if err := runPackageResolverUnsat(ctx); err != nil {
		return err
	}

	return nil
}

func runPackageResolverSat(ctx *z3.Context) error {
	opt := ctx.NewOptimize()

	app := ctx.MkIntConst("app_1_0")
	core1 := ctx.MkIntConst("core_1_0")
	core2 := ctx.MkIntConst("core_2_0")
	tls1 := ctx.MkIntConst("tls_1_0")
	tls2 := ctx.MkIntConst("tls_2_0")
	legacyPlugin := ctx.MkIntConst("legacy_plugin_1_0")

	all := map[string]*z3.Expr{
		"app@1.0":           app,
		"core@1.0":          core1,
		"core@2.0":          core2,
		"tls@1.0":           tls1,
		"tls@2.0":           tls2,
		"legacy-plugin@1.0": legacyPlugin,
	}
	printOrder := []string{"app@1.0", "core@1.0", "core@2.0", "tls@1.0", "tls@2.0", "legacy-plugin@1.0"}

	for _, v := range all {
		opt.Assert(ctx.MkGe(v, ctx.MkInt(0, ctx.MkIntSort())))
		opt.Assert(ctx.MkLe(v, ctx.MkInt(1, ctx.MkIntSort())))
	}

	opt.Assert(ctx.MkEq(app, ctx.MkInt(1, ctx.MkIntSort())))

	// App requires exactly one core version.
	opt.Assert(ctx.MkEq(ctx.MkAdd(core1, core2), app))

	// Dependency chain.
	opt.Assert(ctx.MkImplies(ctx.MkEq(core1, ctx.MkInt(1, ctx.MkIntSort())), ctx.MkEq(tls1, ctx.MkInt(1, ctx.MkIntSort()))))
	opt.Assert(ctx.MkImplies(ctx.MkEq(core2, ctx.MkInt(1, ctx.MkIntSort())), ctx.MkEq(tls2, ctx.MkInt(1, ctx.MkIntSort()))))

	// Only one TLS version can exist.
	opt.Assert(ctx.MkLe(ctx.MkAdd(tls1, tls2), ctx.MkInt(1, ctx.MkIntSort())))

	// Legacy plugin conflicts with tls@2.0.
	opt.Assert(ctx.MkLe(ctx.MkAdd(legacyPlugin, tls2), ctx.MkInt(1, ctx.MkIntSort())))

	// Prefer modern stack, then fewer packages overall.
	modernScore := ctx.MkAdd(
		ctx.MkMul(core2, ctx.MkInt(100, ctx.MkIntSort())),
		ctx.MkMul(tls2, ctx.MkInt(50, ctx.MkIntSort())),
	)
	installedCount := ctx.MkAdd(app, core1, core2, tls1, tls2, legacyPlugin)

	bestModern := opt.Maximize(modernScore)
	bestSmall := opt.Minimize(installedCount)

	if status := opt.Check(); status != z3.Satisfiable {
		return fmt.Errorf("%s returned %s", "package-resolver-sat", status.String())
	}

	model := opt.Model()
	if upper := opt.GetUpper(bestModern); upper != nil {
		fmt.Printf("   modern-score: %s\n", upper.String())
	}
	if lower := opt.GetLower(bestSmall); lower != nil {
		fmt.Printf("   installed-count: %s\n", lower.String())
	}

	fmt.Println("   selected packages:")
	for _, name := range printOrder {
		expr := all[name]
		val, ok := model.Eval(expr, true)
		if !ok {
			return fmt.Errorf("failed to evaluate %s", name)
		}
		num, err := parseIntExpr(val.String())
		if err != nil {
			return err
		}
		if num == 1 {
			fmt.Printf("   - %s\n", name)
		}
	}

	return nil
}

func runPackageResolverUnsat(ctx *z3.Context) error {
	solver := ctx.NewSolver()

	app := ctx.MkIntConst("bad_app_1_0")
	core2 := ctx.MkIntConst("bad_core_2_0")
	tls2 := ctx.MkIntConst("bad_tls_2_0")

	solver.Assert(ctx.MkGe(app, ctx.MkInt(0, ctx.MkIntSort())))
	solver.Assert(ctx.MkLe(app, ctx.MkInt(1, ctx.MkIntSort())))
	solver.Assert(ctx.MkGe(core2, ctx.MkInt(0, ctx.MkIntSort())))
	solver.Assert(ctx.MkLe(core2, ctx.MkInt(1, ctx.MkIntSort())))
	solver.Assert(ctx.MkGe(tls2, ctx.MkInt(0, ctx.MkIntSort())))
	solver.Assert(ctx.MkLe(tls2, ctx.MkInt(1, ctx.MkIntSort())))

	// Requested installation and manual pinning.
	solver.Assert(ctx.MkEq(app, ctx.MkInt(1, ctx.MkIntSort())))
	solver.Assert(ctx.MkEq(core2, ctx.MkInt(1, ctx.MkIntSort())))

	// A policy forbids tls@2.0, but core@2.0 depends on it.
	solver.Assert(ctx.MkEq(tls2, ctx.MkInt(0, ctx.MkIntSort())))
	solver.Assert(ctx.MkImplies(ctx.MkEq(core2, ctx.MkInt(1, ctx.MkIntSort())), ctx.MkEq(tls2, ctx.MkInt(1, ctx.MkIntSort()))))

	if status := solver.Check(); status != z3.Unsatisfiable {
		return fmt.Errorf("%s expected UNSAT, got %s", "package-resolver-unsat", status.String())
	}

	fmt.Println("   status: UNSAT")
	fmt.Println("   explanation: core@2.0 requires tls@2.0, but tls@2.0 is forbidden by policy")
	return nil
}
