export function matchesRuleSearch(query: string, columns: string[]): boolean {
  const values = columns.map((value) => value.toLocaleLowerCase());
  return query
    .trim()
    .toLocaleLowerCase()
    .split(/\s+/)
    .filter(Boolean)
    .every((term) => values.some((value) => value.includes(term)));
}
