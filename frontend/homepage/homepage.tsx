import { AppRunList } from "@/apprunlist/apprunlist";
import { cn } from "@/util/util";
import { Activity, BarChart2, List, Search } from "lucide-react";

// Feature card component for the homepage
interface FeatureCardProps {
    icon: React.ReactNode;
    title: string;
    description: string;
}

const FeatureCard: React.FC<FeatureCardProps> = ({ icon, title, description }) => {
    return (
        <div className="bg-panel border border-border rounded-lg p-6 flex flex-col items-center text-center">
            <div className="text-accent mb-4">{icon}</div>
            <h3 className="text-primary text-lg font-medium mb-2">{title}</h3>
            <p className="text-secondary text-sm">{description}</p>
        </div>
    );
};

export const HomePage: React.FC = () => {
    return (
        <div className="h-screen bg-background flex flex-col">
            {/* Header */}
            <header className="bg-panel border-b border-border p-4 flex items-center justify-between">
                <div className="flex items-center">
                    <img src="/outriglogo.svg" alt="Outrig Logo" className="w-8 h-8 mr-3" />
                    <h1 className="text-primary text-xl font-medium">Outrig</h1>
                </div>
            </header>

            {/* Main content */}
            <main className="flex-grow flex flex-col md:flex-row overflow-hidden">
                {/* Left side: Introduction and features */}
                <div className="w-full md:w-1/2 p-8 flex flex-col">
                    {/* Introduction */}
                    <div className="mb-8">
                        <h2 className="text-primary text-2xl font-medium mb-4">Real-time debugging for Go programs</h2>
                        <p className="text-secondary mb-4">
                            Outrig provides powerful debugging capabilities similar to Chrome DevTools, but designed
                            specifically for Go applications.
                        </p>
                        <p className="text-secondary">
                            Monitor your Go applications in real-time with just one line of code integration. Get
                            instant insights into logs, goroutines, and runtime statistics.
                        </p>
                    </div>

                    {/* Features */}
                    <h3 className="text-primary text-xl font-medium mb-4">Key Features</h3>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                        <FeatureCard
                            icon={<Search size={24} />}
                            title="Log Searching"
                            description="Quickly search and filter through application logs with powerful search capabilities."
                        />
                        <FeatureCard
                            icon={<Activity size={24} />}
                            title="Goroutine Monitoring"
                            description="Track and analyze goroutines to identify bottlenecks and deadlocks."
                        />
                        <FeatureCard
                            icon={<BarChart2 size={24} />}
                            title="Runtime Statistics"
                            description="Monitor memory usage, GC cycles, and other runtime metrics in real-time."
                        />
                        <FeatureCard
                            icon={<List size={24} />}
                            title="Variable Watching"
                            description="Watch variables and execute runtime hooks to debug your application."
                        />
                    </div>
                </div>

                {/* Right side: App run selection */}
                <div
                    className={cn(
                        "w-full md:w-1/2 border-t md:border-t-0 md:border-l border-border",
                        "flex flex-col h-full overflow-hidden"
                    )}
                >
                    <div className="p-6 bg-panel border-b border-border">
                        <h2 className="text-primary text-xl font-medium">Select an Application Run</h2>
                        <p className="text-secondary text-sm mt-1">
                            Choose a running or completed Go application to start debugging.
                        </p>
                    </div>
                    <div className="flex-grow overflow-auto">
                        <AppRunList />
                    </div>
                </div>
            </main>

            {/* Footer */}
            <footer className="bg-panel border-t border-border p-4 text-center text-secondary text-sm">
                <p>Outrig - Powerful debugging for Go applications</p>
            </footer>
        </div>
    );
};
