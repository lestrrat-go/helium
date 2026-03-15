<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#document?>
  <xsl:template match="/">
  <out>
      <H1>Test for document using a fragment identifier().</H1>;
      <P><xsl:copy-of select="document('id010.xml#a')"/></P>;
      <P2><xsl:copy-of select="document('#b',/)"/></P2>;
      <P3><xsl:copy-of select="document('#non-existent',/)"/></P3>;
  </out>
  </xsl:template>
 
</xsl:stylesheet>
