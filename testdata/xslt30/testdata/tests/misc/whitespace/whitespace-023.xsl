<?xml version="1.0"?> 

<?spec xslt#strip?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:output indent="yes"/>
  <xsl:template match="/"><xsl:copy-of select="doc/foo/text()"/></xsl:template>
</xsl:stylesheet>
