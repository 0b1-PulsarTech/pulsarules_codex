import 'package:analyzer/dart/ast/ast.dart';
import 'package:analyzer/dart/ast/token.dart';
import 'package:analyzer/dart/ast/visitor.dart';

import 'lines.dart';

class NumberCollector extends RecursiveAstVisitor<void> {
  NumberCollector(this.lines);

  final Lines lines;
  final numbers = <Map<String, Object>>[];
  int _constDepth = 0;

  void _add(Token token) {
    if (_constDepth > 0) return;
    numbers.add({'value': token.lexeme, 'line': lines.of(token)});
  }

  @override
  void visitVariableDeclarationList(VariableDeclarationList node) {
    final inConst = node.isConst;
    if (inConst) _constDepth++;
    super.visitVariableDeclarationList(node);
    if (inConst) _constDepth--;
  }

  @override
  void visitIntegerLiteral(IntegerLiteral node) {
    _add(node.literal);
    super.visitIntegerLiteral(node);
  }

  @override
  void visitDoubleLiteral(DoubleLiteral node) {
    _add(node.literal);
    super.visitDoubleLiteral(node);
  }
}
