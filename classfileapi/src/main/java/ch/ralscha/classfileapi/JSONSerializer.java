package ch.ralscha.classfileapi;

import java.lang.classfile.ClassFile;
import java.lang.classfile.CodeBuilder;
import java.lang.classfile.TypeKind;
import java.lang.constant.ClassDesc;
import java.lang.constant.ConstantDescs;
import java.lang.constant.MethodTypeDesc;
import java.lang.invoke.MethodHandles;
import java.lang.invoke.MethodType;
import java.lang.reflect.RecordComponent;
import java.util.Objects;
import java.util.concurrent.atomic.AtomicInteger;

public interface JSONSerializer<T> {

  AtomicInteger COUNTER = new AtomicInteger();

  String serialize(T value);

  @SuppressWarnings("unchecked")
  static <T> JSONSerializer<T> from(Class<T> recordClass) {
    Objects.requireNonNull(recordClass, "recordClass must not be null");
    validateRecordClass(recordClass);

    byte[] classBytes = buildImplementation(recordClass);

    try {
      MethodHandles.Lookup lookup =
          MethodHandles.privateLookupIn(recordClass, MethodHandles.lookup());
      MethodHandles.Lookup hiddenLookup =
          lookup.defineHiddenClass(classBytes, true, MethodHandles.Lookup.ClassOption.NESTMATE);
      Class<?> implementationClass = hiddenLookup.lookupClass();
      return (JSONSerializer<T>)
          hiddenLookup
              .findConstructor(implementationClass, MethodType.methodType(void.class))
              .invoke();
    } catch (Throwable exception) {
      throw new IllegalStateException("Failed to instantiate generated serializer", exception);
    }
  }

  private static void validateRecordClass(Class<?> recordClass) {
    if (!recordClass.isRecord()) {
      throw new IllegalArgumentException("Only records are supported");
    }

    for (RecordComponent component : recordClass.getRecordComponents()) {
      if (component.getType() != String.class) {
        throw new IllegalArgumentException("Only record components of type String are supported");
      }
    }
  }

  private static byte[] buildImplementation(Class<?> recordClass) {
    RecordComponent[] components = recordClass.getRecordComponents();
    ClassDesc implementationClass = ClassDesc.of(implementationClassName(recordClass));
    ClassDesc recordClassDesc = ClassDesc.of(recordClass.getName());
    ClassDesc serializerInterface = ClassDesc.of(JSONSerializer.class.getName());
    ClassDesc stringBuilderClass = ClassDesc.of("java.lang.StringBuilder");

    MethodTypeDesc constructorType = ConstantDescs.MTD_void;
    MethodTypeDesc serializeType =
        MethodTypeDesc.of(ConstantDescs.CD_String, ConstantDescs.CD_Object);
    MethodTypeDesc appendType = MethodTypeDesc.of(stringBuilderClass, ConstantDescs.CD_String);
    MethodTypeDesc toStringType = MethodTypeDesc.of(ConstantDescs.CD_String);

    return ClassFile.of()
        .build(
            implementationClass,
            classBuilder ->
                classBuilder
                    .withFlags(ClassFile.ACC_PUBLIC | ClassFile.ACC_FINAL)
                    .withInterfaceSymbols(serializerInterface)
                    .withMethodBody(
                        ConstantDescs.INIT_NAME,
                        constructorType,
                        ClassFile.ACC_PUBLIC,
                        codeBuilder -> buildConstructor(codeBuilder, constructorType))
                    .withMethodBody(
                        "serialize",
                        serializeType,
                        ClassFile.ACC_PUBLIC,
                        codeBuilder ->
                            buildSerializeMethod(
                                codeBuilder,
                                components,
                                recordClassDesc,
                                stringBuilderClass,
                                constructorType,
                                appendType,
                                toStringType)));
  }

  private static void buildConstructor(CodeBuilder codeBuilder, MethodTypeDesc constructorType) {
    codeBuilder
        .aload(codeBuilder.receiverSlot())
        .invokespecial(ConstantDescs.CD_Object, ConstantDescs.INIT_NAME, constructorType)
        .return_();
  }

  private static void buildSerializeMethod(
      CodeBuilder codeBuilder,
      RecordComponent[] components,
      ClassDesc recordClassDesc,
      ClassDesc stringBuilderClass,
      MethodTypeDesc constructorType,
      MethodTypeDesc appendType,
      MethodTypeDesc toStringType) {
    if (components.length == 0) {
      codeBuilder.ldc("{}").areturn();
      return;
    }
    int recordSlot1 = codeBuilder.allocateLocal(TypeKind.REFERENCE);

    codeBuilder
        .aload(codeBuilder.parameterSlot(0))
        .checkcast(recordClassDesc)
        .astore(recordSlot1)
        .new_(stringBuilderClass)
        .dup()
        .invokespecial(stringBuilderClass, ConstantDescs.INIT_NAME, constructorType);

    int recordSlot = recordSlot1;

    for (int index = 0; index < components.length; index++) {
      RecordComponent component = components[index];
      codeBuilder
          .ldc(componentPrefix(component, index))
          .invokevirtual(stringBuilderClass, "append", appendType)
          .aload(recordSlot)
          .invokevirtual(
              recordClassDesc, component.getName(), MethodTypeDesc.of(ConstantDescs.CD_String))
          .invokevirtual(stringBuilderClass, "append", appendType)
          .ldc("\"")
          .invokevirtual(stringBuilderClass, "append", appendType);
    }

    codeBuilder
        .ldc("}")
        .invokevirtual(stringBuilderClass, "append", appendType)
        .invokevirtual(stringBuilderClass, "toString", toStringType)
        .areturn();
  }

  private static String componentPrefix(RecordComponent component, int index) {
    return index == 0
        ? "{\"" + component.getName() + "\":\""
        : ",\"" + component.getName() + "\":\"";
  }

  private static String implementationClassName(Class<?> recordClass) {
    String packageName = recordClass.getPackageName();
    String simpleName =
        recordClass.getName().replace('.', '_').replace('$', '_')
            + "JsonSerializer"
            + COUNTER.incrementAndGet();
    return packageName.isEmpty() ? simpleName : packageName + "." + simpleName;
  }
}
