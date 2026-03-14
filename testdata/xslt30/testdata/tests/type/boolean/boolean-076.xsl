<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xpath#id-node-comparisons?>
<!-- PURPOSE: Test of node order operators -->
<?same-as-1.0 no?>
<xsl:template match="/">
<out>
  <xsl:variable name="set1" select="/dummy/set1/value[1]"/>
  <xsl:variable name="set2" select="/dummy/set2/value[1]"/>
  <xsl:variable name="set3" select="/dummy/set1/*[1]"/>
  <xsl:value-of select="$set1 &lt;&lt; $set2"/> should be true
  <xsl:value-of select="$set1 &lt;&lt; $set3"/> should be false
  <xsl:value-of select="$set1 >> $set2"/> should be false
  <xsl:value-of select="$set1 >> $set3"/> should be false
</out>
</xsl:template>
</xsl:stylesheet>
