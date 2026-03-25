import { useEffect, useState, useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Search,
  X,
  ChevronLeft,
  ChevronRight,
  Loader2,
  Tv,
  AlertCircle,
} from "lucide-react";
import { Link } from "react-router-dom";
import { cn } from "@/lib/utils";
import api from "@/lib/api";
import type { MissingResult, Episode, SonarrInstance } from "@/types";
import { toast } from "sonner";
import axios from "axios";

export default function SonarrPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [instances, setInstances] = useState<SonarrInstance[]>([]);
  const [activeId, setActiveId] = useState<string>("");
  const [data, setData] = useState<MissingResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [stalking, setStalking] = useState(false);
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const instanceParam = searchParams.get("instance");
    api.get("/api/settings").then((res) => {
      const insts: SonarrInstance[] = res.data.sonarr;
      setInstances(insts);
      if (insts.length > 0) {
        const initial =
          instanceParam && insts.find((i) => i.id === instanceParam)
            ? instanceParam
            : insts[0].id;
        setActiveId(initial);
      }
    });
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const t = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 500);
    return () => clearTimeout(t);
  }, [search]);

  const fetchMissing = useCallback(async () => {
    if (!activeId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await api.get(`/api/sonarr/${activeId}/missing`, {
        params: { page, pageSize, search: debouncedSearch || undefined },
      });
      setData(res.data);
      setSelected(new Set());
    } catch {
      setError(
        "Could not reach this Sonarr instance - check your URL and API key in Settings",
      );
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [activeId, page, pageSize, debouncedSearch]);

  useEffect(() => {
    fetchMissing();
  }, [fetchMissing]);

  function switchInstance(id: string) {
    setActiveId(id);
    setPage(1);
    setSearch("");
    setDebouncedSearch("");
    setData(null);
    setSelected(new Set());
    setError(null);
    setSearchParams({ instance: id });
  }

  const episodes = data?.episodes ?? [];
  const totalPages = data ? Math.ceil(data.totalCount / pageSize) : 0;
  const activeInstance = instances.find((i) => i.id === activeId);

  function toggleSelect(id: number) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }

  function toggleSelectAll() {
    if (selected.size === episodes.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(episodes.map((e) => e.id)));
    }
  }

  async function stalk(ids?: number[]) {
    const episodeIds = ids ?? episodes.map((e) => e.id);
    if (episodeIds.length === 0) return;

    setStalking(true);
    try {
      const res = await api.post(`/api/sonarr/${activeId}/stalk`, {
        episodeIds,
      });
      toast.success("Stalk triggered", {
        description: res.data.message,
      });
    } catch (err) {
      toast.error("Hunt failed", {
        description: axios.isAxiosError(err)
          ? (err.response?.data?.error ?? "Could not reach server")
          : "Could not reach server",
      });
    } finally {
      setStalking(false);
    }
  }

  async function stalkAll() {
    setStalking(true);
    try {
      const res = await api.post(`/api/sonarr/${activeId}/stalk/all`);
      toast.success("Stalk All triggered", {
        description: res.data.message,
      });
    } catch (err) {
      toast.error("Hunt failed", {
        description: axios.isAxiosError(err)
          ? (err.response?.data?.error ?? "Could not reach server")
          : "Could not reach server",
      });
    } finally {
      setStalking(false);
    }
  }

  function formatEpisode(ep: Episode) {
    const s = String(ep.seasonNumber).padStart(2, "0");
    const e = String(ep.episodeNumber).padStart(2, "0");
    return `S${s}E${e}`;
  }

  if (instances.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-24 gap-3 text-muted-foreground">
        <Tv className="w-8 h-8 opacity-30" />
        <p className="text-sm">No Sonarr instances configured</p>
        <Button variant="outline" size="sm" asChild>
          <Link to="/settings">Configure in Settings</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-xl font-semibold">Sonarr</h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            {data
              ? `${data.totalCount.toLocaleString()} missing episodes`
              : (activeInstance?.name ?? "")}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {selected.size > 0 && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => stalk([...selected])}
              disabled={stalking}
            >
              {stalking ? (
                <Loader2 className="w-3.5 h-3.5 mr-1.5 animate-spin" />
              ) : (
                <Search className="w-3.5 h-3.5 mr-1.5" />
              )}
              Stalk {selected.size} selected
            </Button>
          )}
          <Button size="sm" onClick={stalkAll} disabled={stalking}>
            {stalking ? (
              <Loader2 className="w-3.5 h-3.5 mr-1.5 animate-spin" />
            ) : (
              <Search className="w-3.5 h-3.5 mr-1.5" />
            )}
            Stalk All
          </Button>
        </div>
      </div>

      {instances.length > 1 && (
        <>
          <div className="hidden md:flex items-center gap-1 border-b border-border/50 pb-0">
            {instances.map((inst) => (
              <button
                key={inst.id}
                onClick={() => switchInstance(inst.id)}
                className={cn(
                  "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
                  activeId === inst.id
                    ? "border-foreground text-foreground"
                    : "border-transparent text-muted-foreground hover:text-foreground",
                )}
              >
                {inst.name}
              </button>
            ))}
          </div>

          <div className="md:hidden">
            <Select value={activeId} onValueChange={switchInstance}>
              <SelectTrigger className="h-9">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {instances.map((inst) => (
                  <SelectItem key={inst.id} value={inst.id}>
                    {inst.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </>
      )}

      <div className="flex items-center gap-2 flex-wrap">
        <div className="relative flex-1 min-w-50">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground" />
          <Input
            placeholder="Filter by series or episode title..."
            className="pl-9 h-9"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          {search && (
            <button
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              onClick={() => setSearch("")}
            >
              <X className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
        <Select
          value={String(pageSize)}
          onValueChange={(v) => {
            setPageSize(Number(v));
            setPage(1);
          }}
        >
          <SelectTrigger className="w-27.5 h-9">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="20">20 / page</SelectItem>
            <SelectItem value="50">50 / page</SelectItem>
            <SelectItem value="100">100 / page</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <Card className="hidden md:block">
        <CardContent className="p-0">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border/50">
                <th className="w-10 p-3 text-left">
                  <input
                    type="checkbox"
                    className="rounded"
                    checked={
                      selected.size === episodes.length && episodes.length > 0
                    }
                    onChange={toggleSelectAll}
                  />
                </th>
                <th className="p-3 text-left font-medium text-muted-foreground">
                  Series
                </th>
                <th className="p-3 text-left font-medium text-muted-foreground w-24">
                  Episode
                </th>
                <th className="p-3 text-left font-medium text-muted-foreground">
                  Title
                </th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={4} className="p-8 text-center">
                    <Loader2 className="w-5 h-5 animate-spin mx-auto text-muted-foreground" />
                  </td>
                </tr>
              ) : error ? (
                <tr>
                  <td colSpan={4} className="p-8 text-center">
                    <div className="flex flex-col items-center gap-2 text-muted-foreground">
                      <AlertCircle className="w-5 h-5 text-destructive" />
                      <p className="text-sm text-destructive">{error}</p>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={fetchMissing}
                      >
                        Retry
                      </Button>
                    </div>
                  </td>
                </tr>
              ) : episodes.length === 0 ? (
                <tr>
                  <td
                    colSpan={4}
                    className="p-8 text-center text-muted-foreground text-sm"
                  >
                    No missing episodes found
                  </td>
                </tr>
              ) : (
                episodes.map((ep) => (
                  <tr
                    key={ep.id}
                    className={cn(
                      "border-b border-border/30 last:border-0 transition-colors cursor-pointer",
                      selected.has(ep.id)
                        ? "bg-primary/5"
                        : "hover:bg-secondary/50",
                    )}
                    onClick={() => toggleSelect(ep.id)}
                  >
                    <td className="p-3">
                      <input
                        type="checkbox"
                        className="rounded"
                        checked={selected.has(ep.id)}
                        onChange={() => toggleSelect(ep.id)}
                        onClick={(e) => e.stopPropagation()}
                      />
                    </td>
                    <td className="p-3 font-medium">{ep.seriesTitle}</td>
                    <td className="p-3">
                      <Badge variant="outline" className="font-mono text-xs">
                        {formatEpisode(ep)}
                      </Badge>
                    </td>
                    <td className="p-3 text-muted-foreground">{ep.title}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </CardContent>
      </Card>

      <div className="md:hidden flex flex-col gap-2">
        {loading ? (
          <div className="flex justify-center p-8">
            <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
          </div>
        ) : error ? (
          <div className="flex flex-col items-center gap-2 p-8 text-center">
            <AlertCircle className="w-5 h-5 text-destructive" />
            <p className="text-sm text-destructive">{error}</p>
            <Button variant="outline" size="sm" onClick={fetchMissing}>
              Retry
            </Button>
          </div>
        ) : episodes.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center p-8">
            No missing episodes found
          </p>
        ) : (
          episodes.map((ep) => (
            <div
              key={ep.id}
              className={cn(
                "rounded-lg border border-border/50 p-3 flex items-center gap-3 cursor-pointer transition-colors",
                selected.has(ep.id)
                  ? "bg-primary/5 border-primary/30"
                  : "bg-card",
              )}
              onClick={() => toggleSelect(ep.id)}
            >
              <input
                type="checkbox"
                className="rounded shrink-0"
                checked={selected.has(ep.id)}
                onChange={() => toggleSelect(ep.id)}
                onClick={(e) => e.stopPropagation()}
              />
              <div className="flex-1 min-w-0">
                <p className="font-medium text-sm truncate">{ep.seriesTitle}</p>
                <p className="text-xs text-muted-foreground truncate mt-0.5">
                  {ep.title}
                </p>
              </div>
              <Badge variant="outline" className="font-mono text-xs shrink-0">
                {formatEpisode(ep)}
              </Badge>
            </div>
          ))
        )}
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">
            Page {page} of {totalPages.toLocaleString()}
          </span>
          <div className="flex items-center gap-1">
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1 || loading}
            >
              <ChevronLeft className="w-4 h-4" />
            </Button>
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page === totalPages || loading}
            >
              <ChevronRight className="w-4 h-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
