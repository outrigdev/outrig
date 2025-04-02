import { cn } from "@/util/util";
import React, { useEffect, useRef } from "react";

export class SparklineModel {
    data: number[] = [];
    maxPoints = 600;
    canvas: HTMLCanvasElement | null = null;
    ctx: CanvasRenderingContext2D | null = null;

    // Configuration properties
    lineColor = "#3b82f6"; // default blue
    lineWidth = 1.5;
    fillColor: string | null = null;
    
    // Cached min/max values
    cachedMin: number | null = null;
    cachedMax: number | null = null;
    
    constructor(options?: { 
        lineColor?: string; 
        lineWidth?: number; 
        fillColor?: string | null;
        data?: number[];
    }) {
        if (options) {
            this.configure({
                lineColor: options.lineColor,
                lineWidth: options.lineWidth,
                fillColor: options.fillColor,
            });
            
            if (options.data) {
                this.redraw(options.data);
            }
        }
    }

    // Get a copy of the current data array
    getData(): number[] {
        return [...this.data];
    }

    // Configure multiple options at once
    configure(options: { lineColor?: string; lineWidth?: number; fillColor?: string | null }): void {
        if (options.lineColor !== undefined) this.lineColor = options.lineColor;
        if (options.lineWidth !== undefined) this.lineWidth = options.lineWidth;
        if (options.fillColor !== undefined) this.fillColor = options.fillColor;
        this.redraw();
    }

    // Update cached min/max values
    updateMinMax(): void {
        if (this.data.length === 0) {
            this.cachedMin = null;
            this.cachedMax = null;
            return;
        }
        
        this.cachedMin = Math.min(...this.data);
        this.cachedMax = Math.max(...this.data);
    }
    
    pushSample(sample: number): void {
        // Check if this sample would change the scale using cached values
        let scaleWillChange = false;
        
        if (this.cachedMin === null || this.cachedMax === null) {
            // First point, no scale change
            this.cachedMin = sample;
            this.cachedMax = sample;
        } else {
            // Check if new point extends the range
            scaleWillChange = sample < this.cachedMin || sample > this.cachedMax;
            
            // Update cached values if needed
            if (sample < this.cachedMin) this.cachedMin = sample;
            if (sample > this.cachedMax) this.cachedMax = sample;
        }
        
        // Add the sample to the data array
        this.data.push(sample);
        
        // Keep only the last maxPoints
        if (this.data.length > this.maxPoints) {
            // If we're removing points, we might need to recalculate min/max
            // if one of the removed points was a min or max value
            const removedPoints = this.data.slice(0, this.data.length - this.maxPoints);
            const containsMin = removedPoints.some(p => p === this.cachedMin);
            const containsMax = removedPoints.some(p => p === this.cachedMax);
            
            this.data = this.data.slice(-this.maxPoints);
            
            // Recalculate min/max if needed
            if (containsMin || containsMax) {
                this.updateMinMax();
                scaleWillChange = true; // Force full redraw when scale changes
            }
        }
        
        // Use incremental redraw if possible and scale isn't changing, otherwise do a full redraw
        if (!scaleWillChange && this.canvas && this.ctx && this.data.length > 1) {
            this.incrementalRedraw(sample);
        } else {
            this.redraw();
        }
    }

    incrementalRedraw(newSample: number): void {
        if (!this.canvas || !this.ctx || this.data.length <= 1) return;
        
        const dpr = window.devicePixelRatio || 1;
        const width = this.canvas.width / dpr;
        const height = this.canvas.height / dpr;
        
        // Use cached min and max values for scaling
        const min = this.cachedMin!; // We know these aren't null here
        const max = this.cachedMax!;
        
        // Handle case where all values are the same
        let range = max - min;
        if (range === 0) {
            range = Math.max(1, Math.abs(min) * 0.1);
        }
        
        // Calculate point spacing
        const pointSpacing = width / (this.data.length > 1 ? this.data.length - 1 : 1);
        
        // Save current canvas state
        this.ctx.save();
        this.ctx.scale(dpr, dpr);
        
        // Shift existing content to the left
        const shiftAmount = pointSpacing;
        const imageData = this.ctx.getImageData(shiftAmount, 0, width - shiftAmount, height);
        this.ctx.clearRect(0, 0, width, height);
        this.ctx.putImageData(imageData, 0, 0);
        
        // Clear the right portion where we'll draw the new point
        this.ctx.clearRect(width - shiftAmount, 0, shiftAmount, height);
        
        // Draw the new segment
        const prevIndex = this.data.length - 2;
        const prevX = (prevIndex * pointSpacing);
        const prevY = height - ((this.data[prevIndex] - min) / range) * height;
        const newY = height - ((newSample - min) / range) * height;
        
        // Draw the new line segment
        this.ctx.beginPath();
        this.ctx.strokeStyle = this.lineColor;
        this.ctx.lineWidth = this.lineWidth;
        this.ctx.lineJoin = "round";
        this.ctx.moveTo(prevX, prevY);
        this.ctx.lineTo(width, newY);
        this.ctx.stroke();
        
        // If fillColor is set, fill the area under the new segment
        if (this.fillColor) {
            this.ctx.beginPath();
            this.ctx.moveTo(prevX, prevY);
            this.ctx.lineTo(width, newY);
            this.ctx.lineTo(width, height);
            this.ctx.lineTo(prevX, height);
            this.ctx.closePath();
            this.ctx.fillStyle = this.fillColor;
            this.ctx.fill();
        }
        
        // Draw minute marker if needed
        if (this.data.length % 60 === 0) {
            this.ctx.strokeStyle = "rgba(128, 128, 128, 0.2)";
            this.ctx.lineWidth = 0.5;
            this.ctx.beginPath();
            this.ctx.moveTo(width, 0);
            this.ctx.lineTo(width, height);
            this.ctx.stroke();
        }
        
        this.ctx.restore();
    }
    
    redraw(data?: number[]): void {
        if (data) {
            // If data is provided, replace the current data
            this.data = data.slice(-this.maxPoints); // Only keep the last maxPoints
            this.updateMinMax(); // Update cached min/max values
        }

        if (!this.canvas || !this.ctx) return;

        this.clear();
        if (this.data.length === 0) return;

        this.drawPath();
    }

    // Clear the canvas
    clear(): void {
        if (!this.canvas || !this.ctx) return;

        const dpr = window.devicePixelRatio || 1;
        this.ctx.clearRect(0, 0, this.canvas.width / dpr, this.canvas.height / dpr);
    }

    // Clear all data
    clearData(): void {
        this.data = [];
        this.cachedMin = null;
        this.cachedMax = null;
        this.clear();
    }

    drawPath(): void {
        if (!this.canvas || !this.ctx || this.data.length === 0) return;

        const dpr = window.devicePixelRatio || 1;
        const width = this.canvas.width / dpr;
        const height = this.canvas.height / dpr;

        // Use cached min and max values if available, otherwise calculate them
        if (this.cachedMin === null || this.cachedMax === null) {
            this.updateMinMax();
        }
        
        const min = this.cachedMin!;
        const max = this.cachedMax!;

        // Handle case where all values are the same
        let range = max - min;
        if (range === 0) {
            // If all values are the same, create an artificial range
            range = Math.max(1, Math.abs(min) * 0.1);
        }

        // Calculate point spacing
        const pointSpacing = width / (this.data.length > 1 ? this.data.length - 1 : 1);

        // Start drawing
        this.ctx.save();
        this.ctx.scale(dpr, dpr);

        // Begin path for the line
        this.ctx.beginPath();
        this.ctx.strokeStyle = this.lineColor;
        this.ctx.lineWidth = this.lineWidth;
        this.ctx.lineJoin = "round";

        // Move to the first point
        const initialY = height - ((this.data[0] - min) / range) * height;
        this.ctx.moveTo(0, initialY);

        // Draw lines to each point
        for (let i = 1; i < this.data.length; i++) {
            const x = i * pointSpacing;
            const y = height - ((this.data[i] - min) / range) * height;
            this.ctx.lineTo(x, y);
        }

        // Stroke the line
        this.ctx.stroke();

        // If fillColor is set, fill the area under the line
        if (this.fillColor) {
            this.ctx.lineTo(width, height);
            this.ctx.lineTo(0, height);
            this.ctx.closePath();
            this.ctx.fillStyle = this.fillColor;
            this.ctx.fill();
        }

        // Draw breakpoint indicators (minute markers)
        if (this.data.length > 60) {
            this.ctx.strokeStyle = "rgba(128, 128, 128, 0.2)";
            this.ctx.lineWidth = 0.5;

            // Draw vertical lines at minute intervals (every 60 points)
            for (let i = 60; i < this.data.length; i += 60) {
                const x = i * pointSpacing;
                this.ctx.beginPath();
                this.ctx.moveTo(x, 0);
                this.ctx.lineTo(x, height);
                this.ctx.stroke();
            }
        }

        this.ctx.restore();
    }
}

interface SparklineProps {
    model: SparklineModel;
    width?: number;
    height?: number;
    className?: string;
}

export const Sparkline = React.memo<SparklineProps>(({ model, width = 100, height = 30, className }) => {
    const canvasRef = useRef<HTMLCanvasElement>(null);

    // Set up canvas and context references on mount
    useEffect(() => {
        if (canvasRef.current) {
            model.canvas = canvasRef.current;
            model.ctx = canvasRef.current.getContext("2d");
            model.redraw(); // Initial draw if model has data
        }
    }, [model]);

    // Handle DPR for high-resolution displays
    const dpr = window.devicePixelRatio || 1;

    return (
        <canvas
            ref={canvasRef}
            width={width * dpr}
            height={height * dpr}
            style={{ width, height }}
            className={cn("overflow-hidden", className)}
        />
    );
});

Sparkline.displayName = "Sparkline";
