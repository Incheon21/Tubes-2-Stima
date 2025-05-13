import React, { useState, useEffect } from 'react';
import type { TreeData, Algorithm } from '../types/types';

interface TreeDetailsProps {
  tree: TreeData;
  targetElement?: string; 
  algorithm: Algorithm;
}

const TreeDetails: React.FC<TreeDetailsProps> = ({ tree, algorithm }) => {
  const [showFullTree, setShowFullTree] = useState<boolean>(true);
  const [loadingImages, setLoadingImages] = useState<boolean>(false);
  const [imageErrors, setImageErrors] = useState<Set<string>>(new Set());
  
  const calculateComplexity = (node: TreeData): number => {
    if (!node.ingredients?.length) return 0;
    return 1 + Math.max(...node.ingredients.map(calculateComplexity));
  };
  
  const complexity = calculateComplexity(tree);
  const complexityLabel = complexity <= 1 ? 'Simple' : complexity <= 3 ? 'Moderate' : 'Complex';
  const complexityColor = complexity <= 1 ? 'green' : complexity <= 3 ? 'blue' : 'purple';

  const resolveImagePath = (node: TreeData): string => {
    if (!node) return '';
    
    if (node.imagePath) {
      if (node.imagePath.startsWith('/images/') || node.imagePath.startsWith('images/')) {
        return node.imagePath;
      }
      
      if (node.imagePath.includes('\\images\\')) {
        const filename = node.imagePath.split('\\').pop();
        return filename ? `/images/${filename}` : '';
      }
      
      if (!node.imagePath.includes('/') && !node.imagePath.includes('\\')) {
        return `/images/${node.imagePath}`;
      }
      
      return node.imagePath;
    }
    
    return '';
  };

  useEffect(() => {
    setImageErrors(new Set());
    setLoadingImages(false);
  }, [tree]);

  const getElementColor = (name: string) => {
    const colors = [
      'from-blue-400 to-indigo-500',
      'from-green-400 to-emerald-500', 
      'from-purple-400 to-fuchsia-500',
      'from-amber-400 to-orange-500',
      'from-red-400 to-rose-500',
      'from-cyan-400 to-sky-500'
    ];
    
    const hash = name.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
    return colors[hash % colors.length];
  };
  
  const getElementInitials = (name: string) => {
    if (!name) return '??';
    
    if (name.includes(' ')) {
      return name.split(' ')
        .filter(word => word.length > 0)
        .slice(0, 2)
        .map(word => word[0].toUpperCase())
        .join('');
    }
    
    return name.substring(0, 2).toUpperCase();
  };

  const getElementImage = (node: TreeData) => {
    if (imageErrors.has(node.name)) {
      return (
        <div className={`w-5 h-5 mr-2 rounded-sm bg-gradient-to-br ${getElementColor(node.name)} flex items-center justify-center text-white text-xs font-medium`}>
          {getElementInitials(node.name)}
        </div>
      );
    }
    
    const imagePath = resolveImagePath(node);
    
    if (imagePath) {
      return (
        <img 
          src={imagePath}
          alt={node.name} 
          className="w-5 h-5 mr-2 object-contain rounded-sm"
          onError={(e) => {
            const target = e.target as HTMLImageElement;
            target.onerror = null;
            target.style.display = 'none';
            
            setImageErrors(prev => new Set(prev).add(node.name));
            
            const parent = target.parentElement;
            if (parent) {
              const fallback = document.createElement('div');
              fallback.className = `w-5 h-5 mr-2 rounded-sm bg-gradient-to-br ${getElementColor(node.name)} flex items-center justify-center text-white text-xs font-medium`;
              fallback.innerText = getElementInitials(node.name);
              parent.insertBefore(fallback, target);
            }
          }}
        />
      );
    } else {
      return (
        <div className={`w-5 h-5 mr-2 rounded-sm bg-gradient-to-br ${getElementColor(node.name)} flex items-center justify-center text-white text-xs font-medium`}>
          {getElementInitials(node.name)}
        </div>
      );
    }
  };

  const getLargeElementImage = (node: TreeData) => {
    if (imageErrors.has(node.name)) {
      return (
        <div className={`w-12 h-12 rounded-full bg-gradient-to-br ${getElementColor(node.name)} flex items-center justify-center text-white text-lg font-medium`}>
          {getElementInitials(node.name)}
        </div>
      );
    }
    
    const imagePath = resolveImagePath(node);
    
    if (imagePath) {
      return (
        <img 
          src={imagePath} 
          alt={node.name} 
          className="w-12 h-12 object-contain" 
          onError={(e) => {
            const target = e.target as HTMLImageElement;
            target.onerror = null;
            setImageErrors(prev => new Set(prev).add(node.name));
            
            const parent = target.parentElement;
            if (parent) {
              target.style.display = 'none';
              const fallback = document.createElement('div');
              fallback.className = `w-12 h-12 rounded-full bg-gradient-to-br ${getElementColor(node.name)} flex items-center justify-center text-white text-lg font-medium`;
              fallback.innerText = getElementInitials(node.name);
              parent.appendChild(fallback);
            }
          }}
        />
      );
    } else {
      return (
        <div className={`w-12 h-12 rounded-full bg-gradient-to-br ${getElementColor(node.name)} flex items-center justify-center text-white text-lg font-medium`}>
          {getElementInitials(node.name)}
        </div>
      );
    }
  };

  return (
    <div className="animate-fadeIn">
      {loadingImages && (
        <div className="flex justify-center items-center py-4">
          <div className="animate-pulse flex space-x-2">
            <div className="w-2 h-2 bg-blue-400 rounded-full"></div>
            <div className="w-2 h-2 bg-blue-400 rounded-full animation-delay-200"></div>
            <div className="w-2 h-2 bg-blue-400 rounded-full animation-delay-400"></div>
          </div>
          <span className="text-sm text-gray-500 ml-3">Loading images...</span>
        </div>
      )}
      
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-800 flex items-center">
          <svg className="w-5 h-5 mr-2 text-blue-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
            <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
          </svg>
          Recipe Details
        </h3>
        
        <div className="flex gap-2">
          {complexity > 0 && (
            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-${complexityColor}-100 text-${complexityColor}-800`}>
              <svg className="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              {complexityLabel} Recipe
            </span>
          )}
          
          <div className="flex">
            <button
              onClick={() => setShowFullTree(true)}
              className={`px-2 py-1 text-xs rounded-l-md border ${
                showFullTree 
                  ? 'bg-blue-100 text-blue-700 border-blue-300' 
                  : 'bg-gray-50 text-gray-600 border-gray-300 hover:bg-gray-100'
              }`}
            >
              Full Tree
            </button>
            <button
              onClick={() => setShowFullTree(false)}
              className={`px-2 py-1 text-xs rounded-r-md border-t border-b border-r ${
                !showFullTree 
                  ? 'bg-blue-100 text-blue-700 border-blue-300' 
                  : 'bg-gray-50 text-gray-600 border-gray-300 hover:bg-gray-100'
              }`}
            >
              Simplified
            </button>
          </div>
        </div>
      </div>
      
      <div className="bg-gradient-to-br from-gray-50 to-blue-50 p-5 rounded-xl border border-blue-100 shadow-sm">
        <div className="flex items-center space-x-4 mb-6 pb-4 border-b border-blue-100">
          <div className="bg-white p-3 rounded-full shadow-sm flex items-center justify-center">
            {getLargeElementImage(tree)}
          </div>
          <div>
            <h2 className="text-xl font-bold text-gray-800 flex items-center">
              {tree.name}
              {tree.isBaseElement && (
                <span className="ml-2 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
                  Base Element
                </span>
              )}
            </h2>
            <p className="text-sm text-gray-500 mt-1">
              {tree.ingredients?.length > 0 
                ? `Created through ${tree.ingredients.length} ingredient${tree.ingredients.length > 1 ? 's' : ''}` 
                : tree.isBaseElement 
                  ? 'A fundamental element' 
                  : 'No recipe found'
              }
            </p>
          </div>
        </div>
        {showFullTree ? (
          tree.ingredients?.length > 0 ? (
            <div className="space-y-4">
              <div className="font-medium text-gray-700 flex items-center">
                <svg className="w-5 h-5 mr-2 text-blue-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
                </svg>
                Complete Recipe Tree:
              </div>
              <div className="bg-white p-4 rounded-lg border border-gray-200 shadow-sm overflow-auto max-h-96">
                {renderRecipeTree(tree, 0)}
              </div>
            </div>
          ) : (
            <div className="bg-white p-4 rounded-lg border border-gray-200 text-gray-600 italic flex items-center justify-center">
              <svg className="w-5 h-5 mr-2 text-gray-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2h-1V9a1 1 0 00-1-1z" clipRule="evenodd" />
              </svg>
              {tree.isBaseElement ? 'This is a base element with no recipe' : 'No recipe information available'}
            </div>
          )
        ) : (
          <div className="space-y-4">
            <div className="font-medium text-gray-700 flex items-center">
              <svg className="w-5 h-5 mr-2 text-blue-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M12.293 5.293a1 1 0 011.414 0l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-2.293-2.293a1 1 0 010-1.414z" clipRule="evenodd" />
              </svg>
              Simplified Recipe Path:
            </div>
            <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-5">
              <div className="flex flex-wrap items-center gap-2">
                {renderShortPath(tree, algorithm)}
              </div>
            </div>
          </div>
        )}
      </div>
      {(algorithm === 'bfs' || algorithm === 'dfs') && tree.ingredients?.length > 0 && showFullTree && (
        <div className="mt-6 animate-fadeIn">
          <h4 className="text-gray-800 font-semibold mb-3 flex items-center">
            <svg className="w-5 h-5 mr-2 text-blue-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M12.293 5.293a1 1 0 011.414 0l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-2.293-2.293a1 1 0 010-1.414z" clipRule="evenodd" />
            </svg>
            {algorithm.toUpperCase()} Optimal Path
          </h4>
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-5">
            <div className="flex flex-wrap items-center gap-2">
              {renderShortPath(tree, algorithm)}
            </div>
            
            <div className="mt-4 text-xs text-gray-500 bg-gray-50 rounded-lg p-3">
              <p>
                The {algorithm.toUpperCase()} algorithm {algorithm === 'bfs' ? 'finds the shortest path by exploring all nearby elements first' : 'explores each crafting branch fully before trying alternatives'}. This can result in different recipe paths depending on the algorithm chosen.
              </p>
            </div>
          </div>
        </div>
      )}

      {tree.ingredients?.length > 0 && showFullTree && (
        <div className="mt-6 bg-gray-50 rounded-xl border border-gray-200 p-5 animate-fadeIn">
          <h4 className="text-gray-800 font-semibold mb-3 flex items-center">
            <svg className="w-5 h-5 mr-2 text-blue-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M3 3a1 1 0 000 2v8a2 2 0 002 2h2.586l-1.293 1.293a1 1 0 101.414 1.414L10 15.414l2.293 2.293a1 1 0 001.414-1.414L12.414 15H15a2 2 0 002-2V5a1 1 0 100-2H3zm11 4a1 1 0 10-2 0v4a1 1 0 102 0V7zm-3 1a1 1 0 10-2 0v3a1 1 0 102 0V8zM8 9a1 1 0 00-2 0v2a1 1 0 102 0V9z" clipRule="evenodd" />
            </svg>
            Recipe Statistics
          </h4>
          
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {renderRecipeStats(tree)}
          </div>
        </div>
      )}

      {/* Debug display for development */}
      {process.env.NODE_ENV === 'development' && imageErrors.size > 0 && (
        <div className="mt-4 p-3 bg-yellow-50 border border-yellow-200 rounded-lg text-sm">
          <p className="font-medium text-yellow-800">Image Loading Issue</p>
          <p className="text-yellow-700">Failed to load {imageErrors.size} images. Using fallbacks instead.</p>
          <details className="mt-1">
            <summary className="cursor-pointer text-yellow-600 font-medium">Debug details</summary>
            <div className="text-xs mt-2 text-yellow-700 bg-yellow-100 p-2 rounded overflow-auto max-h-40">
              {Array.from(imageErrors).map(name => (
                <div key={name} className="mb-1">• {name}</div>
              ))}
            </div>
          </details>
        </div>
      )}
    </div>
  );
  
  function renderRecipeTree(node: TreeData, depth: number) {
    if (!node) return null;
    
    const getBgColor = () => {
      if (node.isBaseElement) return "bg-yellow-50 border-yellow-200";
      if (depth === 0) return "bg-green-50 border-green-200"; 
      if (node.isCircularReference) return "bg-orange-50 border-orange-200";
      return "bg-blue-50 border-blue-200";
    };
    
    const getNodeIcon = () => {
      if (node.isBaseElement) {
        return (
          <svg className="w-4 h-4 text-yellow-600" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
            <path d="M11 17a1 1 0 001.447.894l4-2A1 1 0 0017 15V9.236a1 1 0 00-1.447-.894l-4 2a1 1 0 00-.553.894V17zM15.211 6.276a1 1 0 000-1.788l-4.764-2.382a1 1 0 00-.894 0L4.789 4.488a1 1 0 000 1.788l4.764 2.382a1 1 0 00.894 0l4.764-2.382zM4.447 8.342A1 1 0 003 9.236V15a1 1 0 00.553.894l4 2A1 1 0 009 17v-5.764a1 1 0 00-.553-.894l-4-2z" />
          </svg>
        );
      }
      if (depth === 0) {
        return (
          <svg className="w-4 h-4 text-green-600" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
            <path fillRule="evenodd" d="M5 2a1 1 0 011 1v1h1a1 1 0 010 2H6v1a1 1 0 01-2 0V6H3a1 1 0 010-2h1V3a1 1 0 011-1zm0 10a1 1 0 011 1v1h1a1 1 0 110 2H6v1a1 1 0 11-2 0v-1H3a1 1 0 110-2h1v-1a1 1 0 011-1zM12 2a1 1 0 01.967.744L14.146 7.2 17.5 9.134a1 1 0 010 1.732l-3.354 1.935-1.18 4.455a1 1 0 01-1.933 0L9.854 12.8 6.5 10.866a1 1 0 010-1.732l3.354-1.935 1.18-4.455A1 1 0 0112 2z" clipRule="evenodd" />
          </svg>
        );
      }
      if (node.isCircularReference) {
        return (
          <svg className="w-4 h-4 text-orange-600" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
            <path fillRule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clipRule="evenodd" />
          </svg>
        );
      }
      return (
        <svg className="w-4 h-4 text-blue-600" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
          <path d="M9 2a1 1 0 000 2h2a1 1 0 100-2H9z" />
          <path fillRule="evenodd" d="M4 5a2 2 0 012-2 3 3 0 003 3h2a3 3 0 003-3 2 2 0 012 2v11a2 2 0 01-2 2H6a2 2 0 01-2-2V5zm3 4a1 1 0 000 2h.01a1 1 0 100-2H7zm3 0a1 1 0 000 2h3a1 1 0 100-2h-3zm-3 4a1 1 0 100 2h.01a1 1 0 100-2H7zm3 0a1 1 0 100 2h3a1 1 0 100-2h-3z" clipRule="evenodd" />
        </svg>
      );
    };
  
    return (
      <div className="tree-node animate-fadeIn" style={{ animationDelay: `${depth * 100}ms` }}>
        <div className={`${getBgColor()} rounded-md p-2 border flex items-center mb-1 shadow-sm group hover:shadow-md transition-all duration-200`}>
          <div className="mr-2">
            {getNodeIcon()}
          </div>
          <div className="flex-grow flex items-center">
            {getElementImage(node)}
            <span className={depth === 0 ? "font-medium" : ""}>{node.name}</span>
          </div>
          {node.isBaseElement && (
            <span className="bg-yellow-200 text-yellow-800 text-xs px-1.5 py-0.5 rounded ml-2 hidden group-hover:inline-block">
              Base
            </span>
          )}
          {node.isCircularReference && (
            <span className="bg-orange-200 text-orange-800 text-xs px-1.5 py-0.5 rounded ml-2 hidden group-hover:inline-block">
              Circular
            </span>
          )}
        </div>
  
        {Array.isArray(node.ingredients) && node.ingredients.length > 0 && (
          <div className="ml-6 pl-4 border-l-2 border-dashed border-blue-200 space-y-1">
            {node.ingredients.map((child, idx) => (
              <div key={idx} className="relative">
                <div className="absolute -left-4 top-1/2 transform -translate-y-1/2 w-3 h-px bg-blue-200" />
                {renderRecipeTree(child, depth + 1)}
              </div>
            ))}
          </div>
        )}
      </div>
    );
  }
  
  function renderShortPath(tree: TreeData, algorithm: Algorithm) {
    const baseElements = ['Water', 'Fire', 'Earth', 'Air'];
    const findBaseElements = (node: TreeData): TreeData[] => {
      if (!node) return [];
      
      if (baseElements.includes(node.name) || !node.ingredients || node.ingredients.length === 0) {
        return [node];
      }
      
      let bases: TreeData[] = [];
      if (algorithm === 'dfs') {
        if (Array.isArray(node.ingredients)) {
          for (const ingredient of node.ingredients) {
            bases = [...bases, ...findBaseElements(ingredient)];
          }
        }
      } else {
        if (Array.isArray(node.ingredients)) {
          for (const ingredient of [...node.ingredients].reverse()) {
            bases = [...bases, ...findBaseElements(ingredient)];
          }
        }
      }
      return bases;
    };
    
    const findKeyIntermediates = (node: TreeData): TreeData[] => {
      if (!node || !node.ingredients) return [];
      
      if (node.ingredients.length === 0) {
        return [];
      }
      
      let intermediates: TreeData[] = [];
      if (!baseElements.includes(node.name) && node.name !== tree.name) {
        intermediates.push(node);
      }
      
      if (Array.isArray(node.ingredients)) {
        for (const ingredient of node.ingredients) {
          intermediates = [...intermediates, ...findKeyIntermediates(ingredient)];
        }
      }
      return intermediates;
    };
    
    const baseElementNodes = findBaseElements(tree);
    const intermediateNodes = findKeyIntermediates(tree);
    
    return (
      <>
        {baseElementNodes.map((node, i) => (
          <React.Fragment key={`base-${i}`}>
            <div className="bg-yellow-50 border border-yellow-200 px-3 py-2 rounded-lg flex items-center shadow-sm hover:bg-yellow-100 transition duration-300 animate-fadeIn" style={{animationDelay: `${i * 100}ms`}}>
              {getElementImage(node)}
              {node.name}
            </div>
            {i < baseElementNodes.length - 1 && (
              <div className="text-lg font-bold text-gray-500 animate-fadeIn" style={{animationDelay: `${i * 100 + 50}ms`}}>+</div>
            )}
          </React.Fragment>
        ))}
        
        {baseElementNodes.length > 0 && (intermediateNodes.length > 0 || tree) && (
          <div className="flex items-center mx-3 animate-fadeIn" style={{animationDelay: `${baseElementNodes.length * 100}ms`}}>
            <svg className="w-6 h-6 text-blue-500 animate-pulse" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M12.293 5.293a1 1 0 011.414 0l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-2.293-2.293a1 1 0 010-1.414z" clipRule="evenodd" />
            </svg>
          </div>
        )}
        
        {intermediateNodes.slice(0, 3).map((node, i) => (
          <React.Fragment key={`inter-${i}`}>
            <div className="bg-blue-50 border border-blue-200 px-3 py-2 rounded-lg flex items-center shadow-sm hover:bg-blue-100 transition duration-300 animate-fadeIn" style={{animationDelay: `${baseElementNodes.length * 100 + 100 + i * 100}ms`}}>
              {getElementImage(node)}
              {node.name}
            </div>
            {i < Math.min(2, intermediateNodes.length - 1) && (
              <div className="flex items-center mx-2 animate-fadeIn" style={{animationDelay: `${baseElementNodes.length * 100 + 150 + i * 100}ms`}}>
                <svg className="w-5 h-5 text-gray-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M12.293 5.293a1 1 0 011.414 0l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-2.293-2.293a1 1 0 010-1.414z" clipRule="evenodd" />
                </svg>
              </div>
            )}
          </React.Fragment>
        ))}
        
        {intermediateNodes.length > 3 && (
          <div className="flex items-center mx-3 px-3 py-1 bg-gray-100 rounded-lg text-gray-700 font-medium animate-fadeIn" style={{animationDelay: `${baseElementNodes.length * 100 + 400}ms`}}>
            <span className="animate-bounce">•••</span>
          </div>
        )}
        
        {tree && (
          <>
            <div className="flex items-center mx-3 animate-fadeIn" style={{animationDelay: `${baseElementNodes.length * 100 + 500}ms`}}>
              <svg className="w-6 h-6 text-green-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M12.293 5.293a1 1 0 011.414 0l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-2.293-2.293a1 1 0 010-1.414z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="bg-green-50 border border-green-200 px-3 py-2 rounded-lg flex items-center shadow-sm hover:bg-green-100 transition duration-300 animate-fadeIn animate-pulse" style={{animationDelay: `${baseElementNodes.length * 100 + 600}ms`}}>
              {getElementImage(tree)}
              <span className="font-medium">{tree.name}</span>
            </div>
          </>
        )}
      </>
    );
  }
};

const renderRecipeStats = (tree: TreeData) => {
  const countUniqueIngredients = (node: TreeData, uniqueIngredients = new Set<string>()): Set<string> => {
    if (node.ingredients?.length > 0) {
      node.ingredients.forEach(ingredient => {
        uniqueIngredients.add(ingredient.name);
        countUniqueIngredients(ingredient, uniqueIngredients);
      });
    }
    return uniqueIngredients;
  };

  const uniqueIngredients = countUniqueIngredients(tree);
  const baseElements = ['Water', 'Fire', 'Earth', 'Air'];
  const baseElementsUsed = new Set([...uniqueIngredients].filter(name => baseElements.includes(name)));
  
  const countSteps = (node: TreeData): number => {
    if (!node.ingredients?.length) return 0;
    return 1 + node.ingredients.reduce((sum, ingredient) => sum + countSteps(ingredient), 0);
  };
  
  const steps = countSteps(tree);
  
  const countDepth = (node: TreeData): number => {
    if (!node.ingredients?.length) return 0;
    return 1 + Math.max(...node.ingredients.map(countDepth));
  };
  
  const depth = countDepth(tree);

  return (
    <>
      <div className="p-3 bg-white rounded-lg border border-gray-200 shadow-sm">
        <div className="text-xs text-gray-500 mb-1">Base Elements</div>
        <div className="text-lg font-bold text-blue-700">{baseElementsUsed.size}</div>
      </div>
      <div className="p-3 bg-white rounded-lg border border-gray-200 shadow-sm">
        <div className="text-xs text-gray-500 mb-1">Total Ingredients</div>
        <div className="text-lg font-bold text-blue-700">{uniqueIngredients.size}</div>
      </div>
      <div className="p-3 bg-white rounded-lg border border-gray-200 shadow-sm">
        <div className="text-xs text-gray-500 mb-1">Recipe Steps</div>
        <div className="text-lg font-bold text-blue-700">{steps}</div>
      </div>
      <div className="p-3 bg-white rounded-lg border border-gray-200 shadow-sm">
        <div className="text-xs text-gray-500 mb-1">Recipe Depth</div>
        <div className="text-lg font-bold text-blue-700">{depth}</div>
      </div>
    </>
  );
};

export default TreeDetails;