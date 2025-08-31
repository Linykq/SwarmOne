import { useMemo, useState } from 'react'
import JsonPreview from '../components/JsonPreview'
import Spinner from '../components/Spinner'
import { askSwarm, type AskResponse } from '../lib/api'

type FormState = {
  task: string
  content: string
  expections: string
  source: string
  language: string
}

export default function Home() {
  const [form, setForm] = useState<FormState>({
    task: 'reply email',
    content: '',
    expections: 'Professional; concise',
    source: '',
    language: 'en-US',
  })
  const [templateId, setTemplateId] = useState<string>('task.reply.email.v1')
  const [answer, setAnswer] = useState<string>('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string>('')
  const [meta, setMeta] = useState<AskResponse | null>(null)

  const json = useMemo(() => ({
    Task: form.task,
    Content: form.content,
    Expections: form.expections,
    Source: form.source,
    Language: form.language,
  }), [form])

  const instruction = useMemo(() => {
    return `finish the task as following\n${JSON.stringify(json)}`
  }, [json])

  const onChange = (k: keyof FormState) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    setForm(s => ({ ...s, [k]: e.target.value }))
  }

  const onSubmit = async () => {
    setPending(true); setError(''); setAnswer(''); setMeta(null)
    try {
      const res = await askSwarm(templateId, instruction)
      setAnswer(res.answer)
      setMeta(res)
    } catch (e: any) {
      setError(e?.message || 'Request failed')
    } finally {
      setPending(false)
    }
  }

  // Derive scores for display:
  // - Prefer backend `scores` if provided (e.g., judge model outputs per-runner score).
  // - Otherwise fallback to `votes_per_candidate` (for compatibility with majority mode).
  const viewScores = useMemo(() => {
    if (!meta) return { items: [] as { idx: number; label: string; value: string }[] }
    const n = meta.runners || 0
    const scores = Array.isArray(meta.scores) ? meta.scores : []
    const included = new Set(meta.included_indices || [])
    const items: { idx: number; label: string; value: string }[] = []
    for (let i = 0; i < n; i++) {
      let val: string
      if (!included.has(i)) {
        val = 'N/A' // didn’t participate (empty or blocked)
      } else if (Number.isFinite(scores[i])) {
        val = Number(scores[i]).toFixed(4) // format to 4 decimals
      } else {
        // Participated but judge didn't provide a score (shouldn't happen if backend is correct)
        val = '0.0000'
      }
      items.push({ idx: i, label: `Runner #${i}`, value: val })
    }
    return { items }
  }, [meta])

  return (
    <div className="space-y-8">
      {/* Hero */}
      <section className="text-center py-4">
        <div className="inline-flex items-center gap-2 badge mb-3">
          <span className="w-2 h-2 rounded-full bg-emerald-400"></span>
          Swarm consensus runner (judge mode)
        </div>
        <h1 className="text-3xl md:text-4xl font-semibold tracking-tight">
          Build tasks, preview JSON, <span className="text-brand-400">ask the swarm</span>.
        </h1>
        <p className="text-zinc-300 mt-2">Single-round arbitration with a strong model. Frontend only. </p>
      </section>

      <section className="grid lg:grid-cols-2 gap-6">
        {/* Left: Form */}
        <div className="glass p-5 space-y-4 shadow-soft">
          <h2 className="text-lg font-semibold">Task Builder</h2>
          <div className="grid gap-3">
            <div className="field">
              <label className="text-sm text-zinc-300">Task</label>
              <input className="input" placeholder="e.g. reply email" value={form.task} onChange={onChange('task')} />
            </div>
            <div className="field">
              <label className="text-sm text-zinc-300">Content</label>
              <textarea className="textarea" placeholder="What should the AI work on..." value={form.content} onChange={onChange('content')} />
            </div>
            <div className="field">
              <label className="text-sm text-zinc-300">Expections</label>
              <input className="input" placeholder="Tone, style, length..." value={form.expections} onChange={onChange('expections')} />
            </div>
            <div className="field">
              <label className="text-sm text-zinc-300">Source</label>
              <textarea className="textarea" placeholder="Any references / URLs / notes" value={form.source} onChange={onChange('source')} />
            </div>
            <div className="field">
              <label className="text-sm text-zinc-300">Language</label>
              <input className="input" placeholder="e.g. en-US / zh-CN" value={form.language} onChange={onChange('language')} />
            </div>
            <div className="field">
              <label className="text-sm text-zinc-300">Template ID (optional)</label>
              <input className="input" value={templateId} onChange={(e)=>setTemplateId(e.target.value)} placeholder="task.reply.email.v1" />
            </div>
          </div>

          <div className="flex items-center gap-3">
            <button className="btn-primary" onClick={onSubmit} disabled={pending}>
              {pending ? <Spinner /> : 'Ask Swarm'}
            </button>
            {error && <span className="text-sm text-red-300">{error}</span>}
          </div>
        </div>

        {/* Right: Preview + Answer (sticky on lg) */}
        <div className="space-y-6 lg:sticky lg:top-24 h-fit">
          <div className="glass p-5 shadow-soft">
            <div className="flex items-center justify-between mb-2">
              <h3 className="font-semibold">Preview</h3>
              <span className="badge">instruction</span>
            </div>
            <div className="code overflow-auto">{instruction}</div>
            <div className="mt-3">
              <span className="badge">payload JSON</span>
            </div>
            <div className="mt-2 overflow-auto">
              <JsonPreview value={json} />
            </div>
          </div>

          {/* Answer: scrollable area */}
          <div className="glass p-5 shadow-soft">
            <div className="flex items-center justify-between mb-3">
              <h3 className="font-semibold">Answer</h3>
              {pending && <Spinner />}
            </div>
            {/* Scrollable: max height + vertical scroll */}
            <div className="min-h-[150px] max-h-[420px] overflow-y-auto whitespace-pre-wrap text-zinc-100 pr-2">
              {answer ? answer : <span className="text-zinc-400">No answer yet.</span>}
            </div>

            {/* Judge meta: winner + per-runner scores */}
            {meta && (
              <div className="mt-5 space-y-3">
                <div className="flex items-center gap-3">
                  <span className="badge">winner (judge)</span>
                  <span className="text-sm text-zinc-200">Runner #{meta.winner_index}</span>
                </div>

                <div className="space-y-2">
                  <div className="badge">runner scores</div>
                  <ul className="mt-1 space-y-1">
                    {viewScores.items.map(it => (
                      <li key={it.idx} className="flex items-center justify-between text-sm text-zinc-200">
                        <span className="text-zinc-300">{it.label}</span>
                        <span className={it.value === 'N/A' ? 'text-zinc-500' : 'font-medium'}>
                          {it.value}
                        </span>
                      </li>
                    ))}
                  </ul>
                </div>

                <div className="text-xs text-zinc-400">consensus_id: {meta.consensus_id}</div>
              </div>
            )}
          </div>
        </div>
      </section>
    </div>
  )
}
