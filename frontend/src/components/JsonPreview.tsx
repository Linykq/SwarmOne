export default function JsonPreview({ value }: { value: any }) { return <pre className='code whitespace-pre-wrap break-words'>{JSON.stringify(value, null, 2)}</pre> }
