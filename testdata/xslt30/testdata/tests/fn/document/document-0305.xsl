<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with doc($uri) where $uri is a literal 
  				string representing relative URI and resolved 
  				using the base-uri of the stylesheet module.-->

   <t:template match="/">
      <out>
         <t:copy-of select="doc('doc03.xml')//body"/>
      </out>
   </t:template>
</t:transform>
