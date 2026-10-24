import 'package:analyzer/dart/ast/ast.dart';
import 'package:analyzer/dart/ast/visitor.dart';

import 'lines.dart';

class ElseCollector extends RecursiveAstVisitor<void> {
  ElseCollector(this.lines);

  final Lines lines;
  final elses = <Map<String, Object>>[];

  @override
  void visitIfStatement(IfStatement node) {
    final keyword = node.elseKeyword;
    if (keyword != null) {
      elses.add({'line': lines.of(keyword), 'afterExit': _chainExits(node)});
    }
    super.visitIfStatement(node);
  }

  @override
  void visitIfElement(IfElement node) {
    final keyword = node.elseKeyword;
    if (keyword != null) {
      elses.add({'line': lines.of(keyword), 'afterExit': false});
    }
    super.visitIfElement(node);
  }
}

bool _chainExits(IfStatement node) {
  var root = node;
  final chain = [node];
  while (true) {
    final parent = root.parent;
    if (parent is! IfStatement || !identical(parent.elseStatement, root)) {
      break;
    }
    root = parent;
    chain.add(root);
  }
  final labels = _labelsAround(root);
  return chain.every((link) => _exits(link.thenStatement, labels));
}

Set<String> _labelsAround(Statement node) {
  final labels = <String>{};
  var parent = node.parent;
  while (parent is LabeledStatement) {
    for (final label in parent.labels) {
      labels.add(label.name.lexeme);
    }
    parent = parent.parent;
  }
  return labels;
}

bool _exits(Statement node, Set<String> labels) {
  if (node is Block) {
    return node.statements.isNotEmpty && _exits(node.statements.last, labels);
  }
  if (node is ExpressionStatement) {
    final expression = node.expression;
    return expression is ThrowExpression || expression is RethrowExpression;
  }
  if (node is BreakStatement) {
    final label = node.label;
    return label == null || !labels.contains(label.name.lexeme);
  }
  return node is ReturnStatement || node is ContinueStatement;
}
