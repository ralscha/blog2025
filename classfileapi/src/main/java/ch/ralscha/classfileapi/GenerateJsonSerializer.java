package ch.ralscha.classfileapi;

public class GenerateJsonSerializer {

	public static void main(String[] args) {
		JSONSerializer<Person> serializer = JSONSerializer.from(Person.class);
		String json = serializer.serialize(new Person("Duke", "Java"));
		System.out.println(json);
	}

	record Person(String firstName, String lastName) {
	}

}