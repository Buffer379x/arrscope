// A presentation preference only: never changes stored traits, queries or totals.
export function tagDisplayKey(field: string, value: string) {
  return field.split(" ")[0] + ":" + (value || "__unknown__");
}
export function tagIsDisplayed(
  hidden: Record<string, boolean>,
  field: string,
  value: string,
) {
  return hidden[tagDisplayKey(field, value)] !== true;
}
