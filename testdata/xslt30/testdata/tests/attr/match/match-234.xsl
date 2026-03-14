<xsl:stylesheet version="3.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:xf="http://www.w3.org/2005/xpath-functions" 
  exclude-result-prefixes="xs xf">
  
<xsl:strip-space elements="*"/>  

<xsl:variable name="start-node" as="text()?" 
     select="root/p[1]/span[1]/text()"/>

<xsl:variable name="end-node" as="text()?" 
     select="root/p[2]/span[1]/text()"/>

<xsl:template match="* | node()">
  <xsl:copy>
    <xsl:apply-templates select="node()"/>
  </xsl:copy>
</xsl:template>

<xsl:template match="$start-node" priority="5">
  <S/>
  <xsl:next-match/>
</xsl:template>

<xsl:template match="$end-node">
  <xsl:value-of select="."/>
  <E/>
</xsl:template>

</xsl:stylesheet>

