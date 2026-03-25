import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tv, Search, Clock, AlertCircle } from "lucide-react";
import { Link } from "react-router-dom";
import api from "@/lib/api";
import type { InstanceSummary } from "@/types";

export default function DashboardPage() {
  const [sonarr, setSonarr] = useState<InstanceSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [huntsToday, setHuntsToday] = useState(0);

  useEffect(() => {
    api
      .get("/api/dashboard")
      .then((res) => {
        setSonarr(res.data.sonarr);
        setHuntsToday(res.data.huntsToday);
      })
      .finally(() => setLoading(false));
  }, [setHuntsToday]);

  const totalMissing = sonarr.reduce(
    (acc, s) => acc + (s.missingCount ?? 0),
    0,
  );

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-xl font-semibold">Dashboard</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Overview of all configured apps
        </p>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {[
          {
            label: "Missing Episodes",
            value: loading ? "—" : totalMissing.toLocaleString(),
          },
          { label: "Missing Movies", value: "—" },
          {
            label: "Hunts Today",
            value: loading ? "—" : huntsToday.toString(),
          },
          { label: "Found Today", value: "0" },
        ].map(({ label, value }) => (
          <div key={label} className="bg-secondary/50 rounded-lg p-4">
            <p className="text-xs text-muted-foreground">{label}</p>
            <p className="text-2xl font-semibold mt-1">{value}</p>
          </div>
        ))}
      </div>

      <div>
        <h2 className="text-sm font-medium text-muted-foreground mb-3">
          Sonarr
        </h2>
        <div className="grid md:grid-cols-2 xl:grid-cols-3 gap-4">
          {loading ? (
            Array.from({ length: 2 }).map((_, i) => (
              <Card key={i} className="animate-pulse">
                <CardContent className="h-24" />
              </Card>
            ))
          ) : sonarr.length === 0 ? (
            <Card className="md:col-span-2 xl:col-span-3">
              <CardContent className="flex flex-col items-center justify-center py-10 gap-2 text-muted-foreground">
                <Tv className="w-6 h-6 opacity-30" />
                <p className="text-sm">No Sonarr instances configured</p>
                <Button variant="outline" size="sm" asChild className="mt-2">
                  <Link to="/settings">Configure in Settings</Link>
                </Button>
              </CardContent>
            </Card>
          ) : (
            sonarr.map((inst) => (
              <Card key={inst.id}>
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Tv className="w-4 h-4 text-muted-foreground" />
                      <CardTitle className="text-base">{inst.name}</CardTitle>
                      {inst.error ? (
                        <Badge variant="destructive">Unreachable</Badge>
                      ) : (
                        <Badge variant="secondary">Idle</Badge>
                      )}
                    </div>
                    <Button size="sm" asChild variant="outline">
                      <Link to={`/sonarr?instance=${inst.id}`}>
                        <Search className="w-3.5 h-3.5 mr-1.5" />
                        View
                      </Link>
                    </Button>
                  </div>
                </CardHeader>
                <CardContent>
                  {inst.error ? (
                    <div className="flex items-center gap-2 text-sm text-destructive">
                      <AlertCircle className="w-3.5 h-3.5" />
                      Could not reach this instance
                    </div>
                  ) : (
                    <>
                      <div className="flex items-center justify-between text-sm">
                        <span className="text-muted-foreground">
                          Missing episodes
                        </span>
                        <span className="font-medium tabular-nums">
                          {inst.missingCount.toLocaleString()}
                        </span>
                      </div>
                      <div className="flex items-center justify-between text-sm mt-2">
                        <span className="text-muted-foreground flex items-center gap-1.5">
                          <Clock className="w-3 h-3" /> Last hunt
                        </span>
                        <span className="text-muted-foreground">
                          {inst.lastHunt
                            ? new Date(inst.lastHunt).toLocaleTimeString([], {
                                hour: "2-digit",
                                minute: "2-digit",
                              })
                            : "Never"}
                        </span>
                      </div>
                    </>
                  )}
                </CardContent>
              </Card>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
