package ch.rasc.speldemo.fees;

import java.util.ArrayList;
import java.util.List;

import org.springframework.expression.Expression;
import org.springframework.expression.ExpressionParser;
import org.springframework.expression.spel.standard.SpelExpressionParser;
import org.springframework.expression.spel.support.SimpleEvaluationContext;
import org.springframework.stereotype.Service;

@Service
public class SpelFeeEngine {

	private static final double MINIMUM_FEE = 0.35;

	private final List<Rule> rules;

	public SpelFeeEngine(ExternalFeeRuleLoader ruleLoader) {
		ExpressionParser parser = new SpelExpressionParser();
		this.rules = ruleLoader.load().stream()
				.map(rule -> new Rule(rule,
						parser.parseExpression(rule.conditionExpression()),
						parser.parseExpression(rule.feeExpression())))
				.toList();
	}

	public FeeCalculationResult calculate(FeeRequest request) {
		SimpleEvaluationContext context = SimpleEvaluationContext
				.forReadOnlyDataBinding().build();

		List<FeeLineItem> lineItems = new ArrayList<>();
		double total = 0.0;
		context.setVariable("runningTotal", total);

		for (Rule rule : this.rules) {
			Boolean matches = rule.condition().getValue(context, request, Boolean.class);
			if (!Boolean.TRUE.equals(matches)) {
				continue;
			}

			Double feeDelta = rule.fee().getValue(context, request, Double.class);
			double amount = feeDelta != null ? FeeMath.roundMoney(feeDelta) : 0.0;
			total += amount;
			total = FeeMath.roundMoney(total);
			context.setVariable("runningTotal", total);

			lineItems.add(new FeeLineItem(rule.rule().name(),
					rule.rule().description(), amount));
		}

		if (total < MINIMUM_FEE) {
			double adjustment = FeeMath.roundMoney(MINIMUM_FEE - total);
			total = FeeMath.roundMoney(total + adjustment);
			lineItems.add(new FeeLineItem("MINIMUM_FEE", "Minimum fee safeguard",
					adjustment));
		}

		return new FeeCalculationResult(request, List.copyOf(lineItems), total);
	}

	private record Rule(FeeRule rule, Expression condition, Expression fee) {
	}
}
