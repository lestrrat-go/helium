<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-id?>
  <!-- test second parameter to id() function -->
  <xsl:template match="/">
    <xsl:variable name="temp"><a><b idref="b"/><c/></a></xsl:variable>
    <out><xsl:copy-of select="$temp/a/b/id(@idref, current()//dummy)"/></out>
  </xsl:template>

</xsl:stylesheet>
