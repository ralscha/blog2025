package ch.rasc.speldemo;

import java.lang.reflect.Method;
import java.util.List;
import java.util.Map;

import org.springframework.expression.ExpressionParser;
import org.springframework.expression.common.TemplateParserContext;
import org.springframework.expression.spel.standard.SpelExpressionParser;
import org.springframework.expression.spel.support.StandardEvaluationContext;

public class SpelExamples {

    public static class MutableCustomer {
        private String preferredContact;

        public MutableCustomer(String preferredContact) {
            this.preferredContact = preferredContact;
        }

        public String getPreferredContact() {
            return this.preferredContact;
        }

        public void setPreferredContact(String preferredContact) {
            this.preferredContact = preferredContact;
        }
    }

    public record Customer(String name, int age, String preferredContact, List<String> tags,
            Map<String, String> attributes) {
    }

    public static void main(String[] args) throws Exception {
        ExpressionParser parser = new SpelExpressionParser();

        Integer result = parser.parseExpression("20 + 22").getValue(Integer.class);
        assert result == 42 : "Expected 42 but got " + result;  
        
        Customer customer = new Customer("Ada", 34, null, List.of("vip", "beta"),
            Map.of("country", "CH", "currency", "CHF"));        
        var ctx = new StandardEvaluationContext();
        String name = parser.parseExpression("name").getValue(ctx, customer, String.class);
        assert "Ada".equals(name) : "Expected 'Ada' but got '" + name + "'";

        StandardEvaluationContext context = new StandardEvaluationContext(customer);
        String upperName = parser.parseExpression("name.toUpperCase()").getValue(context, String.class);
        assert "ADA".equals(upperName) : "Expected 'ADA' but got '" + upperName + "'";

        context.setVariable("orderAmount", 420.75);
        Boolean booleanExpression = parser.parseExpression("age >= 18 and #orderAmount > 100").getValue(context, Boolean.class);
        assert booleanExpression : "Expected true but got " + booleanExpression;        

        context.setVariable("riskScore", 71);      
        String ternary = parser.parseExpression("#riskScore > 70 ? 'REVIEW' : 'AUTO'").getValue(context, String.class);
        assert "REVIEW".equals(ternary) : "Expected 'REVIEW' but got '" + ternary + "'";

        String safeNavigation = parser.parseExpression("preferredContact?.toUpperCase() ?: 'EMAIL'").getValue(context, String.class);
        assert "EMAIL".equals(safeNavigation) : "Expected 'EMAIL' but got '" + safeNavigation + "'";

        MutableCustomer mutableCustomer = new MutableCustomer(null);
        StandardEvaluationContext mutableContext = new StandardEvaluationContext(mutableCustomer);
        parser.parseExpression("preferredContact").setValue(mutableContext, "SMS");
        assert "SMS".equals(mutableCustomer.getPreferredContact())
            : "Expected 'SMS' but got '" + mutableCustomer.getPreferredContact() + "'";

        String listIndex = parser.parseExpression("tags[0]").getValue(context, String.class);
        assert "vip".equals(listIndex) : "Expected 'vip' but got '" + listIndex + "'";

        String mapAccess = parser.parseExpression("attributes['country']").getValue(context, String.class);
        assert "CH".equals(mapAccess) : "Expected 'CH' but got '" + mapAccess + "'";

        List<String> selection = parser.parseExpression("tags.?[#this.startsWith('v')]").getValue(context, List.class);
        assert "vip".equals(selection.getFirst()) : "Expected 'vip' but got '" + selection + "'";

        List<String> projection = parser.parseExpression("tags.![#this.toUpperCase()]").getValue(context, List.class);
        assert "VIP".equals(projection.getFirst()) && "BETA".equals(projection.getLast()) : "Expected 'VIP,BETA' but got '" + projection + "'";
        
        List<String> projectedKeys = parser.parseExpression("attributes.![key]").getValue(context, List.class);
        assert projectedKeys.size() == 2 && projectedKeys.containsAll(List.of("country", "currency")) : "Expected 'country,currency' but got '" + projectedKeys + "'";

        List<String> projectedValues = parser.parseExpression("attributes.![value]").getValue(context, List.class);
        assert projectedValues.size() == 2 && projectedValues.containsAll(List.of("CH", "CHF")) : "Expected 'CH,CHF' but got '" + projectedValues + "'";

        String projectionString = parser.parseExpression("tags.![#this.toUpperCase()]").getValue(context, String.class);
        assert "VIP,BETA".equals(projectionString) : "Expected 'VIP,BETA' but got '" + projection + "'";

        Integer typeReference = parser.parseExpression("T(java.lang.Math).max(10, age)").getValue(context, Integer.class);
        assert typeReference == 34 : "Expected 34 but got " + typeReference;

        Method slugMethod = SpelExamples.class.getDeclaredMethod("slugify", String.class);
        context.registerFunction("slug", slugMethod);                
        String slug = parser.parseExpression("#slug(name)").getValue(context, String.class);
        assert "ada".equals(slug) : "Expected 'ada' but got '" + slug + "'";

        String message = parser
                .parseExpression("Customer #{name} in #{attributes['country']} -> #{#slug(name)}",
                        new TemplateParserContext())
                .getValue(context, String.class);
        assert "Customer Ada in CH -> ada".equals(message) : "Expected 'Customer Ada in CH -> ada' but got '" + message + "'";
    }

    public static String slugify(String value) {
        return value == null ? "" : value.trim().toLowerCase().replace(" ", "-");
    }


}
