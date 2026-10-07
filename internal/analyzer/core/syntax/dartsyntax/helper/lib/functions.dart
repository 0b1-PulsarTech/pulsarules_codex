import 'package:analyzer/dart/ast/ast.dart';
import 'package:analyzer/dart/ast/token.dart';

import 'blocks.dart';
import 'complexity.dart';
import 'frame.dart';
import 'lines.dart';

const _testCalls = {
  'test',
  'testWidgets',
  'group',
  'setUp',
  'setUpAll',
  'tearDown',
  'tearDownAll',
};

class FunctionCollector extends ComplexityCounter {
  FunctionCollector(this.lines, this.isTest);

  final Lines lines;
  final bool isTest;
  final functions = <Frame>[];
  final blocks = <Map<String, Object>>[];

  void _scope(Frame? frame, void Function() visitChildren) {
    if (frame == null) {
      visitChildren();
      return;
    }
    functions.add(frame);
    stack.add(frame);
    visitChildren();
    stack.removeLast();
  }

  Frame? _frame(
    String name,
    String kind,
    Token first,
    FunctionBody body, {
    required FormalParameterList? params,
    bool nested = false,
    bool testContainer = false,
  }) {
    final int start, end;
    if (body is BlockFunctionBody) {
      start = lines.of(body.block.leftBracket);
      end = lines.of(body.block.rightBracket);
      if (!nested || testContainer)
        blocks.addAll(blockDepths(body.block, lines));
    } else if (body is ExpressionFunctionBody) {
      start = lines.of(body.expression.beginToken);
      end = lines.endOf(body.expression);
    } else {
      return null;
    }
    final all = params?.parameters ?? const <FormalParameter>[];
    final named = all.where((param) => param.isNamed).length;
    return Frame(
      name: name,
      kind: kind,
      line: lines.of(first),
      start: start,
      end: end,
      params: all.length - named,
      named: named,
      nested: nested,
      testContainer: testContainer,
    );
  }

  String _accessorOr(bool getter, bool setter, String otherwise) =>
      getter || setter ? 'accessor' : otherwise;

  @override
  void visitFunctionDeclaration(FunctionDeclaration node) {
    final expression = node.functionExpression;
    final frame = _frame(
      node.name.lexeme,
      _accessorOr(node.isGetter, node.isSetter, 'function'),
      node.firstTokenAfterCommentAndMetadata,
      expression.body,
      params: expression.parameters,
      testContainer:
          isTest &&
          node.name.lexeme == 'main' &&
          node.parent is CompilationUnit,
    );
    _scope(frame, () => super.visitFunctionDeclaration(node));
  }

  @override
  void visitMethodDeclaration(MethodDeclaration node) {
    final frame = _frame(
      node.name.lexeme,
      _accessorOr(node.isGetter, node.isSetter, 'method'),
      node.firstTokenAfterCommentAndMetadata,
      node.body,
      params: node.parameters,
    );
    _scope(frame, () => super.visitMethodDeclaration(node));
  }

  @override
  void visitConstructorDeclaration(ConstructorDeclaration node) {
    final owner = node.typeName?.name ?? '';
    final suffix = node.name == null ? '' : '.${node.name!.lexeme}';
    final frame = _frame(
      '$owner$suffix',
      'constructor',
      node.firstTokenAfterCommentAndMetadata,
      node.body,
      params: node.parameters,
    );
    _scope(frame, () => super.visitConstructorDeclaration(node));
  }

  @override
  void visitFunctionExpression(FunctionExpression node) {
    final parent = node.parent;
    if (parent is FunctionDeclaration) {
      super.visitFunctionExpression(node);
      return;
    }
    final bound = parent is VariableDeclaration && parent.initializer == node;
    final frame = _frame(
      bound ? parent.name.lexeme : 'closure',
      'closure',
      node.beginToken,
      node.body,
      params: node.parameters,
      nested: !bound,
      testContainer: isTest && _isTestArgument(node),
    );
    _scope(frame, () => super.visitFunctionExpression(node));
  }

  bool _isTestArgument(FunctionExpression node) {
    final list = node.parent;
    final call = list?.parent;
    return list is ArgumentList &&
        call is MethodInvocation &&
        call.target == null &&
        _testCalls.contains(call.methodName.name);
  }
}
