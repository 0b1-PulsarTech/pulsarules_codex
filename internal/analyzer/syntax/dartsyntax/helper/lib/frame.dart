class Frame {
  Frame({
    required this.name,
    required this.kind,
    required this.line,
    required this.start,
    required this.end,
    required this.params,
    required this.named,
    required this.nested,
    required this.testContainer,
  });

  final String name, kind;
  final int line, start, end, params, named;
  final bool nested, testContainer;
  int complexity = 1;

  bool get feedsParent => nested && !testContainer;

  Map<String, Object> toJson() => {
    'name': name,
    'kind': kind,
    'line': line,
    'bodyStart': start,
    'bodyEnd': end,
    'params': params,
    'namedParams': named,
    'complexity': complexity,
    'testContainer': testContainer,
    'nested': nested,
  };
}
