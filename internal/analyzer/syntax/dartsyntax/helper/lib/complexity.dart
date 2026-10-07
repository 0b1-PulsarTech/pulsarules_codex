import 'package:analyzer/dart/ast/ast.dart';
import 'package:analyzer/dart/ast/token.dart';
import 'package:analyzer/dart/ast/visitor.dart';

import 'frame.dart';

class ComplexityCounter extends RecursiveAstVisitor<void> {
  final stack = <Frame>[];

  void bump() {
    for (final frame in stack.reversed) {
      frame.complexity++;
      if (!frame.feedsParent) break;
    }
  }

  bool _isDefault(GuardedPattern guarded) {
    final pattern = guarded.pattern;
    return pattern is WildcardPattern &&
        pattern.type == null &&
        guarded.whenClause == null;
  }

  void count(void Function() visitChildren) {
    bump();
    visitChildren();
  }

  @override
  void visitIfStatement(IfStatement node) =>
      count(() => super.visitIfStatement(node));

  @override
  void visitIfElement(IfElement node) =>
      count(() => super.visitIfElement(node));

  @override
  void visitForStatement(ForStatement node) =>
      count(() => super.visitForStatement(node));

  @override
  void visitForElement(ForElement node) =>
      count(() => super.visitForElement(node));

  @override
  void visitWhileStatement(WhileStatement node) =>
      count(() => super.visitWhileStatement(node));

  @override
  void visitDoStatement(DoStatement node) =>
      count(() => super.visitDoStatement(node));

  @override
  void visitSwitchCase(SwitchCase node) =>
      count(() => super.visitSwitchCase(node));

  @override
  void visitSwitchPatternCase(SwitchPatternCase node) {
    if (!_isDefault(node.guardedPattern)) bump();
    super.visitSwitchPatternCase(node);
  }

  @override
  void visitSwitchExpressionCase(SwitchExpressionCase node) {
    if (!_isDefault(node.guardedPattern)) bump();
    super.visitSwitchExpressionCase(node);
  }

  @override
  void visitCatchClause(CatchClause node) =>
      count(() => super.visitCatchClause(node));

  @override
  void visitConditionalExpression(ConditionalExpression node) =>
      count(() => super.visitConditionalExpression(node));

  @override
  void visitBinaryExpression(BinaryExpression node) {
    final type = node.operator.type;
    if (type == TokenType.AMPERSAND_AMPERSAND || type == TokenType.BAR_BAR)
      bump();
    super.visitBinaryExpression(node);
  }
}
