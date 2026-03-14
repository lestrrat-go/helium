<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xpath#id-general-comparisons?>
<!-- PURPOSE: Test of "<=" and ">=" operators -->
<xsl:template match="/">
<out>
  <xsl:variable name="set1" select="/dummy/set1/value"/>
  <xsl:variable name="set2" select="/dummy/set2/value"/>
  <xsl:value-of select="$set1 &gt; $set2"/> should be false
  <xsl:value-of select="$set2 &gt; $set1"/> should be true
</out>
</xsl:template>
</xsl:stylesheet>
