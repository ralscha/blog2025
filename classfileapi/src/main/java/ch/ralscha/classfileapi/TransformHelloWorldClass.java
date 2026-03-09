package ch.ralscha.classfileapi;

import java.lang.classfile.ClassFile;
import java.lang.classfile.ClassModel;
import java.lang.classfile.ClassTransform;
import java.lang.classfile.CodeBuilder;
import java.lang.classfile.CodeTransform;
import java.lang.classfile.MethodModel;
import java.lang.classfile.instruction.ReturnInstruction;
import java.lang.constant.ClassDesc;
import java.lang.constant.ConstantDescs;
import java.lang.constant.MethodTypeDesc;
import java.lang.reflect.Method;
import java.nio.file.Path;

public class TransformHelloWorldClass {

  public static void main(String[] args) throws Exception {
    Path originalClass = Path.of("target", "generated-classes", "Hello.class");
    ClassFile classFile = ClassFile.of();
    ClassModel classModel = classFile.parse(originalClass);

    CodeTransform addLoggingAspect = addLoggingAspect();

    byte[] transformedBytes =
        classFile.transformClass(
            classModel,
            ClassTransform.transformingMethodBodies(
                TransformHelloWorldClass::isMainMethod, addLoggingAspect));

    runGeneratedClass(transformedBytes);
  }

  private static CodeTransform addLoggingAspect() {
    ClassDesc systemClass = ClassDesc.of("java.lang.System");
    ClassDesc printStreamClass = ClassDesc.of("java.io.PrintStream");
    MethodTypeDesc printlnType = MethodTypeDesc.of(ConstantDescs.CD_void, ConstantDescs.CD_String);
    boolean[] entryInjected = {false};

    return (builder, element) -> {
      if (!entryInjected[0]) {
        emitPrintln(builder, systemClass, printStreamClass, printlnType, "Start of main");
        entryInjected[0] = true;
      }

      if (element instanceof ReturnInstruction) {
        emitPrintln(builder, systemClass, printStreamClass, printlnType, "End of main");
      }

      builder.with(element);
    };
  }

  private static void emitPrintln(
      CodeBuilder builder,
      ClassDesc systemClass,
      ClassDesc printStreamClass,
      MethodTypeDesc printlnType,
      String message) {
    builder
        .getstatic(systemClass, "out", printStreamClass)
        .ldc(message)
        .invokevirtual(printStreamClass, "println", printlnType);
  }

  private static boolean isMainMethod(MethodModel methodModel) {
    return "main".equals(methodModel.methodName().stringValue());
  }

  private static void runGeneratedClass(byte[] bytes) throws ReflectiveOperationException {
    GeneratedClassLoader classLoader = new GeneratedClassLoader();
    Class<?> generatedClass = classLoader.define(bytes);
    Method mainMethod = generatedClass.getMethod("main", String[].class);
    mainMethod.invoke(null, (Object) new String[0]);
  }

  private static final class GeneratedClassLoader extends ClassLoader {

    private Class<?> define(byte[] bytes) {
      return defineClass(null, bytes, 0, bytes.length);
    }
  }
}
