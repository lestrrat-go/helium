<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="3.0">
   
   <xsl:template name="main">
     <out>
       <xsl:evaluate xpath="'2+2'" use-when="system-property('xsl:supports-dynamic-evaluation')='yes'"/>
       <xsl:value-of select="system-property('xsl:supports-dynamic-evaluation')"/>
     </out>
   </xsl:template>
</xsl:transform>
