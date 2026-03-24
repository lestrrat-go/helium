<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xpath#id-general-comparisons?>
<!-- PURPOSE: Test of "<=" and ">=" operators -->
<!-- Modified from bool56 which does string comparisons at XPath 2.0 -->
<?same-as-1.0 no?>
<xsl:template match="/">
<out>
  <xsl:variable name="set1" select="for $i in /dummy/set1/value return number($i)"/>
  <xsl:variable name="set2" select="for $i in /dummy/set2/value return number($i)"/>
  <xsl:value-of select="$set1 &gt; $set2"/> should be false
  <xsl:value-of select="$set2 &gt; $set1"/> should be true
</out>
</xsl:template>
</xsl:stylesheet>
