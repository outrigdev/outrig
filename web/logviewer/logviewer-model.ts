import { Atom, atom, PrimitiveAtom } from "jotai";
import { AppModel } from "../appmodel";

class LogViewerModel {
    searchTerm: PrimitiveAtom<string> = atom("");
    filteredLogLines: Atom<LogLine[]> = atom((get) => {
        const search = get(this.searchTerm);
        const logs = get(AppModel.appRunLogs);
        
        if (!search) {
            return logs;
        }
        
        return logs.filter((log) => 
            log.msg.toLowerCase().includes(search.toLowerCase())
        );
    });
}

export { LogViewerModel };
