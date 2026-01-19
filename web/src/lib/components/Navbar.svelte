<script lang="ts">
    import { Settings, ShieldCheck } from "lucide-svelte";
    export let onOpenSettings: () => void;
</script>

<nav
    class="border-b border-gray-800 bg-gray-900/50 backdrop-blur top-0 sticky z-40"
>
    <div class="max-w-7xl mx-auto px-4 h-16 flex items-center justify-between">
        <div class="flex items-center gap-3">
            <div class="p-2 bg-cyan-500/10 rounded-lg">
                <ShieldCheck class="text-cyan-400" size={24} />
            </div>
            <h1
                class="text-xl font-bold bg-gradient-to-r from-white to-gray-400 bg-clip-text text-transparent"
            >
                Cyber Agent <span
                    class="text-cyan-400 text-xs px-2 py-0.5 bg-cyan-900/30 rounded border border-cyan-800"
                    >BETA</span
                >
            </h1>
        </div>

        <div class="flex items-center gap-4">
            <button
                on:click={onOpenSettings}
                class="p-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
                title="Settings"
            >
                <Settings size={20} />
            </button>
        </div>
    </div>
</nav>
