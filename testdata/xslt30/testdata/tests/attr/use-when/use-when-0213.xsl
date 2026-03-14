<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:fn="http://www.w3.org/2005/xpath-functions"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0"
             exclude-result-prefixes="fn">

<!-- Purpose: Test case in which use-when is used with element-available and function-available functions
  				Where the element and function exist   -->

   <t:template match="/">
	  <out>        
         <t:element name="E1" use-when="element-available('t:value-of')">
	           <t:text>Element defined</t:text>
	     </t:element>        
         <E2 t:use-when="function-available('fn:doc')">
            <t:text>function IS defined</t:text>
         </E2>        
      </out>
   </t:template>
</t:transform>
