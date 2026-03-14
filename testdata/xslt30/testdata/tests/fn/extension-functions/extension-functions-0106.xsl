<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
   extension-element-prefixes="ext" xmlns:ext="http://www.w3.org/xslt30/tests/extension-unknown">
   <!-- Purpose: Fallback for unknown extension instruction.-->

   <xsl:template match="doc">
      <out>
         <ext:special transmogrify="no">
             <xsl:fallback/>
         </ext:special>     
      </out>
   </xsl:template>

</xsl:transform>
