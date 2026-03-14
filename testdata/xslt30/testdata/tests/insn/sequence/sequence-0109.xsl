<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test sequence construction using xsl:sequence with node content -->

<xsl:template match="doc">
  <xsl:variable name="q" as="element() *">
     <e/><f/><g/>
  </xsl:variable>
  <xsl:variable name="p" as="element() *">
     <xsl:sequence select="$q"/>
     <xsl:sequence select="$q[3], $q[2], $q[1]"/>
  </xsl:variable>
  <zzz><xsl:copy-of select="$p"/></zzz>
</xsl:template>
 
</xsl:stylesheet>
