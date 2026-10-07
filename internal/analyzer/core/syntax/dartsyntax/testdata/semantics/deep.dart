int deep(int value, bool flag) {
  if (flag) {
    value++;
  } else if (value > 1) {
    value--;
  } else {
    value = 0;
  }
  switch (value) {
    case 1:
      if (flag) {
        value = 2;
      }
    default:
      break;
  }
  return value;
}
