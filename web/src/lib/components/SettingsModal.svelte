<script lang="ts">
    import { fetchConfig, updateConfig } from "../api";
    import { onMount } from "svelte";
    import { X, Save, Loader2 } from "lucide-svelte";

    export let show = false;
    export let onClose: () => void;

    let configs: Record<string, string> = {};
    let loading = true;
    let saving = false;

    async function load() {
        loading = true;
        try {
            configs = await fetchConfig();
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    $: if (show) load();

    async function save(key: string, value: string) {
        saving = true;
        try {
            await updateConfig(key, value);
            await load(); // Refresh
        } catch (e) {
            alert("Failed to save");
        } finally {
            saving = false;
        }
    }

    // Predefined keys to show if missing
    const defaultKeys = ["GOOGLE_API_KEY", "GOOGLE_CX", "OPENAI_API_KEY"];
</script>

{#if show}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm"
    >
        <div
            class="w-full max-w-2xl bg-gray-900 border border-gray-700 rounded-xl shadow-2xl p-6 relative"
        >
            <button
                on:click={onClose}
                class="absolute top-4 right-4 text-gray-400 hover:text-white"
            >
                <X size={24} />
            </button>

            <h2
                class="text-2xl font-bold mb-6 text-white flex items-center gap-2"
            >
                Configuration
            </h2>

            {#if loading}
                <div class="flex justify-center p-8">
                    <Loader2 class="animate-spin text-cyan-400" size={32} />
                </div>
            {:else}
                <div class="space-y-4 max-h-[60vh] overflow-y-auto pr-2">
                    <!-- Render Default Environment Variables first -->
                    {#each defaultKeys as key}
                        <div class="bg-gray-800 p-4 rounded-lg">
                            <label
                                class="block text-sm font-medium text-gray-400 mb-1"
                                >{key}</label
                            >
                            <div class="flex gap-2">
                                <input
                                    type="text"
                                    value={configs[key] || ""}
                                    class="flex-1 bg-gray-950 border border-gray-700 rounded px-3 py-2 text-white font-mono focus:border-cyan-500 outline-none"
                                    placeholder="Value..."
                                    on:change={(e) =>
                                        save(key, e.currentTarget.value)}
                                />
                            </div>
                        </div>
                    {/each}

                    <!-- Render others -->
                    {#each Object.entries(configs) as [key, value]}
                        {#if !defaultKeys.includes(key)}
                            <div class="bg-gray-800 p-4 rounded-lg">
                                <label
                                    class="block text-sm font-medium text-gray-400 mb-1"
                                    >{key}</label
                                >
                                <div class="flex gap-2">
                                    <input
                                        type="text"
                                        {value}
                                        class="flex-1 bg-gray-950 border border-gray-700 rounded px-3 py-2 text-white font-mono focus:border-cyan-500 outline-none"
                                        on:change={(e) =>
                                            save(key, e.currentTarget.value)}
                                    />
                                </div>
                            </div>
                        {/if}
                    {/each}
                </div>
            {/if}

            <div class="mt-6 text-right">
                <button
                    on:click={onClose}
                    class="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-white font-medium"
                >
                    Close
                </button>
            </div>
        </div>
    </div>
{/if}
