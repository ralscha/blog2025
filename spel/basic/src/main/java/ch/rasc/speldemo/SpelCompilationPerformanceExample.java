package ch.rasc.speldemo;

import java.util.List;

import org.springframework.expression.Expression;
import org.springframework.expression.spel.SpelCompilerMode;
import org.springframework.expression.spel.SpelParserConfiguration;
import org.springframework.expression.spel.standard.SpelExpressionParser;
import org.springframework.expression.spel.support.SimpleEvaluationContext;

public class SpelCompilationPerformanceExample {

    private static final String EXPRESSION = "orders[0].amount * fxRate + baseFee";
    private static final int WARMUP_ITERATIONS = 100_000;
    private static final int MEASURE_ITERATIONS = 2_000_000;

    public record Order(double amount) {
    }

    public record PricingInput(List<Order> orders, double fxRate, double baseFee) {
    }

    public static void main(String[] args) {
        PricingInput input = new PricingInput(List.of(new Order(420.75)), 1.07, 2.50);

        RunResult interpreted = benchmark("Interpreted (OFF)", SpelCompilerMode.OFF, input);
        RunResult compiled = benchmark("Compiled (IMMEDIATE)", SpelCompilerMode.IMMEDIATE, input);

        double interpretedMs = interpreted.durationNanos() / 1_000_000.0;
        double compiledMs = compiled.durationNanos() / 1_000_000.0;
        double speedup = interpretedMs / compiledMs;

        System.out.printf("Expression: %s%n", EXPRESSION);
        System.out.printf("Iterations: %,d%n", MEASURE_ITERATIONS);
        System.out.printf("%s: %.2f ms (checksum=%.2f)%n", interpreted.label(), interpretedMs,
                interpreted.checksum());
        System.out.printf("%s: %.2f ms (checksum=%.2f)%n", compiled.label(), compiledMs,
                compiled.checksum());
        System.out.printf("Speedup: %.2fx%n", speedup);
    }

    private static RunResult benchmark(String label, SpelCompilerMode mode,
            PricingInput input) {
        SpelParserConfiguration config = new SpelParserConfiguration(mode,
                SpelCompilationPerformanceExample.class.getClassLoader());
        SpelExpressionParser parser = new SpelExpressionParser(config);
        Expression expression = parser.parseExpression(EXPRESSION);
        SimpleEvaluationContext context = SimpleEvaluationContext.forReadOnlyDataBinding().withRootObject(input).build();

        evaluateLoop(expression, context, WARMUP_ITERATIONS);

        long start = System.nanoTime();
        double checksum = evaluateLoop(expression, context, MEASURE_ITERATIONS);
        long duration = System.nanoTime() - start;

        return new RunResult(label, duration, checksum);
    }

    private static double evaluateLoop(Expression expression,
    		SimpleEvaluationContext context, int iterations) {
        double checksum = 0.0;
        for (int i = 0; i < iterations; i++) {
            checksum += expression.getValue(context, Double.class);
        }
        return checksum;
    }

    public record RunResult(String label, long durationNanos, double checksum) {
    }
}
