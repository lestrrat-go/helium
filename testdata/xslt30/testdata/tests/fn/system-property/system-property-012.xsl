<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="3.0">

   <xsl:mode streamable="yes" use-when="system-property('xsl:supports-streaming')='yes'"/>
   
   <xsl:template name="main">
     <out><xsl:value-of select="system-property('xsl:supports-streaming')"/></out>
   </xsl:template>
</xsl:transform>
