const timeoutSeconds = 30;
const limits = {
  'low': 5,
  'high': 50,
};
const colorMask = 0xFF;

class Retry {
  static const attempts = 3;
  final int backoff = 250;

  double half(double value) => value * 0.5;

  String label(int count) => 'items: ${count * 2}';

  int first(List<int> values) => values[0];

  void run() {
    const step = 7;
    final retries = 8;
    print(step + retries);
  }
}
