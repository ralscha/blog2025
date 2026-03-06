package ch.rasc.speldemo;

import org.springframework.expression.ExpressionParser;
import org.springframework.expression.spel.standard.SpelExpressionParser;
import org.springframework.expression.spel.support.SimpleEvaluationContext;

public class SimpleEvaluationContextExamples {

    public static class Person {
        private String name;

        public Person(String name) {
            this.name = name;
        }

        public String getName() {
            return this.name;
        }

        public void setName(String name) {
            this.name = name;
        }
    }

    public static void main(String[] args) {
        ExpressionParser parser = new SpelExpressionParser();
        SimpleEvaluationContext readWriteContext = SimpleEvaluationContext.forReadWriteDataBinding().build();

        Person readWritePerson = new Person("Ada");
        String currentName = parser.parseExpression("name").getValue(readWriteContext, readWritePerson, String.class);
        assert "Ada".equals(currentName) : "Expected 'Ada' but got '" + currentName + "'";

        parser.parseExpression("name").setValue(readWriteContext, readWritePerson, "Grace");
        assert "Grace".equals(readWritePerson.getName())
                : "Expected 'Grace' but got '" + readWritePerson.getName() + "'";


        SimpleEvaluationContext readOnlyContext = SimpleEvaluationContext.forReadOnlyDataBinding().build();
        Person readOnlyPerson = new Person("Marie");
        String readOnlyName = parser.parseExpression("name").getValue(readOnlyContext, readOnlyPerson, String.class);
        assert "Marie".equals(readOnlyName) : "Expected 'Marie' but got '" + readOnlyName + "'";

        try {
            parser.parseExpression("name").setValue(readOnlyContext, readOnlyPerson, "Rosalind");
        }
        catch (Exception ex) {
            System.out.println(ex.getMessage());
            // EL1010E: Property or field 'name' cannot be set on object of type 'ch.rasc.speldemo.SimpleEvaluationContextExamples$Person' 
            // - maybe not public or not writable?
        }
    }
}
