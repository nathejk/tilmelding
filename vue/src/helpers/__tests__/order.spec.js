import { describe, it, expect } from 'vitest'
import { aggregateOrderLines, orderTotalDkk, orderDueDkk, orderedSize } from '../order'

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

/**
 * orderedSize reads what a member will actually be given from the orders, not
 * from their own record. Once a product is closed for sale the two disagree: the
 * backend keeps a cancelled selection on the member projection but drops the line
 * from the open order, so only the orders say what will be produced.
 */
describe('orderedSize', () => {
  const paidShirt = {
    status: 'paid',
    lines: [
      { productSku: 'participation.patrulje', memberId: 'm-1', quantity: 1 },
      { productSku: 'tshirt.adult', memberId: 'm-1', quantity: 1, attributes: { size: 'l' } }
    ]
  }

  it('finds the size a member has a line for', () => {
    expect(orderedSize(paidShirt, 'm-1')).toBe('l')
  })

  it('returns empty for a member with no t-shirt line', () => {
    expect(orderedSize(paidShirt, 'm-2')).toBe('')
  })

  it('searches the open order and the paid history together', () => {
    const open = {
      status: 'open',
      lines: [
        { productSku: 'tshirt.adult', memberId: 'm-2', quantity: 1, attributes: { size: 'xxl' } }
      ]
    }
    expect(orderedSize([open, paidShirt], 'm-1')).toBe('l')
    expect(orderedSize([open, paidShirt], 'm-2')).toBe('xxl')
  })

  it('nets out a zero-sum size change and reports the surviving size', () => {
    // The size handed back carries a negative quantity; only the size now owed
    // should be shown. Order of the pair must not matter.
    const exchanged = {
      lines: [
        { productSku: 'tshirt.adult', memberId: 'm-1', quantity: -1, attributes: { size: 'l' } },
        { productSku: 'tshirt.adult', memberId: 'm-1', quantity: 1, attributes: { size: '3xl' } }
      ]
    }
    expect(orderedSize(exchanged, 'm-1')).toBe('3xl')

    const reversed = { lines: [...exchanged.lines].reverse() }
    expect(orderedSize(reversed, 'm-1')).toBe('3xl')
  })

  it('reports nothing when a paid shirt was credited across orders', () => {
    // The credit lives on one order and the charge on another: summing across
    // both is what keeps the answer honest.
    const paid = {
      lines: [
        { productSku: 'tshirt.adult', memberId: 'm-1', quantity: 1, attributes: { size: 'l' } }
      ]
    }
    const open = {
      lines: [
        { productSku: 'tshirt.adult', memberId: 'm-1', quantity: -1, attributes: { size: 'l' } }
      ]
    }
    expect(orderedSize([paid, open], 'm-1')).toBe('')
  })

  it('handles nulls, missing lines and a missing memberId', () => {
    expect(orderedSize(null, 'm-1')).toBe('')
    expect(orderedSize([null, undefined], 'm-1')).toBe('')
    expect(orderedSize({}, 'm-1')).toBe('')
    expect(orderedSize(paidShirt, '')).toBe('')
    expect(orderedSize(paidShirt, undefined)).toBe('')
  })

  it('only matches the requested product', () => {
    const mixed = {
      lines: [
        { productSku: 'tshirt.plain', memberId: 'm-1', quantity: 1, attributes: { size: 's' } },
        { productSku: 'tshirt.adult', memberId: 'm-1', quantity: 1, attributes: { size: 'l' } }
      ]
    }
    expect(orderedSize(mixed, 'm-1')).toBe('l')
    expect(orderedSize(mixed, 'm-1', 'tshirt.plain')).toBe('s')
  })
})

/**
 * A real case from the dev database: a gøgler who changed size twice, so the
 * exchanges are spread across two separate paid orders.
 *
 *   order A: -1 3xl, +1 l     (3xl handed back, l wanted)
 *   order B: -1 l,   +1 xs    (l handed back, xs wanted)
 *
 * Only xs has a positive net once both orders are summed. Reading either order on
 * its own would answer 'l' or 'xs' depending on which one you happened to look
 * at — which is why orderedSize sums across every order it is given rather than
 * resolving one at a time.
 */
describe('orderedSize across chained size changes', () => {
  const orderA = {
    status: 'paid',
    lines: [
      { productSku: 'tshirt.adult', memberId: 'u-1', quantity: -1, attributes: { size: '3xl' } },
      { productSku: 'tshirt.adult', memberId: 'u-1', quantity: 1, attributes: { size: 'l' } }
    ]
  }
  const orderB = {
    status: 'paid',
    lines: [
      { productSku: 'tshirt.adult', memberId: 'u-1', quantity: -1, attributes: { size: 'l' } },
      { productSku: 'tshirt.adult', memberId: 'u-1', quantity: 1, attributes: { size: 'xs' } }
    ]
  }

  it('reports the size that survives both exchanges', () => {
    expect(orderedSize([orderA, orderB], 'u-1')).toBe('xs')
  })

  it('is independent of the order the orders arrive in', () => {
    expect(orderedSize([orderB, orderA], 'u-1')).toBe('xs')
  })
})
