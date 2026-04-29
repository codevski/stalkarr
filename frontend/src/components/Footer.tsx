import { useEffect, useState } from "react";
import { ExternalLink } from "lucide-react";
import api from "@/lib/api";
import type { VersionInfo } from "@/types";

const RELEASES_URL = "https://github.com/codevski/sleeparr/releases/latest";

export default function Footer() {
  const [version, setVersion] = useState<VersionInfo | null>(null);

  useEffect(() => {
    api
      .get("/api/version")
      .then((res) => setVersion(res.data))
      .catch(() => {});
  }, []);

  return (
    <footer className="border-t border-border/50 py-3 mt-auto">
      <div className="max-w-6xl mx-auto px-4 flex items-center justify-between text-xs text-muted-foreground">
        <span>sleeparr</span>
        <div className="flex items-center gap-3">
          {version && <span className="font-mono">{version.current}</span>}
          {version?.hasUpdate && (
            <a
              href={RELEASES_URL}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1 text-blue-400 hover:text-blue-300 transition-colors"
            >
              <span className="relative flex h-2 w-2 mr-1">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75" />
                <span className="relative inline-flex rounded-full h-2 w-2 bg-blue-400" />
              </span>
              {version.latest} available
              <ExternalLink className="w-3 h-3 ml-1" />
            </a>
          )}
        </div>
      </div>
    </footer>
  );
}
