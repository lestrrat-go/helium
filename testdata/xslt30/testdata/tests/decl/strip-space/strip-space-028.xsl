<?xml version="1.0"?>

<!-- test two conflicting xsl:strip-space declarations (recoverable error) --> 
<?spec xslt#strip?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:strip-space elements="a b Q{}c"/>
  <xsl:preserve-space elements="Q{}b"/>
  <xsl:template match="/"><xsl:copy-of select="."/></xsl:template>
</xsl:stylesheet>
