<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:fn="http://www.w3.org/2005/xpath-functions"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="fn"
             version="2.0">
<!-- Purpose: Test that function document() can be called both as document($uri) 
  				and fn:document($uri) and the result is the same. 'fn' is bound to 
  				the Xpath Functions namespace. -->

   <t:template match="/doc">
      <output xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="document('doc02.xml') is fn:document('doc02.xml')"/>
      </output>
   </t:template>
</t:transform>
