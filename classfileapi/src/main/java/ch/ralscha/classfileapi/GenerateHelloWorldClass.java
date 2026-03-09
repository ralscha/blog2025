package ch.ralscha.classfileapi;

import java.lang.classfile.ClassFile;
import java.lang.constant.ClassDesc;
import java.lang.constant.ConstantDescs;
import java.lang.constant.MethodTypeDesc;
import java.lang.reflect.Method;
import java.nio.file.Files;
import java.nio.file.Path;

public class GenerateHelloWorldClass {

  public static void main(String[] args) throws Exception {
    ClassDesc helloClass = ClassDesc.of("Hello");
    ClassDesc systemClass = ClassDesc.of("java.lang.System");
    ClassDesc printStreamClass = ClassDesc.of("java.io.PrintStream");
    MethodTypeDesc mainType =
        MethodTypeDesc.of(ConstantDescs.CD_void, ConstantDescs.CD_String.arrayType());
    MethodTypeDesc printlnType = MethodTypeDesc.of(ConstantDescs.CD_void, ConstantDescs.CD_String);

    byte[] bytes =
        ClassFile.of()
            .build(
                helloClass,
                classBuilder ->
                    classBuilder
                        .withFlags(ClassFile.ACC_PUBLIC)
                        .withMethodBody(
                            ConstantDescs.INIT_NAME,
                            ConstantDescs.MTD_void,
                            ClassFile.ACC_PUBLIC,
                            codeBuilder ->
                                codeBuilder
                                    .aload(0)
                                    .invokespecial(
                                        ConstantDescs.CD_Object,
                                        ConstantDescs.INIT_NAME,
                                        ConstantDescs.MTD_void)
                                    .return_())
                        .withMethodBody(
                            "main",
                            mainType,
                            ClassFile.ACC_PUBLIC | ClassFile.ACC_STATIC,
                            codeBuilder ->
                                codeBuilder
                                    .getstatic(systemClass, "out", printStreamClass)
                                    .ldc("Hello, world!")
                                    .invokevirtual(printStreamClass, "println", printlnType)
                                    .return_()));

    runGeneratedClass(bytes);

    Path output = Path.of("target", "generated-classes", "Hello.class");
    Files.createDirectories(output.getParent());
    Files.write(output, bytes);
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
