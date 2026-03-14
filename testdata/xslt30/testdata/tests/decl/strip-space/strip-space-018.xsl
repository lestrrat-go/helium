<?xml version="1.0"?>

<!-- test that the default namespace is NOT used for names in xsl:strip-space -->
<?spec xslt#strip?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:strip-space elements="a b c" xmlns="http://foo.uri/"/>
  <xsl:template match="/"><xsl:copy-of select="."/></xsl:template>
</xsl:stylesheet>
