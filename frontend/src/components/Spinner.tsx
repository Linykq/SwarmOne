export default function Spinner() {
  return (
    <div className="flex items-center gap-2 text-zinc-300">
      <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
        <circle className="opacity-20" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"/>
        <path className="opacity-90" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"/>
      </svg>
      <span>Thinking...</span>
    </div>
  )
}
