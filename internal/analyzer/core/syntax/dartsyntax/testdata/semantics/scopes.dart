int outer(List<int> items, bool flag) {
  int local(int value) {
    if (value > 1) {
      return 1;
    }
    return 0;
  }

  final bound = (int value) => value > 0 ? 1 : 0;
  items.forEach((item) {
    if (item > 2 && flag) {
      print(item);
    }
  });
  return local(1) + bound(2);
}

void main() {
  if (true) {}
}
