"use client"

import type React from "react"

import { useState } from "react"

function App() {
  const [recipeType, setRecipeType] = useState<"shortest" | "multiple">("shortest")
  const [recipeCount, setRecipeCount] = useState<number>(3)
  const [element, setElement] = useState<string>("")
  const [algorithm, setAlgorithm] = useState<"bfs" | "dfs" | "bidirectional">("bfs")

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    console.log({
      recipeType,
      recipeCount,
      element,
      algorithm,
    })
    // Here you would implement the actual search functionality
  }

  return (
    <div className="min-h-screen bg-[#242424] flex items-start">
      <div className="w-full max-w-[340px] bg-white rounded-lg shadow-lg p-6 space-y-5 m-4">
        <div className="text-center">
          <h1 className="text-3xl font-bold text-purple-700">Little Alchemy 2</h1>
          <p className="text-gray-600 mt-1">Recipe Visualizer</p>
        </div>

        <form onSubmit={handleSearch} className="space-y-4">
          {/* Recipe Type Toggle */}
          <div className="space-y-2">
            <label className="text-sm font-medium text-gray-700 block text-left">Recipe Type</label>
            <div className="flex rounded-md overflow-hidden border border-gray-300">
              <button
                type="button"
                className={`flex-1 py-2 px-4 text-sm font-medium ${
                  recipeType === "shortest" ? "bg-[#1a1a1a] text-white" : "bg-white text-[#646cff]"
                }`}
                onClick={() => setRecipeType("shortest")}
              >
                Shortest Path
              </button>
              <button
                type="button"
                className={`flex-1 py-2 px-4 text-sm font-medium ${
                  recipeType === "multiple" ? "bg-[#1a1a1a] text-white" : "bg-white text-[#646cff]"
                }`}
                onClick={() => setRecipeType("multiple")}
              >
                Multiple Recipes
              </button>
            </div>
          </div>

          {/* Recipe Count Parameter (conditional) */}
          {recipeType === "multiple" && (
            <div className="space-y-2">
              <label htmlFor="recipeCount" className="text-sm font-medium text-gray-700 block text-left">
                Number of Recipes
              </label>
              <input
                type="number"
                id="recipeCount"
                min="1"
                max="10"
                value={recipeCount}
                onChange={(e) => setRecipeCount(Number.parseInt(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-purple-500 focus:border-purple-500"
              />
            </div>
          )}

          {/* Element Input */}
          <div className="space-y-2">
            <label htmlFor="element" className="text-sm font-medium text-gray-700 block text-left">
              Element to Search
            </label>
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  className="h-5 w-5 text-gray-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                  />
                </svg>
              </div>
              <input
                type="text"
                id="element"
                placeholder="e.g. Metal, Life, Human"
                value={element}
                onChange={(e) => setElement(e.target.value)}
                className="w-full pl-10 px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-purple-500 focus:border-purple-500"
                required
              />
            </div>
          </div>

          {/* Algorithm Selection */}
          <div className="space-y-2">
            <label className="text-sm font-medium text-gray-700 block text-left">Search Algorithm</label>
            <div className="grid grid-cols-3 gap-2">
              <button
                type="button"
                className={`py-2 px-3 text-sm font-medium rounded-md border border-gray-300 ${
                  algorithm === "bfs" ? "bg-[#1a1a1a] text-white" : "bg-[#1a1a1a] text-[#646cff]"
                }`}
                onClick={() => setAlgorithm("bfs")}
              >
                BFS
              </button>
              <button
                type="button"
                className={`py-2 px-3 text-sm font-medium rounded-md border border-gray-300 ${
                  algorithm === "dfs" ? "bg-[#1a1a1a] text-white" : "bg-[#1a1a1a] text-[#646cff]"
                }`}
                onClick={() => setAlgorithm("dfs")}
              >
                DFS
              </button>
              <button
                type="button"
                className={`py-2 px-3 text-sm font-medium rounded-md border border-gray-300 ${
                  algorithm === "bidirectional" ? "bg-[#1a1a1a] text-white" : "bg-[#1a1a1a] text-[#646cff]"
                }`}
                onClick={() => setAlgorithm("bidirectional")}
              >
                Bi-Dir
              </button>
            </div>
          </div>

          {/* Submit Button */}
          <button
            type="submit"
            className="w-full flex items-center justify-center py-2 px-4 bg-[#1a1a1a] text-white rounded-md shadow-sm text-sm font-medium hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-500 mt-4"
          >
            Find Recipes
          </button>
        </form>

        <div className="text-center text-sm text-gray-500">
          <p>Find the perfect recipe combinations for any element in Little Alchemy 2</p>
        </div>
      </div>
    </div>
  )
}

export default App
