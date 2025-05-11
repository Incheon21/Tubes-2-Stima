import React, { useState } from 'react';
import type { Algorithm, TreeType, ElementData } from '../types/types';

interface ControlPanelProps {
  target: string;
  setTarget: (target: string) => void;
  treeType: TreeType;
  setTreeType: (type: TreeType) => void;
  algorithm: Algorithm;
  setAlgorithm: (algorithm: Algorithm) => void;
  treeCount: number;
  setTreeCount: (count: number) => void;
  stats: {
    algorithm: string;
    timeElapsed: number;
    nodesVisited: number;
    treesFound: number;
  };
  visualizeRecipes: () => void;
  clearVisualization: () => void;
  elements: ElementData[];
  isLoading: boolean;
}

const ControlPanel: React.FC<ControlPanelProps> = ({
  target,
  setTarget,
  treeType,
  setTreeType,
  algorithm,
  setAlgorithm,
  treeCount,
  setTreeCount,
  stats,
  visualizeRecipes,
  clearVisualization,
  elements,
  isLoading
}) => {
  const [showAlgorithmInfo, setShowAlgorithmInfo] = useState<string | null>(null);
  const [animateHeader, setAnimateHeader] = useState(false);

  // Add animation effect to header on initial load
  React.useEffect(() => {
    setAnimateHeader(true);
  }, []);

  const handleTreeTypeChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const newTreeType = e.target.value as TreeType;
    setTreeType(newTreeType);
    
    // Reset tree count to 1 when selecting best-recipes-tree
    if (newTreeType === 'best-recipes-tree') {
      setTreeCount(1);
    }
  };

  return (
    <div className="w-full lg:w-1/3 bg-white rounded-xl shadow-xl overflow-hidden border border-gray-100 transition-all duration-500 hover:shadow-2xl">
      <div className={`bg-gradient-to-r from-purple-600 via-blue-600 to-indigo-700 py-5 px-6 ${animateHeader ? 'animate-gradient' : ''}`}>
        <h2 className="text-2xl font-bold text-white flex items-center">
          <svg className="w-7 h-7 mr-3 animate-float" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M12 2L2 7l10 5 10-5-10-5z" />
            <path d="M2 17l10 5 10-5" />
            <path d="M2 12l10 5 10-5" />
          </svg>
          Recipe Explorer
        </h2>
        <p className="text-blue-100 mt-1">Find crafting paths for game elements</p>
      </div>
      
      <div className="p-6">
        <div className="space-y-6">
          <div className="mb-5 relative group">
            <label htmlFor="target" className="block font-medium mb-2 text-gray-700 group-hover:text-blue-700 transition-colors duration-300">
              Target Element:
            </label>
            <div className="relative">
              <input 
                type="text" 
                id="target" 
                value={target} 
                onChange={(e) => setTarget(e.target.value)} 
                placeholder="e.g., Brick, Metal, Steam..."
                className="w-full p-3 border border-gray-300 rounded-lg pl-11 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all duration-300 shadow-sm hover:border-blue-400" 
                list="element-list"
              />
              <div className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-blue-500 transition-all duration-300">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9.75 3.104v5.714a2.25 2.25 0 01-.659 1.591L5 14.5M9.75 3.104c-.251.023-.501.05-.75.082m.75-.082a24.301 24.301 0 014.5 0m0 0v5.714c0 .597.237 1.17.659 1.591L19.8 15.3M14.25 3.104c.251.023.501.05.75.082M19.8 15.3l-1.57.393A9.065 9.065 0 0112 15a9.065 9.065 0 00-6.23-.693L5 14.5m14.8.8l1.402 1.402c1.232 1.232.65 3.318-1.067 3.611A48.309 48.309 0 0112 21c-2.773 0-5.491-.235-8.135-.687-1.718-.293-2.3-2.379-1.067-3.61L5 14.5" />
                </svg>
              </div>
            </div>
            <datalist id="element-list">
              {elements.map((element, index) => (
                <option key={index} value={element.name} />
              ))}
            </datalist>
          </div>
          
          <div className="mb-5 relative group">
            <label htmlFor="treeType" className="block font-medium mb-2 text-gray-700 group-hover:text-blue-700 transition-colors duration-300">Tree Type:</label>
            <div className="relative">
              <select 
                id="treeType" 
                value={treeType} 
                onChange={handleTreeTypeChange}
                className="w-full p-3 border border-gray-300 rounded-lg pl-11 appearance-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all duration-300 shadow-sm hover:border-blue-400 bg-white"
              >
                <option value="best-recipes-tree">Best Recipe Tree</option>
                <option value="multiple-recipes-tree">Multiple Recipe Trees</option>
              </select>
              <div className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-blue-500">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 3v11.25A2.25 2.25 0 006 16.5h2.25M3.75 3h-1.5m1.5 0h16.5m0 0h1.5m-1.5 0v11.25A2.25 2.25 0 0118 16.5h-2.25m-7.5 0h7.5m-7.5 0l-1 3m8.5-3l1 3m0 0l.5 1.5m-.5-1.5h-9.5m0 0l-.5 1.5m.75-9l3-3 2.148 2.148A12.061 12.061 0 0116.5 7.605" />
                </svg>
              </div>
              <div className="absolute right-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400 pointer-events-none">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clipRule="evenodd" />
                </svg>
              </div>
            </div>
          </div>

          <div className="mb-5">
            <label className="font-medium mb-3 text-gray-700 group-hover:text-blue-700 flex items-center justify-between">
              Algorithm:
              <span className="text-sm text-blue-500 hover:text-blue-700 cursor-pointer transition-colors flex items-center" 
                onClick={() => setShowAlgorithmInfo(showAlgorithmInfo ? null : 'info')}>
                <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 mr-1" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2h-1V9a1 1 0 00-1-1z" clipRule="evenodd" />
                </svg>
                What's this?
              </span>
            </label>
            
            {showAlgorithmInfo && (
              <div className="mb-4 p-4 bg-blue-50 border border-blue-100 rounded-lg text-sm text-gray-600 animate-fadeIn">
                <h4 className="font-semibold mb-2 text-blue-700">Search Algorithms:</h4>
                <ul className="space-y-2">
                  <li className="flex items-start">
                    <span className="font-semibold text-blue-600 inline-block w-14">BFS:</span>
                    <span>Breadth-First Search explores all neighbor nodes before moving to the next level, finding shortest paths first.</span>
                  </li>
                  <li className="flex items-start">
                    <span className="font-semibold text-blue-600 inline-block w-14">DFS:</span>
                    <span>Depth-First Search explores as far as possible along each branch before backtracking, good for deep paths.</span>
                  </li>
                  <li className="flex items-start">
                    <span className="font-semibold text-blue-600 inline-block w-14">Bidirectional:</span>
                    <span>Searches from both start and goal, meeting in the middle, often faster for complex recipes.</span>
                  </li>
                </ul>
              </div>
            )}
            
            <div className="grid grid-cols-3 gap-3">
              <button 
                className={`relative p-3 rounded-lg transition-all duration-300 transform hover:scale-105 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500
                  ${algorithm === 'bfs' 
                  ? 'bg-gradient-to-r from-blue-500 to-indigo-600 text-white shadow-lg scale-105' 
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}`}
                onClick={() => setAlgorithm('bfs')}
              >
                <div className="flex flex-col items-center">
                  <span className="font-medium">BFS</span>
                  <span className="text-xs mt-1 opacity-75">Breadth First</span>
                </div>
                {algorithm === 'bfs' && (
                  <span className="absolute top-1 right-1 w-2 h-2 bg-white rounded-full animate-pulse"></span>
                )}
              </button>
              <button 
                className={`relative p-3 rounded-lg transition-all duration-300 transform hover:scale-105 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500
                  ${algorithm === 'dfs' 
                  ? 'bg-gradient-to-r from-blue-500 to-indigo-600 text-white shadow-lg scale-105' 
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}`}
                onClick={() => setAlgorithm('dfs')}
              >
                <div className="flex flex-col items-center">
                  <span className="font-medium">DFS</span>
                  <span className="text-xs mt-1 opacity-75">Depth First</span>
                </div>
                {algorithm === 'dfs' && (
                  <span className="absolute top-1 right-1 w-2 h-2 bg-white rounded-full animate-pulse"></span>
                )}
              </button>
              <button 
                className={`relative p-3 rounded-lg transition-all duration-300 transform hover:scale-105 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500
                  ${algorithm === 'bidirectional' 
                  ? 'bg-gradient-to-r from-blue-500 to-indigo-600 text-white shadow-lg scale-105' 
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}`}
                onClick={() => setAlgorithm('bidirectional')}
              >
                <div className="flex flex-col items-center">
                  <span className="font-medium text-xs">Bidirectional</span>
                  <span className="text-xs mt-1 opacity-75">Both Ends</span>
                </div>
                {algorithm === 'bidirectional' && (
                  <span className="absolute top-1 right-1 w-2 h-2 bg-white rounded-full animate-pulse"></span>
                )}
              </button>
            </div>
          </div>

          {/* Only show tree count input when Multiple Recipe Trees is selected */}
          {treeType === 'multiple-recipes-tree' && (
            <div className="mb-5 relative group animate-fadeIn">
              <label htmlFor="treeCount" className="block font-medium mb-2 text-gray-700 group-hover:text-blue-700 transition-colors duration-300">
                Number of Trees/Paths:
              </label>
              <div className="relative">
                <input 
                  type="number" 
                  id="treeCount" 
                  value={treeCount} 
                  onChange={(e) => setTreeCount(parseInt(e.target.value) || 1)}
                  min="1" 
                  max="5"
                  className="w-full p-3 border border-gray-300 rounded-lg pl-11 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all duration-300 shadow-sm hover:border-blue-400" 
                />
                <div className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-blue-500">
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" />
                  </svg>
                </div>
              </div>
            </div>
          )}
          
          <div className="flex gap-3 mb-6">
            <button 
              onClick={visualizeRecipes} 
              disabled={isLoading}
              className="flex-grow py-4 px-6 bg-gradient-to-r from-purple-600 to-blue-600 text-white font-medium rounded-lg shadow-md hover:shadow-lg transition-all duration-300 transform hover:translate-y-[-2px] active:translate-y-0 disabled:opacity-70 disabled:cursor-not-allowed disabled:transform-none flex items-center justify-center gap-2"
            >
              {isLoading ? (
                <>
                  <svg className="animate-spin h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  <span>Finding Recipes...</span>
                </>
              ) : (
                <>
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                    <path d="M11 17a1 1 0 001.447.894l4-2A1 1 0 0017 15V9.236a1 1 0 00-1.447-.894l-4 2a1 1 0 00-.553.894V17zM15.211 6.276a1 1 0 000-1.788l-4.764-2.382a1 1 0 00-.894 0L4.789 4.488a1 1 0 000 1.788l4.764 2.382a1 1 0 00.894 0l4.764-2.382zM4.447 8.342A1 1 0 003 9.236V15a1 1 0 00.553.894l4 2A1 1 0 009 17v-5.764a1 1 0 00-.553-.894l-4-2z" />
                  </svg>
                  <span>Find Recipes</span>
                </>
              )}
            </button>
            <button 
              onClick={clearVisualization}
              className="px-4 py-3 bg-gradient-to-r from-gray-600 to-gray-700 text-white rounded-lg hover:from-gray-700 hover:to-gray-800 transition-all duration-300 flex items-center justify-center transform hover:translate-y-[-2px] active:translate-y-0"
              aria-label="Clear visualization"
            >
              <svg className="w-5 h-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clipRule="evenodd" />
              </svg>
            </button>
          </div>
          
          <div className="grid grid-cols-2 gap-3">
            <div className="p-4 rounded-lg border transition-all duration-300 transform hover:scale-105 hover:shadow-md bg-gradient-to-br from-violet-50 to-blue-50 border-blue-100">
              <div className="text-sm text-gray-600 mb-1">Algorithm</div>
              <div className="text-lg font-bold text-purple-700">{stats.algorithm}</div>
            </div>
            <div className="p-4 rounded-lg border transition-all duration-300 transform hover:scale-105 hover:shadow-md bg-gradient-to-br from-violet-50 to-blue-50 border-blue-100">
              <div className="text-sm text-gray-600 mb-1">Time (ms)</div>
              <div className="text-lg font-bold text-purple-700">{stats.timeElapsed}</div>
            </div>
            <div className="p-4 rounded-lg border transition-all duration-300 transform hover:scale-105 hover:shadow-md bg-gradient-to-br from-violet-50 to-blue-50 border-blue-100">
              <div className="text-sm text-gray-600 mb-1">Nodes Visited</div>
              <div className="text-lg font-bold text-purple-700">{stats.nodesVisited}</div>
            </div>
            <div className="p-4 rounded-lg border transition-all duration-300 transform hover:scale-105 hover:shadow-md bg-gradient-to-br from-violet-50 to-blue-50 border-blue-100">
              <div className="text-sm text-gray-600 mb-1">Trees Found</div>
              <div className="text-lg font-bold text-purple-700">{stats.treesFound}</div>
            </div>
          </div>
          
          <div className="mt-6 p-5 bg-gradient-to-br from-purple-50 to-indigo-50 rounded-lg border border-indigo-100 transition-all duration-300 hover:shadow-md">
            <h3 className="text-lg font-semibold text-purple-800 mb-3 flex items-center">
              <svg className="w-5 h-5 mr-2" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2h-1V9a1 1 0 00-1-1z" clipRule="evenodd" />
              </svg>
              About This Tool
            </h3>
            <p className="text-gray-600 text-sm leading-relaxed">
              This tool helps you discover crafting recipes for Little Alchemy 2 using graph search algorithms. 
              Enter an element name, choose your preferred algorithm, and explore the different paths to create it!
            </p>
            
            <div className="mt-4 flex gap-3">
              <a href="#" className="text-xs text-blue-600 hover:text-blue-800 transition-colors flex items-center">
                <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                How it works
              </a>
              <a href="#" className="text-xs text-blue-600 hover:text-blue-800 transition-colors flex items-center">
                <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
                </svg>
                View source
              </a>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ControlPanel;