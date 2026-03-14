<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:include href="ss5.xsl"/>
<xsl:param name="oParam" select="string('!From ss4!')"/>

<xsl:template match="overview">
  o-overview: 
  <xsl:value-of select="."/>
</xsl:template>
   
</xsl:stylesheet>
