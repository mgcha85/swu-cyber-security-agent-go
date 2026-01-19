
export async function fetchConfig() {
    const res = await fetch('/api/config');
    if (!res.ok) throw new Error('Failed to fetch config');
    return res.json();
}

export async function updateConfig(key: string, value: string) {
    const res = await fetch('/api/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ key, value }),
    });
    if (!res.ok) throw new Error('Failed to update config');
    return res.json();
}

export async function fetchReports() {
    const res = await fetch('/api/reports');
    if (!res.ok) throw new Error('Failed to fetch reports');
    return res.json();
}

export async function ingest(dir: string, collectionName: string) {
    const res = await fetch('/api/ingest', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ dir, collection_name: collectionName })
    });
    if (!res.ok) throw new Error('Failed to ingest');
    return res.json();
}

export async function superChat(query: string, image?: string) {
    const res = await fetch('/api/super/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query, image })
    });
    if (!res.ok) throw new Error('Failed to chat');
    return res.json();
}
