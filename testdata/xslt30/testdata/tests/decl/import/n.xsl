<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<xsl:import href="o.xsl"/>

<xsl:param name="nParam" select="string('!From n!')"/>

<xsl:template match="chapter[@num='1']">
  n-contents of first chapter:  
  <xsl:value-of select="."/>
</xsl:template>
   
</xsl:stylesheet>
