<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- PURPOSE:   Test sequence construction of parentless attributes,
     followed by attachment to a new element -->


<xsl:template match="doc">
  <xsl:variable name="q" as="attribute() *">
    <xsl:attribute name="a">1</xsl:attribute>
    <xsl:attribute name="b">2</xsl:attribute>
    <xsl:attribute name="c">3</xsl:attribute>
    <xsl:attribute name="a">4</xsl:attribute>            
  </xsl:variable>
  <zzz><xsl:copy-of select="$q"/></zzz>
</xsl:template>
 
</xsl:stylesheet>
