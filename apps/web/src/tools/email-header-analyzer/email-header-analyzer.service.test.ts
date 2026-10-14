import { describe, expect, it } from 'vitest'
import { authResults, dkimSignatures, formatDelay, hops, parseHeaders, summary } from './email-header-analyzer.service'

const SAMPLE = `Delivered-To: alice@example.net
Received: by 2002:a05:1234:5678 with SMTP id abc;
        Tue, 13 Oct 2026 10:00:07 -0700 (PDT)
Received: from mail.example.com (mail.example.com. [192.0.2.25])
        by mx.example.net with ESMTPS id xyz
        for <alice@example.net>;
        Tue, 13 Oct 2026 10:00:05 -0700 (PDT)
Received: from [10.0.0.5] (unknown [198.51.100.7])
\tby mail.example.com (Postfix) with ESMTPSA id 1234;
\tTue, 13 Oct 2026 17:00:01 +0000 (UTC)
Authentication-Results: mx.example.net;
       dkim=pass header.i=@example.com header.s=s1 header.b=abc;
       spf=pass (domain of bob@example.com designates 192.0.2.25 as permitted sender) smtp.mailfrom=bob@example.com;
       dmarc=pass (p=REJECT sp=REJECT dis=NONE) header.from=example.com
DKIM-Signature: v=1; a=rsa-sha256; c=relaxed/relaxed; d=example.com; s=s1;
        h=from:to:subject; bh=xyz; b=abc
From: Bob <bob@example.com>
To: alice@example.net
Subject: Hello
Date: Tue, 13 Oct 2026 17:00:00 +0000

Body text: not a header
`

describe('parseHeaders', () => {
  it('unfolds and stops at the body', () => {
    const h = parseHeaders(SAMPLE)
    expect(h.find(x => x.name === 'Subject')?.value).toBe('Hello')
    expect(h.some(x => x.name === 'Body text')).toBe(false)
    expect(h.filter(x => x.name === 'Received')).toHaveLength(3)
  })
})

describe('hops', () => {
  it('orders hops chronologically with delays', () => {
    const list = hops(parseHeaders(SAMPLE))
    expect(list.map(h => [h.from, h.by, h.with, h.delay])).toEqual([
      ['[10.0.0.5] (unknown [198.51.100.7])', 'mail.example.com (Postfix)', 'ESMTPSA', undefined],
      ['mail.example.com (mail.example.com. [192.0.2.25])', 'mx.example.net', 'ESMTPS', 4],
      ['', '2002:a05:1234:5678', 'SMTP', 2],
    ])
  })
})

describe('auth and DKIM', () => {
  it('parses Authentication-Results', () => {
    const r = authResults(parseHeaders(SAMPLE))
    expect(r.map(a => `${a.method}=${a.result}`)).toEqual(['dkim=pass', 'spf=pass', 'dmarc=pass'])
    expect(r[1]!.detail).toBe('smtp.mailfrom=bob@example.com')
  })
  it('parses DKIM signatures and the summary', () => {
    const h = parseHeaders(SAMPLE)
    expect(dkimSignatures(h)).toEqual([{ domain: 'example.com', selector: 's1', algorithm: 'rsa-sha256' }])
    expect(summary(h).map(x => x.name)).toEqual(['From', 'To', 'Subject', 'Date', 'Delivered-To'])
  })
})

describe('formatDelay', () => {
  it.each([[0, '0 s'], [5, '5 s'], [125, '2 min 5 s'], [3600, '1 h'], [-3, '−3 s']])('%d → %s', (s, want) => {
    expect(formatDelay(s)).toBe(want)
  })
})
