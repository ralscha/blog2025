package ch.ralscha.classfileapi;

import java.lang.classfile.ClassFile;
import java.lang.classfile.ClassModel;
import java.lang.classfile.CodeElement;
import java.lang.classfile.CodeModel;
import java.lang.classfile.MethodElement;
import java.lang.classfile.MethodModel;
import java.nio.file.Path;

public class ParseHelloWorldClass {

	public static void main(String[] args) throws Exception {
		Path classFile = Path.of("target", "generated-classes", "Hello.class");
		ClassModel classModel = ClassFile.of().parse(classFile);

		System.out.println("Class: " + classModel.thisClass().asSymbol().displayName());
		System.out.println("Methods:");

		for (MethodModel method : classModel.methods()) {
			System.out.println(
					"- " + method.methodName().stringValue() + " " + method.methodTypeSymbol().displayDescriptor());

			for (MethodElement methodElement : method) {
				if (methodElement instanceof CodeModel codeModel) {
					System.out.println("  Code:");
					for (CodeElement codeElement : codeModel) {
						System.out.println("  - " + codeElement);
					}
				}
			}
		}
	}

}