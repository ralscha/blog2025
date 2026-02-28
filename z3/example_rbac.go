package main

import (
	"fmt"

	z3 "github.com/Z3Prover/z3/src/api/go"
)

func runRBAC() error {
	ctx := z3.NewContext()
	strSort := ctx.MkStringSort()

	userCountry := ctx.MkConst(ctx.MkStringSymbol("user_country"), strSort)
	nodeLocation := ctx.MkConst(ctx.MkStringSymbol("node_location"), strSort)
	nodeRunning := ctx.MkConst(ctx.MkStringSymbol("node_running"), strSort)
	fooapp := ctx.MkString("fooapp")

	// Policy 1: node must be in the SAME country as the user and run "fooapp".
	policy := ctx.MkAnd(
		ctx.MkEq(nodeLocation, userCountry),
		ctx.MkEq(nodeRunning, fooapp),
	)

	solver := ctx.NewSolver()

	// User 1: user in Canada, node in Canada running "fooapp" → should allow
	solver.Assert(policy)

	// user role
	solver.Assert(ctx.MkEq(userCountry, ctx.MkString("Canada")))
	solver.Assert(ctx.MkEq(nodeLocation, ctx.MkString("Canada")))
	solver.Assert(ctx.MkEq(nodeRunning, fooapp))

	if solver.Check() == z3.Satisfiable {
		fmt.Println("User 1: Access Allowed")
	} else {
		fmt.Println("User 1: Access Denied")
	}

	// User 2: user in Canada, node in USA running "fooapp" → should deny
	solver.Reset()
	solver.Assert(policy)

	// user role
	solver.Assert(ctx.MkEq(userCountry, ctx.MkString("Canada")))
	solver.Assert(ctx.MkEq(nodeLocation, ctx.MkString("USA")))
	solver.Assert(ctx.MkEq(nodeRunning, fooapp))
	if solver.Check() == z3.Satisfiable {
		fmt.Println("User 2: Access Allowed")
	} else {
		fmt.Println("User 2: Access Denied")
	}

	// User 3: user in Canada, node in Canada running "barapp" → should deny
	solver.Reset()
	solver.Assert(policy)
	solver.Assert(ctx.MkEq(userCountry, ctx.MkString("Canada")))
	solver.Assert(ctx.MkEq(nodeLocation, ctx.MkString("Canada")))
	solver.Assert(ctx.MkEq(nodeRunning, ctx.MkString("barapp")))
	if solver.Check() == z3.Satisfiable {
		fmt.Println("User 3: Access Allowed")
	} else {
		fmt.Println("User 3: Access Denied")
	}

	// Regex-based policy
	// Policy: node location must match /us-east-[a-z]+/ and run "fooapp".
	{
		az := ctx.MkReRange(ctx.MkString("a"), ctx.MkString("z"))
		locationRegex := ctx.MkReConcat(
			ctx.MkToRe(ctx.MkString("us-east-")),
			ctx.MkRePlus(az),
		)
		policy3 := ctx.MkAnd(
			ctx.MkInRe(nodeLocation, locationRegex),
			ctx.MkEq(nodeRunning, fooapp),
		)

		testCases := []struct{ loc, run string }{
			{"us-east-virginia", "fooapp"},
			{"us-west-oregon", "fooapp"},
			{"us-east-123", "fooapp"}, // digits don't match [a-z]+
			{"us-east-a", "fooapp"},
			{"us-east-virginia", "barapp"},
		}
		for _, tc := range testCases {
			solver := ctx.NewSolver()
			solver.Assert(policy3)
			solver.Assert(ctx.MkEq(nodeLocation, ctx.MkString(tc.loc)))
			solver.Assert(ctx.MkEq(nodeRunning, ctx.MkString(tc.run)))
			access := "Denied"
			if solver.Check() == z3.Satisfiable {
				access = "Allowed"
			}
			fmt.Printf("   location=%-20s running=%-8s => %s\n", tc.loc, tc.run, access)
		}
	}

	return nil
}
