import React from 'react';
import type{ Algorithm, TreeType, ElementData } from '../types/types';

interface ControlPanelProps {
  serverUrl: string;
  setServerUrl: (url: string) => void;
  target: string;
  setTarget: (target: string) => void;
  treeType: TreeType;
  setTreeType: (type: TreeType) => void;
  algorithm: Algorithm;
  setAlgorithm: (algorithm: Algorithm) => void;
  treeCount: number;
  setTreeCount: (count: number) => void;
  logs: {message: string, type: string}[];
  stats: {
    algorithm: string;
    timeElapsed: number;
    nodesVisited: number;
    treesFound: number;
  };
  testConnection: () => void;
  visualizeRecipes: () => void;
  clearVisualization: () => void;
  elements: ElementData[];
  isLoading: boolean;
}

const ControlPanel: React.FC<ControlPanelProps> = ({
  serverUrl,
  setServerUrl,
  target,
  setTarget,
  treeType,
  setTreeType,
  algorithm,
  setAlgorithm,
  treeCount,
  setTreeCount,
  logs,
  stats,
  testConnection,
  visualizeRecipes,
  clearVisualization,
  elements,
  isLoading
}) => {
    const setLogs = (newLogs: { message: string; type: string }[]) => {
        console.log("Logs updated:", newLogs);
    };

 return (
    <div className="w-full lg:w-1/3 bg-white rounded-xl shadow-xl overflow-hidden border border-gray-100">
      <div className="bg-gradient-to-r from-blue-600 to-indigo-600 py-4 px-6">
        <h2 className="text-xl font-semibold text-white">Recipe Explorer Controls</h2>
      </div>
      
      <div className="p-6">
        <div className="bg-blue-50 p-4 rounded-lg mb-6 border border-blue-100">
          <div className="mb-3">
            <label htmlFor="serverUrl" className="block font-medium mb-2 text-gray-700">Server URL:</label>
            <div className="relative">
              <input 
                type="text" 
                id="serverUrl" 
                value={serverUrl} 
                onChange={(e) => setServerUrl(e.target.value)} 
                placeholder="e.g., http://localhost:8080"
                className="w-full p-3 border border-gray-300 rounded-lg pl-10 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition" 
              />
              <svg className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 2a8 8 0 100 16 8 8 0 000-16zm0 14a6 6 0 110-12 6 6 0 010 12z" clipRule="evenodd" />
                <path fillRule="evenodd" d="M10 4a1 1 0 100 2 1 1 0 000-2zm0 10a1 1 0 100-2 1 1 0 000 2z" clipRule="evenodd" />
              </svg>
            </div>
          </div>
          <button 
            onClick={testConnection} 
            disabled={isLoading}
            className="w-full bg-blue-500 hover:bg-blue-600 text-white py-3 px-4 rounded-lg transition transform hover:scale-[1.02] active:scale-[0.98] disabled:bg-blue-300 flex items-center justify-center gap-2"
          >
            {isLoading ? (
              <>
                <svg className="animate-spin h-5 w-5 text-black" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                <span>Testing Connection...</span>
              </>
            ) : (
              <>
                <svg className="w-5 h-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="black">
                  <path fillRule="evenodd" d="M3 5a2 2 0 012-2h10a2 2 0 012 2v8a2 2 0 01-2 2h-2.22l.123.489.804.804A1 1 0 0113 18H7a1 1 0 01-.707-1.707l.804-.804L7.22 15H5a2 2 0 01-2-2V5zm5.771 7H5V5h10v7H8.771z" clipRule="evenodd" />
                </svg>
                <span className="text-lg font-bold text-black">Test Connection</span>
              </>
            )}
          </button>
        </div>
        
        <div className="space-y-5">
          <div className="mb-4">
            <label htmlFor="target" className="block font-medium mb-2 text-gray-700">Target Element:</label>
            <div className="relative">
              <input 
                type="text" 
                id="target" 
                value={target} 
                onChange={(e) => setTarget(e.target.value)} 
                placeholder="e.g., Brick, Metal, Steam..."
                className="w-full p-3 border border-gray-300 rounded-lg pl-10 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition" 
                list="element-list"
              />
              <svg className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 2a8 8 0 100 16 8 8 0 000-16zm0 14a6 6 0 110-12 6 6 0 010 12zm-1-5a1 1 0 011-1h2a1 1 0 110 2h-2a1 1 0 01-1-1z" clipRule="evenodd" />
              </svg>
            </div>
            <datalist id="element-list">
              {elements.map((element, index) => (
                <option key={index} value={element.name} />
              ))}
            </datalist>
          </div>
          
          <div className="mb-4">
            <label htmlFor="treeType" className="block font-medium mb-2 text-gray-700">Tree Type:</label>
            <div className="relative">
              <select 
                id="treeType" 
                value={treeType} 
                onChange={(e) => setTreeType(e.target.value as TreeType)}
                className="w-full p-3 border border-gray-300 rounded-lg pl-10 appearance-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition bg-white"
              >
                <option value="best-recipes-tree">Best Recipe Tree</option>
                <option value="multiple-recipes-tree">Multiple Recipe Trees</option>
              </select>
              <svg className="absolute right-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400 pointer-events-none" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clipRule="evenodd" />
              </svg>
              <svg className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
              </svg>
            </div>
          </div>

          <div className="mb-4">
            <label className="block font-medium mb-2 text-gray-700">Algorithm:</label>
            <div className="grid grid-cols-3 gap-3">
              <button 
                className={`relative p-3 rounded-lg transition focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500
                  ${algorithm === 'bfs' 
                  ? 'bg-gradient-to-r from-blue-500 to-indigo-500 text-white shadow-lg' 
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}`}
                onClick={() => setAlgorithm('bfs')}
              >
                <div className="flex flex-col items-center">
                  <span className="font-medium">BFS</span>
                  <span className="text-xs mt-1 opacity-75">Breadth First</span>
                </div>
                {algorithm === 'bfs' && (
                  <span className="absolute top-1 right-1 w-2 h-2 bg-white rounded-full"></span>
                )}
              </button>
              <button 
                className={`relative p-3 rounded-lg transition focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500
                  ${algorithm === 'dfs' 
                  ? 'bg-gradient-to-r from-blue-500 to-indigo-500 text-white shadow-lg' 
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}`}
                onClick={() => setAlgorithm('dfs')}
              >
                <div className="flex flex-col items-center">
                  <span className="font-medium">DFS</span>
                  <span className="text-xs mt-1 opacity-75">Depth First</span>
                </div>
                {algorithm === 'dfs' && (
                  <span className="absolute top-1 right-1 w-2 h-2 bg-white rounded-full"></span>
                )}
              </button>
              <button 
                className={`relative p-3 rounded-lg transition focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500
                  ${algorithm === 'bidirectional' 
                  ? 'bg-gradient-to-r from-blue-500 to-indigo-500 text-white shadow-lg' 
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}`}
                onClick={() => setAlgorithm('bidirectional')}
              >
                <div className="flex flex-col items-center">
                  <span className="font-medium text-xs flex">Bidirectional</span>
                  <span className="text-xs mt-1 opacity-75">Bidirectional BFS</span>
                </div>
                {algorithm === 'bidirectional' && (
                  <span className="absolute top-1 right-1 w-2 h-2 bg-white rounded-full"></span>
                )}
              </button>
            </div>
          </div>

          <div className="mb-4">
            <label htmlFor="treeCount" className="block font-medium mb-2 text-gray-700">Number of Trees/Paths:</label>
            <div className="relative">
              <input 
                type="number" 
                id="treeCount" 
                value={treeCount} 
                onChange={(e) => setTreeCount(parseInt(e.target.value))}
                min="1" 
                max="5"
                className="w-full p-3 border border-gray-300 rounded-lg pl-10 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition" 
              />
              <svg className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path d="M10 12a2 2 0 100-4 2 2 0 000 4z" />
                <path fillRule="evenodd" d="M.458 10C1.732 5.943 5.522 3 10 3s8.268 2.943 9.542 7c-1.274 4.057-5.064 7-9.542 7S1.732 14.057.458 10zM14 10a4 4 0 11-8 0 4 4 0 018 0z" clipRule="evenodd" />
              </svg>
            </div>
          </div>
          
          <div className="flex gap-3 mb-6">
            <button 
              onClick={visualizeRecipes} 
              disabled={isLoading}
              className="flex-grow py-3 px-6 bg-gradient-to-r from-emerald-500 to-teal-500 text-white font-medium rounded-lg shadow-md hover:shadow-lg transition transform hover:translate-y-[-2px] active:translate-y-0 disabled:opacity-70 disabled:cursor-not-allowed disabled:transform-none flex items-center justify-center gap-2"
            >
              {isLoading ? (
                <>
                  <svg className="animate-spin h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  <span>Finding...</span>
                </>
              ) : (
                <>
                  <svg className="w-5 h-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
                  </svg>
                  <span>Find Recipes</span>
                </>
              )}
            </button>
            <button 
              onClick={clearVisualization}
              className="px-4 py-3 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition flex items-center justify-center"
              aria-label="Clear visualization"
            >
              <svg className="w-5 h-5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clipRule="evenodd" />
              </svg>
            </button>
          </div>
          
          <div className="grid grid-cols-2 gap-3 mb-6">
            <div className="bg-gradient-to-br from-blue-50 to-indigo-50 p-4 rounded-lg border border-blue-100 shadow-sm">
              <div className="text-sm text-gray-600 mb-1">Algorithm</div>
              <div className="text-lg font-bold text-blue-700">{stats.algorithm}</div>
            </div>
            <div className="bg-gradient-to-br from-blue-50 to-indigo-50 p-4 rounded-lg border border-blue-100 shadow-sm">
              <div className="text-sm text-gray-600 mb-1">Time (ms)</div>
              <div className="text-lg font-bold text-blue-700">{stats.timeElapsed}</div>
            </div>
            <div className="bg-gradient-to-br from-blue-50 to-indigo-50 p-4 rounded-lg border border-blue-100 shadow-sm">
              <div className="text-sm text-gray-600 mb-1">Nodes Visited</div>
              <div className="text-lg font-bold text-blue-700">{stats.nodesVisited}</div>
            </div>
            <div className="bg-gradient-to-br from-blue-50 to-indigo-50 p-4 rounded-lg border border-blue-100 shadow-sm">
              <div className="text-sm text-gray-600 mb-1">Trees Found</div>
              <div className="text-lg font-bold text-blue-700">{stats.treesFound}</div>
            </div>
          </div>
          
          <div>
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-gray-700 font-medium">Debug Log</h3>
              {logs.length > 0 && (
                <button 
                  onClick={() => setLogs([])} 
                  className="text-xs text-gray-500 hover:text-gray-700"
                >
                  Clear log
                </button>
              )}
            </div>
            <div className="h-48 overflow-y-auto border border-gray-300 p-3 rounded-lg bg-gray-50 font-mono text-sm">
              {logs.length === 0 ? (
                <div className="text-gray-400 text-center py-4">No log entries yet</div>
              ) : (
                logs.map((log, index) => (
                  <div 
                    key={index} 
                    className={`${
                      log.type === 'error' ? 'text-red-600' : 
                      log.type === 'success' ? 'text-green-600' : 
                      'text-gray-600'
                    } mb-1`}
                  >
                    <span className="opacity-75">[{new Date().toLocaleTimeString()}]</span> {log.message}
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ControlPanel;