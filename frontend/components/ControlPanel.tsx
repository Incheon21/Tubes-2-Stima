import React, { useState, useEffect } from 'react';
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
  const [searchFocused, setSearchFocused] = useState(false);

  // Add animation effect to header on initial load
  useEffect(() => {
    setAnimateHeader(true);
  }, []);

  // Make sure treeCount is always 1 when treeType is 'best-recipes-tree'
  useEffect(() => {
    if (treeType === 'best-recipes-tree' && treeCount !== 1) {
      setTreeCount(1);
    }
  }, [treeType, treeCount, setTreeCount]);

  // Validate tree count to prevent invalid values
  const handleTreeCountChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    
    if (value === '') {
      return;
    }
    
    const count = parseInt(value);
    
    // Set minimum of 1
    if (count < 1) {
      setTreeCount(1);
    } else {
      setTreeCount(count);
    }
  };

  const algorithmDescriptions = {
    bfs: 'Breadth-First Search finds the shortest paths by exploring all neighbors before moving deeper.',
    dfs: 'Depth-First Search explores each branch as far as possible before backtracking.',
    bidirectional: 'Searches from both start and goal simultaneously, often faster for complex recipes.'
  };

  return (
    <div className="w-full lg:w-1/3 bg-white rounded-3xl shadow-2xl overflow-hidden border-0 transition-all duration-500 hover:shadow-[0_20px_60px_-15px_rgba(0,0,0,0.2)]">
      {/* Animated header with gradient background */}
      <div 
        className={`bg-gradient-to-r from-indigo-600 via-purple-600 to-blue-500 py-7 px-8 relative overflow-hidden ${
          animateHeader ? 'animate-gradient-slow' : ''
        }`}
      >
        {/* Decorative elements */}
        <div className="absolute top-0 left-0 w-full h-full opacity-20">
          <div className="absolute top-10 left-10 w-20 h-20 rounded-full bg-white blur-2xl"></div>
          <div className="absolute bottom-10 right-10 w-16 h-16 rounded-full bg-white blur-xl"></div>
        </div>
        
        <h2 className="text-3xl font-extrabold text-white flex items-center relative z-10">
          <svg className="w-8 h-8 mr-4 filter drop-shadow-lg animate-float" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M12 2L2 7l10 5 10-5-10-5z" />
            <path d="M2 17l10 5 10-5" />
            <path d="M2 12l10 5 10-5" />
          </svg>
          Recipe Explorer
        </h2>
        <p className="text-blue-100 mt-2 tracking-wide text-lg font-light max-w-xs">
          Discover crafting paths for game elements with advanced algorithms
        </p>
      </div>
      
      <div className="p-8">
        <div className="space-y-7">
          {/* Target Element Search with animated focus effect */}
          <div className="relative group">
            <label 
              htmlFor="target" 
              className="flex items-center text-sm font-bold mb-3 text-gray-700 group-hover:text-indigo-700 transition-colors duration-300 uppercase tracking-wider"
            >
              Target Element
              <span className="ml-2 px-2 py-1 bg-indigo-100 text-indigo-700 text-xs rounded-full">Required</span>
            </label>
            <div className={`relative transition-all duration-500 ${searchFocused ? 'scale-[1.02]' : ''}`}>
              <input 
                type="text" 
                id="target" 
                value={target} 
                onChange={(e) => setTarget(e.target.value)} 
                onFocus={() => setSearchFocused(true)}
                onBlur={() => setSearchFocused(false)}
                placeholder="Search element (e.g., Metal, Steam, Brick...)"
                className={`w-full p-4 border-2 rounded-xl pl-12 outline-none transition-all duration-300
                  bg-gradient-to-r from-white to-gray-50
                  ${searchFocused 
                    ? 'border-indigo-500 shadow-lg shadow-indigo-100' 
                    : 'border-gray-200 hover:border-indigo-300'}`}
                list="element-list"
              />
              <div className={`absolute left-3 top-1/2 transform -translate-y-1/2 transition-all duration-300 ${
                searchFocused ? 'text-indigo-600 scale-110' : 'text-indigo-400'
              }`}>
                <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
              </div>
            </div>
            <datalist id="element-list">
              {elements.map((element, index) => (
                <option key={index} value={element.name} />
              ))}
            </datalist>
            <p className="text-xs text-gray-500 mt-2 ml-1 italic">
              Find recipes for any element in the game's crafting system
            </p>
          </div>
          
          {/* Tree Type Selection with visual cards */}
          <div className="space-y-3">
            <label className="flex items-center text-sm font-bold text-gray-700 uppercase tracking-wider">
              Tree Visualization Type
            </label>
            
            <div className="grid grid-cols-2 gap-4 mt-2">
              <div 
                className={`relative p-4 rounded-xl border-2 transition-all duration-300 cursor-pointer ${
                  treeType === 'best-recipes-tree'
                    ? 'border-indigo-500 bg-indigo-50 shadow-md'
                    : 'border-gray-200 hover:border-indigo-300 bg-white'
                }`}
                onClick={() => setTreeType('best-recipes-tree')}
              >
                <div className="flex flex-col h-full">
                  <div className="flex items-center mb-2">
                    <div className={`w-4 h-4 rounded-full mr-2 ${
                      treeType === 'best-recipes-tree' ? 'bg-indigo-500' : 'border-2 border-gray-300'
                    }`}></div>
                    <span className="font-medium">Best Recipe</span>
                  </div>
                  <p className="text-xs text-gray-500 mb-2">
                    Shows the optimal crafting path
                  </p>
                  <div className="mt-auto text-center">
                    <svg xmlns="http://www.w3.org/2000/svg" className={`h-8 w-8 mx-auto ${treeType === 'best-recipes-tree' ? 'text-indigo-500' : 'text-gray-400'}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                    </svg>
                  </div>
                </div>
              </div>
              
              <div 
                className={`relative p-4 rounded-xl border-2 transition-all duration-300 cursor-pointer ${
                  treeType === 'multiple-recipes-tree'
                    ? 'border-indigo-500 bg-indigo-50 shadow-md'
                    : 'border-gray-200 hover:border-indigo-300 bg-white'
                }`}
                onClick={() => setTreeType('multiple-recipes-tree')}
              >
                <div className="flex flex-col h-full">
                  <div className="flex items-center mb-2">
                    <div className={`w-4 h-4 rounded-full mr-2 ${
                      treeType === 'multiple-recipes-tree' ? 'bg-indigo-500' : 'border-2 border-gray-300'
                    }`}></div>
                    <span className="font-medium">Multiple Recipes</span>
                  </div>
                  <p className="text-xs text-gray-500 mb-2">
                    Shows multiple crafting paths
                  </p>
                  <div className="mt-auto text-center">
                    <svg xmlns="http://www.w3.org/2000/svg" className={`h-8 w-8 mx-auto ${treeType === 'multiple-recipes-tree' ? 'text-indigo-500' : 'text-gray-400'}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Only show tree count input when Multiple Recipe Trees is selected */}
          {treeType === 'multiple-recipes-tree' && (
            <div className="animate-fadeIn space-y-3">
              <label htmlFor="treeCount" className="block text-sm font-bold text-gray-700 uppercase tracking-wider">
                Number of Trees to Find
              </label>
              <div className="flex items-center">
                <button 
                  onClick={() => treeCount > 1 && setTreeCount(treeCount - 1)}
                  className="p-3 bg-gray-100 rounded-l-lg hover:bg-gray-200 transition-colors"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 12H4" />
                  </svg>
                </button>
                <input 
                  type="number" 
                  id="treeCount" 
                  value={treeCount} 
                  onChange={handleTreeCountChange}
                  min="1" 
                  className="w-full p-3 text-center text-lg font-medium border-y-2 border-gray-200 outline-none"
                />
                <button 
                  onClick={() => setTreeCount(treeCount + 1)}
                  className="p-3 bg-gray-100 rounded-r-lg hover:bg-gray-200 transition-colors"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                  </svg>
                </button>
              </div>
              <p className="text-xs text-gray-500 italic">
                More trees will show alternative crafting paths, but may take longer to calculate
              </p>
            </div>
          )}
          
          {/* Algorithm Selection */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <label className="text-sm font-bold text-gray-700 uppercase tracking-wider">
                Search Algorithm
              </label>
              <button 
                className="text-sm text-indigo-600 hover:text-indigo-800 flex items-center transition-colors group"
                onClick={() => setShowAlgorithmInfo(showAlgorithmInfo ? null : 'info')}
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 mr-1 group-hover:animate-pulse" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span className="group-hover:underline">How algorithms work</span>
              </button>
            </div>
            
            {showAlgorithmInfo && (
              <div className="my-4 p-5 bg-gradient-to-r from-indigo-50 to-blue-50 rounded-xl border border-indigo-100 animate-fadeIn">
                <div className="flex items-start mb-3">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 mr-2 text-indigo-600 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
                  </svg>
                  <p className="text-sm text-indigo-900 font-medium">
                    Different algorithms excel at finding different types of crafting paths. Choose the one that best fits your needs.
                  </p>
                </div>
                <ul className="space-y-3 mt-4">
                  {(['bfs', 'dfs', 'bidirectional'] as Algorithm[]).map(algo => (
                    <li key={algo} className="flex items-center p-3 bg-white bg-opacity-60 rounded-lg">
                      <div className={`h-3 w-3 rounded-full mr-3 ${
                        algorithm === algo ? 'bg-indigo-500' : 'bg-gray-300'
                      }`}></div>
                      <div>
                        <div className="font-semibold text-indigo-900">
                          {algo === 'bfs' ? 'BFS (Breadth-First Search)' : 
                           algo === 'dfs' ? 'DFS (Depth-First Search)' : 
                           'Bidirectional Search'}
                        </div>
                        <div className="text-xs text-gray-600 mt-1">{algorithmDescriptions[algo]}</div>
                      </div>
                    </li>
                  ))}
                </ul>
              </div>
            )}
            
            <div className="grid grid-cols-3 gap-4">
              {(['bfs', 'dfs', 'bidirectional'] as Algorithm[]).map(algo => (
                <button 
                  key={algo}
                  onClick={() => setAlgorithm(algo)}
                  className={`relative p-4 rounded-xl transition-all duration-300 transform ${
                    algorithm === algo 
                      ? 'bg-gradient-to-br from-indigo-500 to-purple-600 text-white font-medium shadow-lg scale-[1.03]' 
                      : 'bg-white border-2 border-gray-200 text-gray-700 hover:border-indigo-300 hover:bg-indigo-50'
                  }`}
                >
                  <div className="flex flex-col items-center">
                    <span className="text-lg font-semibold">{algo.toUpperCase()}</span>
                    <span className="text-xs mt-1 opacity-75">
                      {algo === 'bfs' ? 'Breadth First' : 
                       algo === 'dfs' ? 'Depth First' : 'Both Ways'}
                    </span>
                  </div>
                  {algorithm === algo && (
                    <span className="absolute top-2 right-2 flex h-3 w-3">
                      <span className="animate-ping absolute h-full w-full rounded-full bg-white opacity-75"></span>
                      <span className="rounded-full h-3 w-3 bg-white"></span>
                    </span>
                  )}
                </button>
              ))}
            </div>
          </div>
          
          {/* Action Buttons */}
          <div className="pt-3">
            <div className="flex gap-4">
              <button 
                onClick={visualizeRecipes} 
                disabled={isLoading}
                className={`flex-grow py-4 px-6 rounded-xl text-white font-semibold shadow-lg transition-all duration-300 transform flex items-center justify-center gap-3
                  ${isLoading 
                    ? 'bg-gray-400 cursor-not-allowed'
                    : 'bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-700 hover:to-purple-700 hover:shadow-indigo-200 hover:shadow-xl hover:-translate-y-1 active:translate-y-0'
                  }`}
              >
                {isLoading ? (
                  <>
                    <svg className="animate-spin h-5 w-5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    <span>Finding Recipes...</span>
                  </>
                ) : (
                  <>
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                    </svg>
                    <span>Find Recipes</span>
                  </>
                )}
              </button>
              <button 
                onClick={clearVisualization}
                className="p-4 bg-white border-2 border-gray-300 text-gray-700 rounded-xl hover:bg-gray-100 transition-all duration-300 flex items-center justify-center transform hover:-translate-y-1 active:translate-y-0"
                aria-label="Clear visualization"
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </div>

          {/* Stats Dashboard */}
          <div className="mt-6 pt-6 border-t border-gray-200">
            <h3 className="text-sm font-bold text-gray-700 uppercase tracking-wider mb-4">Search Results</h3>
            <div className="grid grid-cols-2 gap-4">
              <div className="stats-card">
                <div className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-1">Algorithm</div>
                <div className="flex items-center">
                  <div className="w-3 h-3 rounded-full bg-indigo-500 mr-2"></div>
                  <div className="text-xl font-bold text-indigo-700">{stats.algorithm}</div>
                </div>
              </div>
              
              <div className="stats-card">
                <div className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-1">Time</div>
                <div className="flex items-center">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 text-purple-500 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <div className="text-xl font-bold text-purple-700">{stats.timeElapsed} <span className="text-xs">ms</span></div>
                </div>
              </div>
              
              <div className="stats-card">
                <div className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-1">Nodes</div>
                <div className="flex items-center">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 text-indigo-500 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                  </svg>
                  <div className="text-xl font-bold text-indigo-700">{stats.nodesVisited}</div>
                </div>
              </div>
              
              <div className="stats-card">
                <div className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-1">Trees</div>
                <div className="flex items-center">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 text-purple-500 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
                  </svg>
                  <div className="text-xl font-bold text-purple-700">{stats.treesFound}</div>
                </div>
              </div>
            </div>
          </div>
          
          {/* About Section with Hover Card Effect */}
          <div className="mt-8 group">
            <div className="p-6 bg-gradient-to-br from-indigo-50 via-purple-50 to-blue-50 rounded-xl border border-indigo-100 transition-all duration-500 group-hover:shadow-xl relative overflow-hidden">
              {/* Decorative elements */}
              <div className="absolute -top-10 -right-10 w-24 h-24 bg-indigo-100 rounded-full opacity-50 group-hover:animate-ping-slow"></div>
              <div className="absolute -bottom-12 -left-12 w-32 h-32 bg-purple-100 rounded-full opacity-30 group-hover:animate-ping-slow animation-delay-500"></div>
              
              <div className="relative z-10">
                <h3 className="text-xl font-bold text-indigo-900 mb-3 flex items-center">
                  <svg className="w-6 h-6 mr-2 text-indigo-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  About Recipe Explorer
                </h3>
                <p className="text-gray-700 leading-relaxed">
                  This tool helps you discover optimal crafting recipes using graph search algorithms. 
                  Enter any target element, choose your preferred algorithm, and explore different 
                  paths to create it from basic elements!
                </p>
                
                <div className="mt-5 flex flex-wrap gap-3">
                  <a href="#" className="inline-flex items-center px-3 py-2 bg-white rounded-lg text-sm text-indigo-700 hover:bg-indigo-50 transition-colors border border-indigo-100 font-medium">
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
                    </svg>
                    How it works
                  </a>
                  <a href="#" className="inline-flex items-center px-3 py-2 bg-white rounded-lg text-sm text-indigo-700 hover:bg-indigo-50 transition-colors border border-indigo-100 font-medium">
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
                    </svg>
                    View source
                  </a>
                  <a href="#" className="inline-flex items-center px-3 py-2 bg-white rounded-lg text-sm text-indigo-700 hover:bg-indigo-50 transition-colors border border-indigo-100 font-medium">
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    FAQ
                  </a>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      
    </div>
  );
};

export default ControlPanel;