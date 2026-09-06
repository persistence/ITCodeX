/**
 * ITCodeX Meta Admin E2E smoke — simulates frontend API flows
 */
const BASE = 'http://127.0.0.1:8000'
const COL = `web_e2e_${Date.now().toString(36)}`

let passed = 0
let failed = 0
const bugs = []

/** 与前端 client.ts 一致：大整数转字符串，避免 Snowflake 精度丢失 */
function parseAPIJSON(text) {
  const safe = text.replace(/([:\[,]\s*)(-?\d{16,})(?=\s*[,\]}])/g, '$1"$2"')
  return JSON.parse(safe)
}

async function req(method, path, body, query) {
  let url = `${BASE}${path}`
  if (query) {
    const p = new URLSearchParams()
    for (const [k, v] of Object.entries(query)) {
      if (v === undefined || v === null) continue
      p.set(k, typeof v === 'object' ? JSON.stringify(v) : String(v))
    }
    url += `?${p}`
  }
  const res = await fetch(url, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  const text = await res.text()
  let json = {}
  try {
    json = parseAPIJSON(text)
  } catch {
    json = { code: res.status, message: text }
  }
  return { status: res.status, json }
}

function ok(name, cond, detail = '') {
  if (cond) {
    passed++
    console.log(`  ✓ ${name}`)
  } else {
    failed++
    console.log(`  ✗ ${name}${detail ? ' — ' + detail : ''}`)
    bugs.push({ name, detail })
  }
}

async function main() {
  console.log(`\nE2E against ${BASE}, collection=${COL}\n`)

  // 1. list
  console.log('1. Collections list')
  {
    const { json } = await req('GET', '/api/meta/collections', undefined, { pageSize: 50 })
    ok('list code=0', json.code === 0, JSON.stringify(json).slice(0, 120))
    ok('list has data.list', Array.isArray(json.data?.list))
  }

  // 2. create empty name (frontend bug regression)
  console.log('\n2. Create with empty name (expect fail)')
  {
    const { json } = await req('POST', '/api/meta/collections', {
      name: '',
      displayName: '测试',
      type: 'general',
      presetFields: ['id', 'createdAt', 'updatedAt'],
    })
    ok('empty name rejected', json.code !== 0, `code=${json.code} msg=${json.message}`)
  }

  // 3. create valid
  console.log('\n3. Create collection')
  {
    const { json } = await req('POST', '/api/meta/collections', {
      name: COL,
      displayName: 'Web E2E 测试表',
      type: 'general',
      description: 'automated',
      categories: ['e2e'],
      presetFields: ['id', 'createdAt', 'updatedAt', 'createdBy', 'updatedBy'],
      options: { simplePagination: false },
      fields: [
        {
          name: 'title',
          displayName: '标题',
          type: 'string',
          isRequired: true,
          validation: { maxLength: 100 },
        },
        {
          name: 'amount',
          displayName: '金额',
        type: 'double',
        isRequired: false,
      },
      {
        name: 'status',
        displayName: '状态',
        type: 'select',
        options: { enum: ['draft', 'done'], options: ['draft', 'done'] },
      },
      {
        name: 'secret',
        displayName: '密钥',
        type: 'password',
      },
      {
        name: 'birthday',
        displayName: '生日',
        type: 'date',
      },
    ],
  })
  ok('create code=0', json.code === 0, json.message)
  ok('create returns name', json.data?.name === COL, JSON.stringify(json.data)?.slice(0, 100))
}

  // 4. sync
  console.log('\n4. Sync')
  {
    const { json } = await req('POST', `/api/meta/collections/${COL}/sync`)
    ok('sync code=0', json.code === 0, json.message)
  }

  // 5. get detail + fields
  console.log('\n5. Get detail & fields')
  let fields = []
  {
    const { json } = await req('GET', `/api/meta/collections/${COL}`)
    ok('get code=0', json.code === 0)
    ok('has fields', (json.data?.fields?.length || 0) > 0, `count=${json.data?.fields?.length}`)
    ok('displayName ok', json.data?.displayName === 'Web E2E 测试表')
    fields = json.data?.fields || []
    const title = fields.find((f) => f.name === 'title')
    ok('title.required true', title?.required === true, JSON.stringify(title))
    const status = fields.find((f) => f.name === 'status')
    ok(
      'select options present',
      !!(status?.options?.enum || status?.options?.options),
      JSON.stringify(status?.options),
    )
  }

  // 6. add field
  console.log('\n6. Add field')
  {
    const { json } = await req('POST', `/api/meta/collections/${COL}/fields`, {
      name: 'memo',
      displayName: '备注',
      type: 'text',
    })
    ok('add field code=0', json.code === 0, json.message)
  }

  // 7. update field
  console.log('\n7. Update field')
  {
    const { json } = await req('PUT', `/api/meta/collections/${COL}/fields/memo`, {
      displayName: '备注说明',
      isRequired: false,
    })
    ok('update field code=0', json.code === 0, json.message)
    const memo = json.data?.list?.find?.((f) => f.name === 'memo') || (await req('GET', `/api/meta/collections/${COL}/fields`)).json.data?.list?.find((f) => f.name === 'memo')
    // list response after update
  }
  {
    const { json } = await req('GET', `/api/meta/collections/${COL}/fields`)
    const memo = json.data?.list?.find((f) => f.name === 'memo')
    ok('memo displayName updated', memo?.displayName === '备注说明', JSON.stringify(memo))
  }

  // 8. CRUD records
  console.log('\n8. Records CRUD')
  let recordId
  {
    const { json } = await req('POST', `/api/c/${COL}`, {
      title: '第一条',
      amount: 12.5,
      status: 'draft',
      secret: 'p@ss',
      birthday: '2026-09-06',
    })
    ok('create record', json.code === 0, json.message)
    recordId = json.data?.id
    ok('has id', recordId != null, String(recordId))
  }
  {
    // except must use field names, not type names
    const { json } = await req('GET', `/api/c/${COL}`, undefined, {
      page: 1,
      pageSize: 20,
      except: 'password',
    })
    ok('list records', json.code === 0 && json.data?.list?.length >= 1)
    const row = json.data?.list?.[0]
    ok(
      'except=password (type name) does not hide field "secret"',
      row && 'secret' in row,
      'backend except matches field name, frontend must pass field names',
    )
  }
  {
    const { json } = await req('GET', `/api/c/${COL}`, undefined, {
      page: 1,
      pageSize: 20,
      except: 'secret',
    })
    const row = json.data?.list?.[0]
    ok('except=secret hides password field', row && !('secret' in row), JSON.stringify(row))
  }
  {
    const { json } = await req('PUT', `/api/c/${COL}/${recordId}`, { title: '已更新', status: 'done' })
    ok('update record', json.code === 0, json.message)
  }
  {
    const { json } = await req('GET', `/api/c/${COL}/${recordId}`)
    ok('get record title', json.data?.title === '已更新', JSON.stringify(json.data))
  }
  {
    // 422 validation
    const { json } = await req('POST', `/api/c/${COL}`, { amount: 1 })
    ok('missing required title => 422', json.code === 422, `code=${json.code}`)
    ok('fieldErrors present', !!json.data?.fieldErrors, JSON.stringify(json.data))
  }
  {
    const { json } = await req('GET', `/api/c/${COL}/count`)
    ok('count >= 1', json.data?.count >= 1, JSON.stringify(json.data))
  }

  // 9. filter like frontend search
  console.log('\n9. Filter / sort')
  {
    const { json } = await req('GET', `/api/c/${COL}`, undefined, {
      filter: { $or: [{ title: { $like: '%更新%' } }] },
      sort: '-created_at',
      page: 1,
      pageSize: 20,
    })
    ok('filter like works', json.code === 0 && json.data?.list?.length >= 1, json.message)
  }

  // 10. indexes
  console.log('\n10. Indexes')
  {
    const { json } = await req('POST', `/api/meta/collections/${COL}/indexes`, {
      fields: ['title'],
      unique: false,
      name: 'idx_title',
    })
    ok('create index', json.code === 0, json.message)
  }
  {
    const { json } = await req('GET', `/api/meta/collections/${COL}/indexes`)
    ok('list indexes', json.code === 0 && (json.data?.list?.length || 0) >= 1)
  }
  {
    const { json } = await req('DELETE', `/api/meta/collections/${COL}/indexes`, { fields: ['title'] })
    ok('delete index', json.code === 0, json.message)
  }

  // 11. scripts
  console.log('\n11. Scripts')
  let scriptId
  {
    const { json } = await req('POST', '/api/meta/scripts', {
      collectionName: COL,
      name: 'e2e hook',
      hookPoint: 'beforeCreate',
      content: `package main\n\nimport "context"\n\nfunc BeforeCreate(ctx context.Context, data map[string]any) (map[string]any, error) {\n\treturn data, nil\n}\n`,
      enabled: true,
      priority: 0,
    })
    ok('save script', json.code === 0, json.message)
    scriptId = json.data?.id
  }
  {
    const { json } = await req('POST', '/api/meta/scripts/validate', {
      content: 'package main\nfunc Bad( {',
    })
    ok('validate invalid => valid=false', json.code === 0 && json.data?.valid === false, JSON.stringify(json.data))
  }
  if (scriptId) {
    const { json } = await req('POST', `/api/meta/scripts/${scriptId}/disable`)
    ok('disable script', json.code === 0, json.message)
    // try save with enabled false
    const saveOff = await req('POST', '/api/meta/scripts', {
      id: scriptId,
      collectionName: COL,
      name: 'e2e hook',
      hookPoint: 'beforeCreate',
      content: `package main\n\nimport "context"\n\nfunc BeforeCreate(ctx context.Context, data map[string]any) (map[string]any, error) {\n\treturn data, nil\n}\n`,
      enabled: false,
      priority: 0,
    })
    const stillOn = saveOff.json.data?.enabled === true
    ok(
      'save enabled=false is respected',
      saveOff.json.code === 0 && saveOff.json.data?.enabled === false,
      `enabled=${saveOff.json.data?.enabled}`,
    )
    // re-enable for cleanup delete
    await req('POST', '/api/meta/scripts', {
      id: scriptId,
      collectionName: COL,
      name: 'e2e hook',
      hookPoint: 'beforeCreate',
      content: `package main\n\nimport "context"\n\nfunc BeforeCreate(ctx context.Context, data map[string]any) (map[string]any, error) {\n\treturn data, nil\n}\n`,
      enabled: true,
      priority: 0,
    })
    await req('DELETE', `/api/meta/scripts/${scriptId}`)
  }

  // 12. update collection
  console.log('\n12. Update collection')
  {
    const { json } = await req('PUT', `/api/meta/collections/${COL}`, {
      displayName: 'Web E2E 改名',
      description: 'updated',
      categories: ['e2e', 'web'],
      options: { simplePagination: true },
    })
    ok('update collection', json.code === 0, json.message)
  }

  // 13. delete field
  console.log('\n13. Delete field')
  {
    const { json } = await req('DELETE', `/api/meta/collections/${COL}/fields/memo`)
    ok('delete field', json.code === 0, json.message)
  }

  // 14. delete record + collection
  console.log('\n14. Cleanup')
  {
    const { json } = await req('DELETE', `/api/c/${COL}/${recordId}`)
    ok('delete record', json.code === 0, json.message)
  }
  {
    const { json } = await req('DELETE', `/api/meta/collections/${COL}`)
    ok('delete collection', json.code === 0, json.message)
  }
  {
    const { json } = await req('GET', `/api/meta/collections/${COL}`)
    ok('get after delete fails', json.code !== 0)
  }

  // 15. frontend static
  console.log('\n15. Frontend')
  {
    const res = await fetch('http://localhost:5173/')
    const html = await res.text()
    ok('vite serves html', res.status === 200 && html.includes('root'))
  }

  console.log(`\n======== RESULT: ${passed} passed, ${failed} failed ========`)
  if (bugs.length) {
    console.log('Failed cases:')
    bugs.forEach((b) => console.log(` - ${b.name}: ${b.detail}`))
  }
  process.exit(failed > 0 ? 1 : 0)
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
