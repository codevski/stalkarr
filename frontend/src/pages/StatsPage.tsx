import { BarChart2, TrendingUp, CheckCircle2, Clock } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export default function StatsPage() {
  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-xl font-semibold">Stats</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Hunt history and performance
        </p>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {[
          { label: "Total Hunts", value: "0", icon: BarChart2 },
          { label: "Episodes Found", value: "0", icon: CheckCircle2 },
          { label: "Success Rate", value: "—", icon: TrendingUp },
          { label: "Avg Hunt Time", value: "—", icon: Clock },
        ].map(({ label, value, icon: Icon }) => (
          <div key={label} className="bg-secondary/50 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <Icon className="w-3.5 h-3.5 text-muted-foreground" />
              <p className="text-xs text-muted-foreground">{label}</p>
            </div>
            <p className="text-2xl font-semibold">{value}</p>
          </div>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Hunt History</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center justify-center py-12 text-muted-foreground gap-2">
            <BarChart2 className="w-8 h-8 opacity-30" />
            <p className="text-sm">No hunt history yet</p>
            <p className="text-xs">
              Stats will appear here once hunting begins
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
