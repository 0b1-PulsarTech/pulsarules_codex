void main() {
  group('outer', () {
    test('inner', () {
      if (1 > 0) {
        expect(1, 1);
      }
    });
    if (true) {}
  });
  other.test('inner', () {});
}
