import 'package:analyzer/dart/ast/ast.dart';
import 'package:analyzer/dart/ast/token.dart';
import 'package:analyzer/dart/ast/visitor.dart';

import 'lines.dart';

class NameCollector extends RecursiveAstVisitor<void> {
  NameCollector(this.lines);

  final Lines lines;
  final names = <Map<String, Object>>[];
  final _types = <bool>[];
  int _local = 0;

  void _add(Token? token, String kind, {bool member = false}) {
    if (token == null || isBlankName(token.lexeme)) return;
    final public =
        member &&
        _local == 0 &&
        !isPrivate(token.lexeme) &&
        (_types.isEmpty || _types.last);
    names.add({
      'name': token.lexeme,
      'kind': kind,
      'line': lines.of(token),
      'exported': public,
      'exempt': false,
    });
  }

  void _type(Token? name, void Function() visitChildren) {
    _add(name, 'type', member: true);
    _types.add(name != null && !isPrivate(name.lexeme));
    visitChildren();
    _types.removeLast();
  }

  @override
  void visitClassDeclaration(ClassDeclaration node) =>
      _type(node.namePart.typeName, () => super.visitClassDeclaration(node));

  @override
  void visitMixinDeclaration(MixinDeclaration node) =>
      _type(node.name, () => super.visitMixinDeclaration(node));

  @override
  void visitEnumDeclaration(EnumDeclaration node) =>
      _type(node.namePart.typeName, () => super.visitEnumDeclaration(node));

  @override
  void visitExtensionDeclaration(ExtensionDeclaration node) =>
      _type(node.name, () => super.visitExtensionDeclaration(node));

  @override
  void visitExtensionTypeDeclaration(ExtensionTypeDeclaration node) => _type(
    node.namePart.typeName,
    () => super.visitExtensionTypeDeclaration(node),
  );

  @override
  void visitGenericTypeAlias(GenericTypeAlias node) {
    _add(node.name, 'type', member: true);
    super.visitGenericTypeAlias(node);
  }

  @override
  void visitFunctionTypeAlias(FunctionTypeAlias node) {
    _add(node.name, 'type', member: true);
    super.visitFunctionTypeAlias(node);
  }

  @override
  void visitClassTypeAlias(ClassTypeAlias node) {
    _add(node.name, 'type', member: true);
    super.visitClassTypeAlias(node);
  }

  @override
  void visitEnumConstantDeclaration(EnumConstantDeclaration node) {
    _add(node.name, 'enum value', member: true);
    super.visitEnumConstantDeclaration(node);
  }

  @override
  void visitFunctionDeclaration(FunctionDeclaration node) {
    _add(node.name, 'function', member: true);
    super.visitFunctionDeclaration(node);
  }

  @override
  void visitMethodDeclaration(MethodDeclaration node) {
    if (!node.isOperator) _add(node.name, 'method', member: true);
    super.visitMethodDeclaration(node);
  }

  @override
  void visitVariableDeclaration(VariableDeclaration node) {
    final list = node.parent as VariableDeclarationList;
    final owner = list.parent;
    final constant = list.isConst ? 'constant' : 'variable';
    if (owner is FieldDeclaration) {
      _add(node.name, list.isConst ? 'constant' : 'field', member: true);
    } else if (owner is TopLevelVariableDeclaration) {
      _add(node.name, constant, member: true);
    } else if (owner is ForPartsWithDeclarations) {
      _add(node.name, 'loop variable');
    } else {
      _add(node.name, constant);
    }
    super.visitVariableDeclaration(node);
  }

  @override
  void visitDeclaredIdentifier(DeclaredIdentifier node) {
    _add(node.name, 'loop variable');
    super.visitDeclaredIdentifier(node);
  }

  @override
  void visitDeclaredVariablePattern(DeclaredVariablePattern node) {
    _add(node.name, 'variable');
    super.visitDeclaredVariablePattern(node);
  }

  @override
  void visitFormalParameterList(FormalParameterList node) {
    for (final parameter in node.parameters) {
      if (parameter is FieldFormalParameter ||
          parameter is SuperFormalParameter) {
        continue;
      }
      _add(parameter.name, 'parameter');
    }
    super.visitFormalParameterList(node);
  }

  @override
  void visitImportDirective(ImportDirective node) {
    _add(node.prefix?.token, 'import alias');
    super.visitImportDirective(node);
  }

  @override
  void visitLabel(Label node) {
    final owner = node.parent;
    if (owner is LabeledStatement || owner is SwitchMember)
      _add(node.name, 'label');
    super.visitLabel(node);
  }

  @override
  void visitCatchClause(CatchClause node) {
    _add(node.exceptionParameter?.name, 'catch parameter');
    _add(node.stackTraceParameter?.name, 'catch parameter');
    super.visitCatchClause(node);
  }

  @override
  void visitTypeParameter(TypeParameter node) {
    _add(node.name, 'type parameter');
    super.visitTypeParameter(node);
  }

  @override
  void visitBlockFunctionBody(BlockFunctionBody node) {
    _local++;
    super.visitBlockFunctionBody(node);
    _local--;
  }

  @override
  void visitExpressionFunctionBody(ExpressionFunctionBody node) {
    _local++;
    super.visitExpressionFunctionBody(node);
    _local--;
  }
}
