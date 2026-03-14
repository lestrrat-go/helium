<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Tests xsl:sequence instruction containing xsl:fallback. -->

   <xslt:template match="doc">
      <out>
         <xslt:sequence select="'text'">
            <xslt:fallback>
               <xslt:value-of select="'text'"/>
            </xslt:fallback>
         </xslt:sequence>
      </out>
   </xslt:template>
</xslt:transform>
