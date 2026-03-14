<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:include href="ss3.xsl"/>
<xsl:param name="mParam" select="string('!From ss2!')"/>

<xsl:template match="author">
  m-author:  
  <xsl:value-of select="."/>
</xsl:template>
   
</xsl:stylesheet>
