int loops(List<int> items, int? maybe) {
  var total = 0;
  while (total < 3) {
    total++;
  }
  total += maybe ?? 1;
  total += items.first?.sign ?? 0;
  var other = maybe;
  other ??= 2;
  return total + other;
}

List<int> spread(bool flag, List<int> items) => [
      if (flag) 1,
      for (final item in items) item,
    ];
