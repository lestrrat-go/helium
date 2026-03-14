<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
   extension-element-prefixes="xs" xmlns:xs="http://www.w3.org/2001/XMLSchema">
   <!-- Purpose: A reserved namespace must not be used as an extension namespace.-->

   <xsl:template match="doc">
      <out>
         <xs:special transmogrify="no">
            <xsl:fallback/>
         </xs:special>    
      </out>
   </xsl:template>

   <xsl:template match="text()"/>
</xsl:transform>
