/** All prices travel as integer pence to avoid floating point drift. */
const gbp = new Intl.NumberFormat('en-GB', {
  style: 'currency',
  currency: 'GBP',
  minimumFractionDigits: 2,
});

export function formatPence(pence: number): string {
  return gbp.format(pence / 100);
}

/** "£45" for whole pounds, "£4.50" otherwise — used on product cards. */
export function formatPenceShort(pence: number): string {
  return pence % 100 === 0 ? `£${pence / 100}` : formatPence(pence);
}
