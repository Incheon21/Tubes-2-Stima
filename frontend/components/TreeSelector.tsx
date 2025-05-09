import React from 'react';

interface TreeSelectorProps {
  count: number;
  currentIndex: number;
  setCurrentIndex: (index: number) => void;
}

const TreeSelector: React.FC<TreeSelectorProps> = ({ count, currentIndex, setCurrentIndex }) => {
  return (
    <div className="flex gap-2 overflow-x-auto pb-2">
      {Array.from({ length: count }).map((_, index) => (
        <button
          key={index}
          onClick={() => setCurrentIndex(index)}
          className={`flex items-center justify-center px-4 py-2 rounded-lg transition-all duration-200 whitespace-nowrap
            ${index === currentIndex 
              ? 'bg-gradient-to-r from-blue-500 to-indigo-500 text-white font-medium shadow-md translate-y-[-2px]' 
              : 'bg-white border border-gray-300 hover:bg-gray-100 text-gray-700'
            }`}
        >
          <span className="mr-2">Recipe Tree {index + 1}</span>
          {index === currentIndex && (
            <svg className="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
            </svg>
          )}
        </button>
      ))}
    </div>
  );
};

export default TreeSelector;