export function colorStyle(color: string | undefined) {
  if (!color || !/^#[0-9a-f]{6}$/i.test(color)) return undefined;
  const rgb = [1, 3, 5]
    .map((i) => parseInt(color.slice(i, i + 2), 16) / 255)
    .map((c) => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4));
  const luminance = rgb[0] * 0.2126 + rgb[1] * 0.7152 + rgb[2] * 0.0722;
  return {
    backgroundColor: color,
    borderColor: color,
    color:
      (luminance + 0.05) / 0.05 >= 1.05 / (luminance + 0.05)
        ? "#000000"
        : "#ffffff",
  };
}
