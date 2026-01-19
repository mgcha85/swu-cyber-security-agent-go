<script lang="ts">
  import Navbar from "./lib/components/Navbar.svelte";
  import SettingsModal from "./lib/components/SettingsModal.svelte";
  import ReportViewer from "./lib/components/ReportViewer.svelte";
  import { superChat, fetchReports } from "./lib/api";
  import {
    Loader2,
    Upload,
    Search,
    FileText,
    BrainCircuit,
    History,
  } from "lucide-svelte";

  // State
  let showSettings = false;
  let activeTab: "dashboard" | "history" = "dashboard";

  interface Result {
    final_report: string;
    agent_reports: Record<string, string>;
    vlm_analysis?: string;
    gnn_data?: string;
  }

  // Dashboard State
  let query = "";
  let analyzing = false;
  let result: Result | null = null;
  let selectedFile: File | null = null;
  let selectedFilePreview: string | null = null;

  // History State
  let reports: any[] = [];
  let loadingHistory = false;

  function handleFileSelect(event: Event) {
    const target = event.target as HTMLInputElement;
    if (target.files && target.files.length > 0) {
      selectedFile = target.files[0];
      const reader = new FileReader();
      reader.onload = (e) => (selectedFilePreview = e.target?.result as string);
      reader.readAsDataURL(selectedFile);
    }
  }

  async function handleAnalyze() {
    if (!query) return;
    analyzing = true;
    result = null;

    // Convert image to base64 if selected
    let imageBase64 = "";
    if (selectedFilePreview) {
      // Remove prefix "data:image/png;base64,"
      imageBase64 = selectedFilePreview.split(",")[1];
    }

    try {
      result = await superChat(query, imageBase64);
    } catch (e) {
      alert("Analysis failed: " + e);
    } finally {
      analyzing = false;
    }
  }

  async function loadHistory() {
    loadingHistory = true;
    try {
      reports = await fetchReports();
    } catch (e) {
      console.error(e);
    } finally {
      loadingHistory = false;
    }
  }

  // Reload history when tab changes
  $: if (activeTab === "history") loadHistory();
</script>

<div
  class="min-h-screen bg-gray-950 text-gray-200 font-sans selection:bg-cyan-500/30"
>
  <Navbar onOpenSettings={() => (showSettings = true)} />

  <SettingsModal show={showSettings} onClose={() => (showSettings = false)} />

  <main class="max-w-7xl mx-auto px-4 py-8">
    <!-- Tabs -->
    <div class="flex gap-4 mb-8 border-b border-gray-800">
      <button
        class="px-4 py-3 flex items-center gap-2 border-b-2 transition-colors {activeTab ===
        'dashboard'
          ? 'border-cyan-500 text-cyan-400'
          : 'border-transparent text-gray-400 hover:text-white'}"
        on:click={() => (activeTab = "dashboard")}
      >
        <BrainCircuit size={18} /> Dashboard
      </button>
      <button
        class="px-4 py-3 flex items-center gap-2 border-b-2 transition-colors {activeTab ===
        'history'
          ? 'border-cyan-500 text-cyan-400'
          : 'border-transparent text-gray-400 hover:text-white'}"
        on:click={() => (activeTab = "history")}
      >
        <History size={18} /> History
      </button>
    </div>

    {#if activeTab === "dashboard"}
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <!-- Input Panel -->
        <div class="lg:col-span-1 space-y-6">
          <div
            class="bg-gray-900 rounded-xl p-6 border border-gray-800 shadow-xl"
          >
            <h2
              class="text-lg font-semibold text-white mb-4 flex items-center gap-2"
            >
              <Search size={20} class="text-cyan-400" /> New Analysis
            </h2>

            <div class="space-y-4">
              <div>
                <label class="block text-sm font-medium text-gray-400 mb-1"
                  >Query / Threat Name</label
                >
                <textarea
                  bind:value={query}
                  class="w-full bg-gray-950 border border-gray-700 rounded-lg p-3 text-white focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none h-32 resize-none"
                  placeholder="e.g., Analyze the recent Account Hijacking trends in the energy sector..."
                ></textarea>
              </div>

              <div>
                <label class="block text-sm font-medium text-gray-400 mb-1"
                  >Upload GNN Graph (Optional)</label
                >
                <div
                  class="border-2 border-dashed border-gray-700 rounded-lg p-4 text-center hover:bg-gray-800/50 transition-colors relative cursor-pointer group"
                >
                  <input
                    type="file"
                    accept="image/*"
                    class="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
                    on:change={handleFileSelect}
                  />
                  <div
                    class="flex flex-col items-center gap-2 text-gray-500 group-hover:text-cyan-400"
                  >
                    {#if selectedFilePreview}
                      <img
                        src={selectedFilePreview}
                        alt="Preview"
                        class="h-24 object-contain rounded"
                      />
                      <span class="text-xs">{selectedFile?.name}</span>
                    {:else}
                      <Upload size={24} />
                      <span class="text-xs">Click or drag image</span>
                    {/if}
                  </div>
                </div>
              </div>

              <button
                on:click={handleAnalyze}
                disabled={analyzing || !query}
                class="w-full py-3 bg-cyan-600 hover:bg-cyan-500 disabled:bg-gray-800 disabled:text-gray-500 disabled:cursor-not-allowed rounded-lg text-white font-bold flex items-center justify-center gap-2 transition-all shadow-lg hover:shadow-cyan-500/20"
              >
                {#if analyzing}
                  <Loader2 class="animate-spin" size={20} /> Analyzing...
                {:else}
                  <BrainCircuit size={20} /> Start Analysis
                {/if}
              </button>
            </div>
          </div>
        </div>

        <!-- Output Panel -->
        <div class="lg:col-span-2">
          {#if analyzing}
            <div
              class="bg-gray-900/50 rounded-xl p-12 border border-gray-800 flex flex-col items-center justify-center text-center animate-pulse"
            >
              <Loader2 class="text-cyan-500 w-12 h-12 animate-spin mb-4" />
              <h3 class="text-xl font-medium text-white">
                Synthesizing Intelligence...
              </h3>
              <p class="text-gray-400 mt-2">
                Agents are cross-referencing Knowledge Bases and GNN
                predictions.
              </p>
            </div>
          {:else if result}
            <div class="space-y-6">
              <!-- Final Report Card -->
              <div
                class="bg-gray-900 rounded-xl border border-gray-700 shadow-2xl overflow-hidden"
              >
                <div
                  class="bg-gradient-to-r from-gray-800 to-gray-900 p-4 border-b border-gray-700 flex items-center justify-between"
                >
                  <h2
                    class="text-lg font-bold text-white flex items-center gap-2"
                  >
                    <FileText class="text-cyan-400" size={20} /> Comprehensive Report
                  </h2>
                  <span
                    class="text-xs text-gray-400 bg-gray-800 px-2 py-1 rounded"
                    >Generated Just Now</span
                  >
                </div>
                <div class="p-6 bg-gray-900/90">
                  <ReportViewer markdown={result.final_report} />
                </div>
              </div>

              <!-- Individual Agent Findings (Collapsible) -->
              <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                {#each Object.entries(result.agent_reports || {}) as [agentName, report]}
                  <div
                    class="bg-gray-900 rounded-lg border border-gray-800 p-4 hover:border-gray-600 transition-colors"
                  >
                    <h4
                      class="text-sm font-bold text-cyan-300 mb-2 uppercase tracking-wider"
                    >
                      {agentName}
                    </h4>
                    <div
                      class="text-sm text-gray-400 line-clamp-4 hover:line-clamp-none transition-all cursor-pointer"
                    >
                      {report}
                    </div>
                  </div>
                {/each}
              </div>

              <!-- GNN & VLM Data -->
              <div class="bg-gray-900 rounded-lg border border-gray-800 p-4">
                <h4 class="text-sm font-bold text-green-400 mb-2">
                  GNN & VLM Insights
                </h4>
                <p class="text-sm text-gray-300 mb-2">
                  <span class="text-gray-500">GNN Context:</span>
                  {result.gnn_data}
                </p>
                {#if result.vlm_analysis}
                  <div
                    class="text-sm text-gray-300 p-3 bg-gray-800/50 rounded border border-gray-700"
                  >
                    <span class="text-gray-500 block mb-1">VLM Analysis:</span>
                    {result.vlm_analysis}
                  </div>
                {/if}
              </div>
            </div>
          {:else}
            <div
              class="bg-gray-900/30 rounded-xl p-12 border border-dashed border-gray-800 flex flex-col items-center justify-center text-center"
            >
              <div
                class="w-16 h-16 bg-gray-800 rounded-full flex items-center justify-center mb-4"
              >
                <BrainCircuit class="text-gray-600" size={32} />
              </div>
              <h3 class="text-xl font-medium text-gray-500">
                Ready to Analyze
              </h3>
              <p class="text-gray-600 mt-2 max-w-sm">
                Enter a threat query or CVE ID to task the agent swarm.
              </p>
            </div>
          {/if}
        </div>
      </div>
    {/if}

    {#if activeTab === "history"}
      <div
        class="bg-gray-900 rounded-xl border border-gray-800 shadow-xl overflow-hidden"
      >
        <div class="overflow-x-auto">
          <table class="w-full text-left text-sm">
            <thead class="bg-gray-800 text-gray-400 uppercase tracking-wider">
              <tr>
                <th class="px-6 py-4 font-medium">Date</th>
                <th class="px-6 py-4 font-medium">Threat Name</th>
                <th class="px-6 py-4 font-medium">File</th>
                <th class="px-6 py-4 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-800">
              {#if loadingHistory}
                <tr
                  ><td colspan="4" class="px-6 py-8 text-center text-gray-500"
                    >Loading history...</td
                  ></tr
                >
              {:else if reports.length === 0}
                <tr
                  ><td colspan="4" class="px-6 py-8 text-center text-gray-500"
                    >No reports found.</td
                  ></tr
                >
              {:else}
                {#each reports as report}
                  <tr class="hover:bg-gray-800/50 transition-colors">
                    <td class="px-6 py-4 text-gray-300 whitespace-nowrap">
                      {new Date(report.created_at).toLocaleString()}
                    </td>
                    <td class="px-6 py-4 font-medium text-white"
                      >{report.threat_name}</td
                    >
                    <td class="px-6 py-4 text-gray-400">{report.filename}</td>
                    <td class="px-6 py-4 text-right">
                      <button
                        class="text-cyan-400 hover:text-cyan-300 font-medium"
                        >View</button
                      >
                    </td>
                  </tr>
                {/each}
              {/if}
            </tbody>
          </table>
        </div>
      </div>
    {/if}
  </main>
</div>
