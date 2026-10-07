import 'package:analyzer/dart/ast/ast.dart';
import 'package:analyzer/dart/ast/visitor.dart';

import 'doc_text.dart';
import 'lines.dart';

class DocCollector extends RecursiveAstVisitor<void> {
  DocCollector(this.lines, this.isTest);

  final Lines lines;
  final bool isTest;
  final sites = <Map<String, Object>>[];
  final _owners = <String>[];

  void _site(
    AnnotatedNode node,
    String name,
    String kind, {
    bool exempt = false,
  }) {
    final doc = node.documentationComment;
    if (doc == null || doc.tokens.isEmpty) return;
    final first = node.metadata.isEmpty
        ? node.firstTokenAfterCommentAndMetadata
        : node.metadata.first.beginToken;
    final shape = docShape(doc, lines, first.precedingComments);
    final symbol = _owners.isEmpty ? name : '${_owners.last}.$name';
    sites.add({
      'symbol': symbol,
      'kind': kind,
      'line': lines.of(doc.tokens.first),
      'exempt': shape.exempt || exempt,
      'mixedDirective': shape.mixed,
    });
  }

  void _type(AnnotatedNode node, String? name, void Function() visitChildren) {
    final label = name ?? 'extension';
    _site(node, label, 'type');
    _owners.add(label);
    visitChildren();
    _owners.removeLast();
  }

  @override
  void visitClassDeclaration(ClassDeclaration node) => _type(
    node,
    node.namePart.typeName.lexeme,
    () => super.visitClassDeclaration(node),
  );

  @override
  void visitMixinDeclaration(MixinDeclaration node) =>
      _type(node, node.name.lexeme, () => super.visitMixinDeclaration(node));

  @override
  void visitEnumDeclaration(EnumDeclaration node) => _type(
    node,
    node.namePart.typeName.lexeme,
    () => super.visitEnumDeclaration(node),
  );

  @override
  void visitExtensionDeclaration(ExtensionDeclaration node) => _type(
    node,
    node.name?.lexeme,
    () => super.visitExtensionDeclaration(node),
  );

  @override
  void visitExtensionTypeDeclaration(ExtensionTypeDeclaration node) => _type(
    node,
    node.namePart.typeName.lexeme,
    () => super.visitExtensionTypeDeclaration(node),
  );

  @override
  void visitGenericTypeAlias(GenericTypeAlias node) {
    _site(node, node.name.lexeme, 'type');
    super.visitGenericTypeAlias(node);
  }

  @override
  void visitFunctionTypeAlias(FunctionTypeAlias node) {
    _site(node, node.name.lexeme, 'type');
    super.visitFunctionTypeAlias(node);
  }

  @override
  void visitClassTypeAlias(ClassTypeAlias node) {
    _site(node, node.name.lexeme, 'type');
    super.visitClassTypeAlias(node);
  }

  @override
  void visitEnumConstantDeclaration(EnumConstantDeclaration node) {
    _site(node, node.name.lexeme, 'field');
    super.visitEnumConstantDeclaration(node);
  }

  @override
  void visitFunctionDeclaration(FunctionDeclaration node) {
    final main =
        isTest && node.name.lexeme == 'main' && node.parent is CompilationUnit;
    _site(node, node.name.lexeme, 'function', exempt: main);
    super.visitFunctionDeclaration(node);
  }

  @override
  void visitMethodDeclaration(MethodDeclaration node) {
    _site(node, node.name.lexeme, 'method');
    super.visitMethodDeclaration(node);
  }

  @override
  void visitConstructorDeclaration(ConstructorDeclaration node) {
    final suffix = node.name == null ? '' : node.name!.lexeme;
    _site(node, suffix.isEmpty ? 'new' : suffix, 'constructor');
    super.visitConstructorDeclaration(node);
  }

  @override
  void visitFieldDeclaration(FieldDeclaration node) {
    _site(node, node.fields.variables.first.name.lexeme, 'field');
    super.visitFieldDeclaration(node);
  }

  @override
  void visitTopLevelVariableDeclaration(TopLevelVariableDeclaration node) {
    _site(node, node.variables.variables.first.name.lexeme, 'variable');
    super.visitTopLevelVariableDeclaration(node);
  }
}
