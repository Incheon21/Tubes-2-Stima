import React, { useRef, useEffect } from 'react';

interface TreeSelectorProps {
  count: number;
  currentIndex: number;
  setCurrentIndex: (index: number) => void;
  treeNames?: string[]; // Optional array of custom names for trees
}

const TreeSelector: React.FC<TreeSelectorProps> = ({ 
  count, 
  currentIndex, 
  setCurrentIndex, 
  treeNames 
}) => {
  const scrollRef = useRef<HTMLDivElement>(null);
  
  // Scroll selected button into view when currentIndex changes
  useEffect(() => {
    if (scrollRef.current) {
      const selectedButton = scrollRef.current.querySelector(`[data-index="${currentIndex}"]`);
      if (selectedButton) {
        selectedButton.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'center' });
      }
    }
  }, [currentIndex]);
  
  // Create pagination dots for mobile
  const renderPaginationDots = () => {
    return (
      <div className="flex justify-center gap-1.5 mt-2 md:hidden">
        {Array.from({ length: count }).map((_, index) => (
          <button
            key={`dot-${index}`}
            onClick={() => setCurrentIndex(index)}
            aria-label={`View Recipe Tree ${index + 1}`}
            className={`w-2 h-2 rounded-full transition-all duration-300 ${
              index === currentIndex 
                ? 'bg-blue-500 scale-125' 
                : 'bg-gray-300 hover:bg-gray-400'
            }`}
          />
        ))}
      </div>
    );
  };

  return (
    <div className="space-y-2">
      {count > 1 && (
        <div className="flex items-center mb-2">
          <div className="h-0.5 flex-1 bg-gradient-to-r from-transparent to-gray-200"></div>
          <h3 className="px-3 text-sm font-medium text-gray-600">Alternate Recipe Paths</h3>
          <div className="h-0.5 flex-1 bg-gradient-to-l from-transparent to-gray-200"></div>
        </div>
      )}
      
      <div 
        ref={scrollRef}
        className="flex gap-2 overflow-x-auto pb-2 scrollbar-thin scrollbar-thumb-gray-300 scrollbar-track-transparent"
      >
        {Array.from({ length: count }).map((_, index) => (
          <button
            key={index}
            data-index={index}
            onClick={() => setCurrentIndex(index)}
            className={`flex items-center justify-center px-4 py-2 rounded-lg transition-all duration-300 whitespace-nowrap
              ${index === currentIndex 
                ? 'bg-gradient-to-r from-blue-500 to-indigo-600 text-white font-medium shadow-md translate-y-[-2px] ring-2 ring-blue-300' 
                : 'bg-white border border-gray-300 hover:bg-gray-50 hover:border-gray-400 hover:translate-y-[-1px] text-gray-700 hover:shadow-sm'
              }`}
          >
            <svg 
              className={`w-4 h-4 mr-2 transition-transform duration-300 ${index === currentIndex ? 'text-blue-200' : 'text-blue-400'}`}
              xmlns="http://www.w3.org/2000/svg" 
              viewBox="0 0 20 20" 
              fill="currentColor"
              style={{ transform: index === currentIndex ? 'scale(1.2)' : 'scale(1)' }}
            >
              <path d="M9 2a1 1 0 000 2h2a1 1 0 100-2H9z" />
              <path fillRule="evenodd" d="M4 5a2 2 0 012-2 3 3 0 003 3h2a3 3 0 003-3 2 2 0 012 2v11a2 2 0 01-2 2H6a2 2 0 01-2-2V5zm3 4a1 1 0 000 2h.01a1 1 0 100-2H7zm3 0a1 1 0 000 2h3a1 1 0 100-2h-3zm-3 4a1 1 0 100 2h.01a1 1 0 100-2H7zm3 0a1 1 0 100 2h3a1 1 0 100-2h-3z" clipRule="evenodd" />
            </svg>
            
            <span>
              {treeNames && treeNames[index] 
                ? treeNames[index] 
                : `Recipe Path ${index + 1}`
              }
            </span>
            
            {index === currentIndex && (
              <svg 
                className="w-4 h-4 ml-2 text-blue-200 animate-pulse" 
                xmlns="http://www.w3.org/2000/svg" 
                viewBox="0 0 20 20" 
                fill="currentColor"
              >
                <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
              </svg>
            )}
          </button>
        ))}
      </div>
      
      {count > 3 && renderPaginationDots()}
      
      {count > 1 && (
        <div className="flex justify-between text-xs text-gray-500 px-1 mt-1">
          <span>
            Showing recipe path {currentIndex + 1} of {count}
          </span>
          
          <div className="flex gap-1">
            <button
              onClick={() => setCurrentIndex(Math.max(0, currentIndex - 1))}
              disabled={currentIndex === 0}
              className="p-1 rounded hover:bg-gray-100 disabled:opacity-50 disabled:hover:bg-transparent"
              aria-label="Previous recipe path"
            >
              <svg className="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z" clipRule="evenodd" />
              </svg>
            </button>
            <button
              onClick={() => setCurrentIndex(Math.min(count - 1, currentIndex + 1))}
              disabled={currentIndex === count - 1}
              className="p-1 rounded hover:bg-gray-100 disabled:opacity-50 disabled:hover:bg-transparent"
              aria-label="Next recipe path"
            >
              <svg className="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clipRule="evenodd" />
              </svg>
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default TreeSelector;