import { describe, it, expect } from 'vitest'
import { aggregateOrderLines, orderTotalDkk, orderDueDkk } from '../order'

/**
 * A free t-shirt size change is stored as a zero-sum pair of lines. These rows
 * are the only on-page confirmation that the new size was registered, so they
 * must survive aggregation as two distinct rows with the sign intact — see
 * PRD 002.
 */
describe('aggregateOrderLines with a size-change credit', () => {
  const order = {
    totalAmount: 0,
    dueAmount: 0,
    lines: [
      {
        productSku: 'tshirt.adult',
        productName: 'T-shirt',
        unitPrice: 17500,
        quantity: 1,
        lineTotal: 17500,
        attributes: { size: '3xl' }
      },
      {
        productSku: 'tshirt.adult',
        productName: 'T-shirt',
        unitPrice: 17500,
        quantity: -1,
        lineTotal: -17500,
        attributes: { size: 'xxl' }
      }
    ]
  }

  it('keeps the two sizes as separate rows', () => {
    const rows = aggregateOrderLines(order)
    expect(rows).toHaveLength(2)
    expect(rows.map((r) => r.text)).toEqual(['T-shirt (3XL)', 'T-shirt (XXL)'])
  })

  it('preserves the negative count and amount', () => {
    const [gained, returned] = aggregateOrderLines(order)
    expect(gained.count).toBe(1)
    expect(gained.amount).toBe(175)
    expect(returned.count).toBe(-1)
    expect(returned.amount).toBe(-175)
  })

  it('nets to nothing owed, so the change is free', () => {
    expect(orderTotalDkk(order)).toBe(0)
    expect(orderDueDkk(order)).toBe(0)
  })

  it('does not collapse a credit into the charge of the same product', () => {
    // Same sku, same price, opposite signs: grouping is per (sku, size), so a
    // credit must never cancel out the charge it is paired with and leave the
    // user looking at an empty order.
    const rows = aggregateOrderLines(order)
    expect(rows.some((r) => r.count < 0)).toBe(true)
    expect(rows.some((r) => r.count > 0)).toBe(true)
  })
})
