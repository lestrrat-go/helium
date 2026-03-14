<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">
<?spec xpath#id-general-comparisons?>
<!-- PURPOSE: Test of comparison between boolean and node-set in backwards compatibility mode -->
<xsl:template match="/">
<out>
  <xsl:variable name="set1" select="/dummy/set1/value"/>
  <xsl:variable name="set2" select="/dummy/set2/value"/>
  <xsl:value-of select="$set1 = true()"/> should be true
  <xsl:value-of select="$set2 = true()"/> should be true
</out>
</xsl:template>
</xsl:stylesheet>
