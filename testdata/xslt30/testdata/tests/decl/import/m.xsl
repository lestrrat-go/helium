<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<xsl:include href="n.xsl"/>

<xsl:param name="mParam" select="string('!From m!')"/>

<xsl:template match="author">
  m-author:  
  <xsl:value-of select="."/>
</xsl:template>
   
</xsl:stylesheet>
