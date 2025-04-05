import { AppModel } from "@/appmodel";
import { X } from "lucide-react";
import React, { useEffect, useRef } from "react";

interface ModalProps {
    isOpen: boolean;
    title: string;
    children: React.ReactNode;
}

export const Modal: React.FC<ModalProps> = ({ isOpen, title, children }) => {
    const modalRef = useRef<HTMLDivElement>(null);

    // Add a class to the body when the modal is open to prevent scrolling
    useEffect(() => {
        if (isOpen) {
            document.body.classList.add("modal-open");
        } else {
            document.body.classList.remove("modal-open");
        }

        return () => {
            document.body.classList.remove("modal-open");
        };
    }, [isOpen]);

    if (!isOpen) return null;

    return (
        <div 
            className="fixed inset-0 flex items-center justify-center z-50" 
            onClick={() => AppModel.closeSettingsModal()}
        >
            {/* Backdrop */}
            <div
                className="absolute inset-0 bg-[#000000]/50 backdrop-blur-[2px]"
                aria-hidden="true"
                onClick={() => AppModel.closeSettingsModal()}
            ></div>
            
            {/* Modal content - stop propagation to prevent closing when clicked */}
            <div
                ref={modalRef}
                className="bg-panel border border-border rounded-md shadow-lg w-[500px] max-w-[90vw] max-h-[80vh] flex flex-col focus:outline-none z-10"
                tabIndex={-1}
                onClick={(e) => e.stopPropagation()}
                role="dialog" 
                aria-modal="true"
            >
                {/* Header */}
                <div className="flex justify-between items-center px-4 py-3 border-b border-border">
                    <h2 className="text-primary font-medium">{title}</h2>
                    <button
                        onClick={() => {
                            AppModel.closeSettingsModal();
                        }}
                        className="text-muted hover:text-primary cursor-pointer"
                        aria-label="Close"
                    >
                        <X size={18} />
                    </button>
                </div>

                {/* Content */}
                <div className="p-4 overflow-auto">{children}</div>
            </div>
        </div>
    );
};
