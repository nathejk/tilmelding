/**
 * Helpers for rendering the server-side `order` envelope returned by the
 * patrulje / klan / personnel endpoints. The backend stores all amounts in
 * minor units (øre) — these helpers convert to whole DKK for display and
 * aggregate per-member derived lines back into one row per product.
 *
 * The shape of an order is:
 *   {
 *     orderId: "...",
 *     status: "open" | "paid" | "cancelled",
 *     currency: "DKK",
 *     totalAmount: 177500,      // øre
 *     paidAmount: 0,            // øre
 *     dueAmount: 177500,        // øre
 *     lines: [{
 *       lineId, productSku, productName,
 *       unitPrice, quantity, lineTotal,    // all øre
 *       origin: "derived" | "manual",
 *       attributes: { memberId?, size?, ... }
 *     }, ...],
 *     ...
 *   }
 *
 * `order` is null when the team / person hasn't saved any state yet (the
 * server returns it as null in that case). All helpers handle null safely.
 */

const oreToDkk = (ore) => Math.round((Number(ore) || 0) / 100)

/**
 * aggregateOrderLines collapses derived per-member lines into one row per
 * productSku for display. T-shirts further break out by attributes.size so
 * the backoffice can see the size mix without scrolling.
 *
 * Each returned row has the legacy {text, count, unitPrice, amount} shape
 * the existing templates already render, so view changes are minimal.
 */
export function aggregateOrderLines(order) {
  if (!order || !Array.isArray(order.lines)) return []

  const groups = new Map()
  for (const line of order.lines) {
    if (!line) continue
    const size = line.attributes && line.attributes.size
    const key = size ? `${line.productSku}|${size}` : line.productSku
    const text = size ? `${line.productName} (${size.toUpperCase()})` : line.productName
    const existing = groups.get(key)
    if (existing) {
      existing.count += line.quantity || 0
      existing.amountOre += line.lineTotal || 0
    } else {
      groups.set(key, {
        text,
        count: line.quantity || 0,
        unitPriceOre: line.unitPrice || 0,
        amountOre: line.lineTotal || 0
      })
    }
  }

  return Array.from(groups.values()).map((g) => ({
    text: g.text,
    count: g.count,
    unitPrice: oreToDkk(g.unitPriceOre),
    amount: oreToDkk(g.amountOre)
  }))
}

export function orderTotalDkk(order) {
  return oreToDkk(order && order.totalAmount)
}

export function orderPaidDkk(order) {
  return oreToDkk(order && order.paidAmount)
}

export function orderDueDkk(order) {
  return Math.max(0, oreToDkk(order && order.dueAmount))
}

/** True when there's anything left to pay. */
export function orderHasDue(order) {
  return orderDueDkk(order) > 0
}

/** True when the order is locked (paid or cancelled) and the form should be read-only. */
export function orderIsLocked(order) {
  return !!order && (order.status === 'paid' || order.status === 'cancelled')
}

/**
 * orderShortLines returns a compact summary string of an order's lines,
 * collapsed by SKU+size in the same way the full grid does. Suitable for
 * a one-liner in the paid-orders history (e.g. "4× Patrulje-deltagelse,
 * 1× T-shirt (L)").
 */
export function orderShortLines(order) {
  return aggregateOrderLines(order)
    .map((line) => `${line.count}× ${line.text}`)
    .join(', ')
}

/**
 * orderDateShort extracts the YYYY-MM-DD prefix from an order's
 * changedAt timestamp. The backend stores time.Time.String() output
 * which is not natively parseable by the Date constructor; the prefix
 * is unambiguous and good enough for a payment-history label.
 */
export function orderDateShort(order) {
  if (!order || !order.changedAt) return ''
  return String(order.changedAt).substring(0, 10)
}

/**
 * totalPaidDkk sums paidAmount across the open order (if any) and every
 * paid order in the history list. Used by the "Indbetalt" total which
 * spans the full payment history, not just the currently-open cart.
 */
export function totalPaidDkk(open, paidOrders) {
  let total = orderPaidDkk(open)
  if (Array.isArray(paidOrders)) {
    for (const po of paidOrders) {
      total += orderPaidDkk(po)
    }
  }
  return total
}
