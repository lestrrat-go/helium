<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:math="http://exslt.org/math"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="math"
             version="2.0">
<!-- Purpose: Uses function-available to test the existance of an extension function 
  				with a non-zero arity inside a use-when instruction. -->

   <t:template match="doc">
		    <out>
         <t:value-of select="'Available'" use-when="function-available('math:abs', 2)"/>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
